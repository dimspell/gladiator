package p2p

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
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

func (m mockSession) GetUserID() int64 {
	return m.ID
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
		Connected:  nil,
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
	ctx := t.Context()
	logger.SetPlainTextLogger(os.Stdout, slog.LevelDebug)

	// Channels for synchronization
	offerReceived := make(chan struct{}, 1)
	answerReceived := make(chan struct{}, 1)

	chanOffer := make(chan wire.Offer, 1)
	chanAnswer := make(chan wire.Offer, 1)

	// For player1 (host)
	host := &Peer{UserID: 1}
	hostManager := &mockPeerManager{
		host:  host,
		peers: map[int64]*Peer{},
	}
	hostHandler := &PeerToPeerMessageHandler{
		peerManager:    hostManager,
		newTCPRedirect: redirect.NewNoop,
		newUDPRedirect: redirect.NewNoop,
	}
	hostSession := &mockSession{ID: 1,
		onSendRTCOffer: func(offer wire.Offer) {
			chanOffer <- offer
			close(chanOffer) // Close channel after sending offer
		},
	}
	hostHandler.session = hostSession

	// For player 2 (guest)
	guestManager := &mockPeerManager{}
	first, _ := guestManager.CreatePeer(wire.Player{UserID: 1})
	second, _ := guestManager.CreatePeer(wire.Player{UserID: 2})
	guestManager.host = first
	guestManager.peers = map[int64]*Peer{
		1: first,
		2: second,
	}

	guestHandler := &PeerToPeerMessageHandler{
		peerManager:    guestManager,
		newTCPRedirect: redirect.NewNoop,
		newUDPRedirect: redirect.NewNoop,
	}
	guestSession := &mockSession{
		ID: 2,
		onSendRTCAnswer: func(offer wire.Offer) {
			chanAnswer <- offer
			close(chanAnswer) // Close channel after sending answer
		},
	}
	guestHandler.session = guestSession

	hostSession.onSendRTCICECandidate = func(ice webrtc.ICECandidateInit, fromUserID int64) {
		waitToReceive(answerReceived) // Wait until answer is received before handling ICE

		if err := guestHandler.handleRTCCandidate(ctx, ice, fromUserID); err != nil {
			log.Printf("AddICECandidate returned error: %v", err)
		}
	}
	guestSession.onSendRTCICECandidate = func(ice webrtc.ICECandidateInit, fromUserID int64) {
		waitToReceive(offerReceived) // Wait until offer is received before handling ICE

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
	offerReceived <- struct{}{}
	close(offerReceived)

	// Send RTC Answer and handle it (host handles RTC Answer)
	answer := waitToReceive(chanAnswer)
	if err := hostHandler.handleRTCAnswer(ctx, answer, guest.UserID); err != nil {
		log.Printf("handleRTCAnswer: %v", err)
		return
	}
	answerReceived <- struct{}{}
	close(answerReceived)

	fmt.Println("End")
	time.Sleep(time.Second * 3)
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
			session:        &mockSession{ID: 2},
			peerManager:    peerManager,
			newTCPRedirect: redirect.NewNoop,
			newUDPRedirect: redirect.NewNoop,
		}
		if err := h.handleHostMigration(t.Context(), newHostPlayer); err != nil {
			t.Error(err)
		}
		if peerManager.host.UserID != 3 {
			t.Error("host not migrated")
		}
	})
}
