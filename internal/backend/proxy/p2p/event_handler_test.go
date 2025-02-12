package p2p

import (
	"context"
	"strconv"
	"testing"

	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
)

type mockSession struct {
	ID string
}

func (m mockSession) GetUserID() string {
	return m.ID
}

func (m mockSession) SendRTCICECandidate(_ context.Context, _ webrtc.ICECandidateInit, _ string) error {
	return nil
}

func (m mockSession) SendRTCOffer(_ context.Context, _ webrtc.SessionDescription, _ string) error {
	return nil
}

func (m mockSession) SendRTCAnswer(_ context.Context, _ webrtc.SessionDescription, _ string) error {
	return nil
}

type mockPeerManager struct {
	host  *Peer
	peers map[int]*Peer
}

func (m *mockPeerManager) AddPeer(peer *Peer) {
	m.peers[peer.UserID] = peer
}

func (m *mockPeerManager) GetPeer(peerId string) (*Peer, bool) {
	p, ok := m.peers[peerId]
	return p, ok
}

func (m *mockPeerManager) RemovePeer(peerId string) {
	delete(m.peers, peerId)
}

func (m *mockPeerManager) CreatePeer(player wire.Player) (*Peer, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, err
	}
	return &Peer{
		UserID:     strconv.Itoa(int(player.UserID)),
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

func TestPeerToPeerMessageHandler_handleJoinRoom(t *testing.T) {
	t.Run("I am host", func(t *testing.T) {
		peerManager := &mockPeerManager{
			// host: player2,
			peers: map[string]*Peer{},
		}
		h := &PeerToPeerMessageHandler{
			session:        &mockSession{ID: "1"},
			peerManager:    peerManager,
			newTCPRedirect: redirect.NewNoop,
			newUDPRedirect: redirect.NewNoop,
		}
		other := wire.Player{UserID: 2}
		if err := h.handleJoinRoom(context.TODO(), other); err != nil {
			t.Errorf("handleJoinRoom returned error: %v", err)
			return
		}
		assert.Len(t, peerManager.peers, 1)
	})
	t.Run("I am guest", func(t *testing.T) {
		host := &Peer{UserID: "1"}
		peerManager := &mockPeerManager{
			host: host,
			peers: map[string]*Peer{
				"1": host,
			},
		}
		h := &PeerToPeerMessageHandler{
			session:        &mockSession{ID: "2"},
			peerManager:    peerManager,
			newTCPRedirect: redirect.NewNoop,
			newUDPRedirect: redirect.NewNoop,
		}
		other := wire.Player{UserID: 3}
		if err := h.handleJoinRoom(context.TODO(), other); err != nil {
			t.Errorf("handleJoinRoom returned error: %v", err)
			return
		}
		assert.Len(t, peerManager.peers, 2)
	})
}

func TestPeerToPeerMessageHandler_handleRTCOffer(t *testing.T) {
	t.Run("I am host", func(t *testing.T) { t.Log("Not implemented") })
	t.Run("I am guest", func(t *testing.T) { t.Log("Not implemented") })
}

func TestPeerToPeerMessageHandler_handleRTCAnswer(t *testing.T) {
	t.Run("I am host", func(t *testing.T) { t.Log("Not implemented") })
	t.Run("I am guest", func(t *testing.T) { t.Log("Not implemented") })
}

func TestPeerToPeerMessageHandler_handleRTCCandidate(t *testing.T) {
	t.Run("I am host", func(t *testing.T) { t.Log("Not implemented") })
	t.Run("I am guest", func(t *testing.T) { t.Log("Not implemented") })
}

func TestPeerToPeerMessageHandler_handleLeave(t *testing.T) {
	t.Run("I am host", func(t *testing.T) { t.Log("Not implemented") })
	t.Run("I am guest", func(t *testing.T) { t.Log("Not implemented") })
}

func TestPeerToPeerMessageHandler_handleHostMigration(t *testing.T) {
	t.Run("I am a host, switching to new host", func(t *testing.T) {})

	t.Run("Host left, I am a guest, I will become new host", func(t *testing.T) {})

	t.Run("Host left, I am a guest, other become host", func(t *testing.T) {
		player1 := &Peer{
			UserID:     "1", // host
			Addr:       nil,
			Mode:       0,
			Connection: nil,
			Connected:  nil,
			PipeTCP:    nil,
			PipeUDP:    nil,
		}
		player3 := &Peer{
			UserID:     "3", // to-be-host
			Addr:       nil,
			Mode:       0,
			Connection: nil,
			Connected:  nil,
			PipeTCP:    nil,
			PipeUDP:    nil,
		}

		peerManager := &mockPeerManager{
			host: player1,
			peers: map[string]*Peer{
				"1": player1,
				"3": player3,
			},
		}
		h := &PeerToPeerMessageHandler{
			session:        &mockSession{ID: "2"},
			peerManager:    peerManager,
			newTCPRedirect: redirect.NewNoop,
			newUDPRedirect: redirect.NewNoop,
		}
		if err := h.handleHostMigration(context.TODO(), wire.Player{UserID: 3}); err != nil {
			t.Error(err)
		}
		if peerManager.host.UserID != "3" {
			t.Error("host not migrated")
		}
	})
}
