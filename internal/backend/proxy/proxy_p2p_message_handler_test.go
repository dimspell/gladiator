package proxy

import (
	"context"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
	"testing"
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
	peers map[string]*Peer
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
	//TODO implement me
	panic("implement me")
}

func (m *mockPeerManager) Host() (*Peer, bool) {
	return m.host, true
}

func (m *mockPeerManager) SetHost(host *Peer, newHost wire.Player) {
	m.host = host
}

func TestPeerToPeerMessageHandler_handleHostMigration(t *testing.T) {
	newRedir := redirect.NewNoop

	//t.Run("I am a host, switching to new host", func(t *testing.T) {})
	t.Run("I am a guest and host left, other become host", func(t *testing.T) {
		player2 := &Peer{
			UserID:     "2", // host
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
			host: player2,
			peers: map[string]*Peer{
				"2": player2,
				"3": player3,
			},
		}
		h := &PeerToPeerMessageHandler{
			session:        &mockSession{ID: "1"},
			peerManager:    peerManager,
			newTCPRedirect: newRedir,
			newUDPRedirect: newRedir,
		}
		if err := h.handleHostMigration(context.TODO(), wire.Player{UserID: 3}); err != nil {
			t.Error(err)
		}
		if peerManager.host.UserID != "3" {
			t.Error("host not migrated")
		}
	})
}
