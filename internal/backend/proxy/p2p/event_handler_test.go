package p2p

import (
	"context"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
)

type mockSession struct {
	ID int64

	onSendRTCICECandidate func(webrtc.ICECandidateInit, int64)
	onSendRTCOffer        func(wire.Offer)
	onSendRTCAnswer       func(wire.Offer)
}

func (m mockSession) SendRTCICECandidate(_ context.Context, candidate webrtc.ICECandidateInit, fromUserID int64) error {
	if m.onSendRTCICECandidate != nil {
		m.onSendRTCICECandidate(candidate, fromUserID)
	}
	return nil
}

func (m mockSession) SendRTCOffer(_ context.Context, sdpOffer webrtc.SessionDescription, fromUserID int64) error {
	if m.onSendRTCOffer != nil {
		m.onSendRTCOffer(wire.Offer{
			Player: wire.Player{UserID: fromUserID},
			Offer:  sdpOffer,
		})
	}
	return nil
}

func (m mockSession) SendRTCAnswer(_ context.Context, sdpAnswer webrtc.SessionDescription, fromUserID int64) error {
	if m.onSendRTCAnswer != nil {
		m.onSendRTCAnswer(wire.Offer{
			Player: wire.Player{UserID: fromUserID},
			Offer:  sdpAnswer,
		})
	}
	return nil
}

type mockPeerManager struct {
	host  *Peer
	peers map[int64]*Peer
}

func (m *mockPeerManager) AddPeer(peer *Peer) {
	m.peers[peer.UserID] = peer
}

func (m *mockPeerManager) GetPeer(peerId int64) (*Peer, bool) {
	p, ok := m.peers[peerId]
	return p, ok
}

func (m *mockPeerManager) RemovePeer(peerId int64) {
	delete(m.peers, peerId)
}

func (m *mockPeerManager) CreatePeer(player wire.Player) (*Peer, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, err
	}
	return &Peer{
		UserID:     player.UserID,
		Addr:       nil,
		Mode:       0,
		Connection: peerConnection,
		Connected:  make(chan struct{}),
		PipeTCP:    nil,
		PipeUDP:    nil,
	}, nil
}

func (m *mockPeerManager) Host() (*Peer, bool) {
	return m.host, true
}

func (m *mockPeerManager) SetHost(host *Peer, newHost wire.Player) {
	m.host = host
}

func waitToReceive[T any](ch chan T) T {
	select {
	case res := <-ch: // Wait until offer is received before handling ICE
		return res
	case <-time.After(time.Second * 3):
		break
	}
	var zero T
	return zero
}

func TestPeerToPeerMessageHandler_Handle(t *testing.T) {
	logger.SetPlainTextLogger(os.Stdout, slog.LevelDebug)

	t.Run("The first player is joining the solo host", func(t *testing.T) {
		ctx := t.Context()

		// Channels for synchronization
		chanOffer := make(chan wire.Offer, 1)
		chanAnswer := make(chan wire.Offer, 1)

		// For player1 (host)
		host := &Peer{UserID: 1, Connected: make(chan struct{})}
		hostManager := &mockPeerManager{
			host:  host,
			peers: map[int64]*Peer{},
		}
		hostSession := &mockSession{ID: 1,
			onSendRTCOffer: func(offer wire.Offer) {
				chanOffer <- offer
				close(chanOffer) // Close channel after sending offer
			},
		}
		hostHandler := &PeerToPeerMessageHandler{
			UserID:         1,
			peerManager:    hostManager,
			newTCPRedirect: redirect.NewNoop,
			newUDPRedirect: redirect.NewNoop,
			session:        hostSession,
			logger:         slog.With("user", "host"),
		}

		// For player 2 (guest)
		guestManager := &mockPeerManager{}
		first, _ := guestManager.CreatePeer(wire.Player{UserID: 1})
		second, _ := guestManager.CreatePeer(wire.Player{UserID: 2})
		guestManager.host = first
		guestManager.peers = map[int64]*Peer{
			1: first,
			2: second,
		}

		guestSession := &mockSession{
			ID: 2,
			onSendRTCAnswer: func(offer wire.Offer) {
				chanAnswer <- offer
				close(chanAnswer) // Close channel after sending answer
			},
		}
		guestHandler := &PeerToPeerMessageHandler{
			UserID:         2,
			peerManager:    guestManager,
			newTCPRedirect: redirect.NewNoop,
			newUDPRedirect: redirect.NewNoop,
			session:        guestSession,
			logger:         slog.With("user", "guest"),
		}

		hostSession.onSendRTCICECandidate = func(ice webrtc.ICECandidateInit, fromUserID int64) {
			if err := guestHandler.handleRTCCandidate(ctx, ice, fromUserID); err != nil {
				log.Printf("AddICECandidate returned error: %v", err)
			}
		}
		guestSession.onSendRTCICECandidate = func(ice webrtc.ICECandidateInit, fromUserID int64) {
			if err := hostHandler.handleRTCCandidate(ctx, ice, fromUserID); err != nil {
				log.Printf("handleRTCCandidate returned error: %v", err)
			}
		}

		// New player is joining (host handles join)
		guest := wire.Player{UserID: 2}
		if err := hostHandler.handleJoinRoom(ctx, guest); err != nil {
			t.Errorf("handleJoinRoom returned error: %v", err)
			return
		}

		// Send RTC Offer and handle it (guest handles RTC Offer)
		offer := waitToReceive(chanOffer)
		if err := guestHandler.handleRTCOffer(ctx, offer, host.UserID); err != nil {
			log.Printf("handleRTCOffer: %v", err)
			return
		}

		// Send RTC Answer and handle it (host handles RTC Answer)
		answer := waitToReceive(chanAnswer)
		if err := hostHandler.handleRTCAnswer(ctx, answer, guest.UserID); err != nil {
			log.Printf("handleRTCAnswer: %v", err)
			return
		}

		select {
		case <-second.Connected:
			t.Logf("guest established connection with the host")
			// return
		case <-time.After(time.Second * 3):
			t.Errorf("timed out waiting to connect")
		}

		var wg sync.WaitGroup
		wg.Add(2)

		second.Connection.OnDataChannel(func(dc *webrtc.DataChannel) {
			wg.Done()
		})
		wg.Wait()
	})
}

func TestPeerToPeerMessageHandler_handleLeave(t *testing.T) {
	t.Run("Peer leaves room", func(t *testing.T) {
		peerManager := &mockPeerManager{
			peers: map[int64]*Peer{
				2: {UserID: 2},
			},
		}
		h := &PeerToPeerMessageHandler{
			session:     &mockSession{ID: 1},
			peerManager: peerManager,
			logger:      slog.Default(),
		}

		leavingPlayer := wire.Player{UserID: 2}
		assert.NoError(t, h.handleLeaveRoom(t.Context(), leavingPlayer))
		_, ok := peerManager.peers[leavingPlayer.UserID]
		assert.False(t, ok, "Peer should be removed from peerManager")
	})
}

func TestPeerToPeerMessageHandler_handleHostMigration(t *testing.T) {
	t.Run("I am a host, switching to new host", func(t *testing.T) {})

	t.Run("Host left, I am a guest, I will become new host", func(t *testing.T) {})

	t.Run("Host left, I am a guest, other become host", func(t *testing.T) {
		player1 := &Peer{
			UserID:     1, // host
			Addr:       nil,
			Mode:       0,
			Connection: nil,
			Connected:  nil,
			PipeTCP:    nil,
			PipeUDP:    nil,
		}
		player3 := &Peer{
			UserID:     3, // to-be-host
			Addr:       nil,
			Mode:       0,
			Connection: nil,
			Connected:  nil,
			PipeTCP:    nil,
			PipeUDP:    nil,
		}
		newHostPlayer := wire.Player{UserID: 3}

		peerManager := &mockPeerManager{
			host: player1,
			peers: map[int64]*Peer{
				1: player1,
				3: player3,
			},
		}
		h := &PeerToPeerMessageHandler{
			UserID:         2,
			session:        &mockSession{ID: 2},
			peerManager:    peerManager,
			newTCPRedirect: redirect.NewNoop,
			newUDPRedirect: redirect.NewNoop,
			logger:         slog.Default(),
		}
		if err := h.handleHostMigration(t.Context(), newHostPlayer); err != nil {
			t.Error(err)
		}
		if peerManager.host.UserID != 3 {
			t.Error("host not migrated")
		}
	})
}
