package p2p

import (
	"context"
	"fmt"
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

func TestPeerToPeerMessageHandler_Handle(t *testing.T) {
	logger.SetPlainTextLogger(os.Stdout, slog.LevelDebug)

	// Channels for synchronization
	offerSent := make(chan struct{}, 1)
	answerSent := make(chan struct{}, 1)
	offerReceived := make(chan struct{}, 1)
	answerReceived := make(chan struct{}, 1)
	defer close(offerSent)
	defer close(answerSent)
	defer close(offerReceived)
	defer close(answerReceived)

	// For player1 (host)
	host := &Peer{UserID: 1}
	hostManager := &mockPeerManager{
		host: host,
		peers: map[int64]*Peer{
			// 1: host,
		},
	}

	chanOffer := make(chan wire.Offer, 1)
	hostHandler := &PeerToPeerMessageHandler{
		peerManager:    hostManager,
		newTCPRedirect: redirect.NewNoop,
		newUDPRedirect: redirect.NewNoop,
	}
	hostHandler.session = &mockSession{ID: 1,
		onSendRTCOffer: func(offer wire.Offer) {
			chanOffer <- offer
			close(chanOffer) // Close channel after sending offer
			offerSent <- struct{}{}
		},
		onSendRTCICECandidate: func(ice webrtc.ICECandidateInit, fromUserID int64) {
			<-answerReceived // Wait until answer is received before handling ICE

			if err := hostHandler.handleRTCCandidate(t.Context(), ice, fromUserID); err != nil {
				t.Errorf("AddICECandidate returned error: %v", err)
			}
		},
	}

	// For player 2 (guest)
	guestManager := &mockPeerManager{}
	// guestManager.peers:
	first, _ := guestManager.CreatePeer(wire.Player{UserID: 1})
	second, _ := guestManager.CreatePeer(wire.Player{UserID: 2})
	guestManager.host = first
	guestManager.peers = map[int64]*Peer{
		1: first,
		2: second,
	}

	chanAnswer := make(chan wire.Offer, 1)
	guestHandler := &PeerToPeerMessageHandler{
		peerManager:    guestManager,
		newTCPRedirect: redirect.NewNoop,
		newUDPRedirect: redirect.NewNoop,
	}
	guestHandler.session = &mockSession{
		ID: 2,
		onSendRTCAnswer: func(offer wire.Offer) {
			chanAnswer <- offer
			close(chanAnswer) // Close channel after sending answer
			answerSent <- struct{}{}
		},
		onSendRTCICECandidate: func(ice webrtc.ICECandidateInit, fromUserID int64) {
			<-offerReceived // Wait until offer is received before handling ICE

			if err := guestHandler.handleRTCCandidate(t.Context(), ice, fromUserID); err != nil {
				t.Errorf("handleRTCCandidate returned error: %v", err)
			}
		},
	}

	// New player is joining (host handles join)
	guest := wire.Player{UserID: 2}
	if err := hostHandler.handleJoinRoom(t.Context(), guest); err != nil {
		t.Errorf("handleJoinRoom returned error: %v", err)
		return
	}

	// Send RTC Offer and handle it (guest handles RTC Offer)
	select {
	case offer := <-chanOffer:
		offerReceived <- struct{}{}
		if err := guestHandler.handleRTCOffer(t.Context(), offer, offer.Player.UserID); err != nil {
			t.Errorf("handleRTCOffer: %v", err)
			return
		}
	case <-time.After(time.Second * 1):
		t.Errorf("timed out waiting for RTC Offer")
	}

	// Send RTC Answer and handle it (host handles RTC Answer)
	select {
	case answer := <-chanAnswer:
		answerReceived <- struct{}{}
		if err := hostHandler.handleRTCAnswer(t.Context(), answer, answer.Player.UserID); err != nil {
			t.Errorf("handleRTCAnswer: %v", err)
			return
		}
	case <-time.After(time.Second * 1):
		t.Errorf("timed out waiting for RTC Answer")
	}

	fmt.Println("End")
	// time.Sleep(time.Millisecond * 1000)
	t.Log(hostHandler.peerManager)
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
