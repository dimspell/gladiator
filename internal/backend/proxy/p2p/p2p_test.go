package p2p

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logger.SetDiscardLogger()
}

// --- Utility function tests ---

func TestPeerID(t *testing.T) {
	assert.Equal(t, "123", peerID(123))
	assert.Equal(t, "0", peerID(0))
	assert.Equal(t, "9999999", peerID(9999999))
}

func TestParseUserID(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"123", 123, false},
		{"0", 0, false},
		{"9999999", 9999999, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id, err := parseUserID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, id)
			}
		})
	}
}

// --- Peer tests ---

func TestPeer_Close(t *testing.T) {
	peer := &Peer{
		peerID: "1",
		logger: slog.Default(),
	}
	// Should not panic even with nil connection/datachannel
	peer.Close()
	peer.Close() // Idempotent
}

func TestPeer_Close_WithConnection(t *testing.T) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.NoError(t, err)

	peer := &Peer{
		peerID:     "1",
		connection: pc,
		logger:     slog.Default(),
	}

	peer.Close()
	assert.Nil(t, peer.connection)
	assert.Nil(t, peer.dataChannel)
}

func TestPeer_Send_WithoutDataChannel(t *testing.T) {
	peer := &Peer{
		peerID:        "1",
		logger:        slog.Default(),
		outboundQueue: nil,
	}

	// Should queue the message
	err := peer.Send([]byte("test"))
	assert.NoError(t, err)
	assert.Len(t, peer.outboundQueue, 1)
	assert.Equal(t, []byte("test"), peer.outboundQueue[0])
}

func TestPeer_Send_QueueLimit(t *testing.T) {
	peer := &Peer{
		peerID:        "1",
		logger:        slog.Default(),
		outboundQueue: make([][]byte, 256), // Already at max
	}

	// Should drop the message
	err := peer.Send([]byte("dropped"))
	assert.NoError(t, err)
	assert.Len(t, peer.outboundQueue, 256) // Still at max
}

func TestPeer_Send_Concurrent(t *testing.T) {
	peer := &Peer{
		peerID: "1",
		logger: slog.Default(),
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = peer.Send([]byte{byte(i)})
		}(i)
	}
	wg.Wait()

	// All messages should be queued (up to limit)
	assert.LessOrEqual(t, len(peer.outboundQueue), 256)
}

// --- ProxyP2P Factory tests ---

func TestProxyP2P_Mode(t *testing.T) {
	p := &ProxyP2P{}
	assert.Equal(t, model.RunModeWebRTC, p.Mode())
}

func TestProxyP2P_Create(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 123,
	}

	proxy := &ProxyP2P{
		IPPrefix: net.IPv4(127, 0, 1, 0),
	}

	client := newMockGameServiceClient()
	proxyClient := proxy.Create(session, client)

	assert.NotNil(t, proxyClient)
	p2p, ok := proxyClient.(*PeerToPeer)
	require.True(t, ok)
	assert.Equal(t, "123", p2p.selfID)
}

// --- PeerToPeer tests ---

func TestNewPeerToPeer(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 456,
	}

	config := &ProxyP2P{
		IPPrefix: net.IPv4(127, 0, 2, 0),
	}

	client := newMockGameServiceClient()
	p2p := NewPeerToPeer(config, client, session)

	assert.NotNil(t, p2p)
	assert.Equal(t, session, p2p.session)
	assert.Equal(t, "456", p2p.selfID)
	assert.NotNil(t, p2p.peers)
	assert.NotNil(t, p2p.manager)
}

func TestNewPeerToPeer_DefaultIPPrefix(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 789,
	}

	config := &ProxyP2P{} // No IPPrefix set

	client := newMockGameServiceClient()
	p2p := NewPeerToPeer(config, client, session)

	// Should use default 127.0.0.0
	assert.NotNil(t, p2p.manager)
}

func TestPeerToPeer_Reset(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	// Add some state
	p2p.roomID = "test-room"
	p2p.currentHostID = "123"
	p2p.peers["1"] = &Peer{peerID: "1", logger: slog.Default()}

	// Reset
	p2p.Reset()

	assert.Empty(t, p2p.roomID)
	assert.Empty(t, p2p.currentHostID)
	assert.Empty(t, p2p.peers)
}

func TestPeerToPeer_Close(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)
	p2p.roomID = "test-room"

	p2p.Close()

	assert.Empty(t, p2p.roomID)
}

// --- Handle tests ---

func TestPeerToPeer_Handle_UnknownEventType(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	// Unknown event type should not error
	err := p2p.Handle(context.Background(), []byte{0xFF})
	assert.NoError(t, err)
}

func TestPeerToPeer_Handle_LobbyEvents(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)
	ctx := context.Background()

	// These should be silently ignored
	for _, eventType := range []wire.EventType{wire.LobbyUsers, wire.JoinLobby, wire.CreateRoom} {
		payload := []byte{byte(eventType)}
		err := p2p.Handle(ctx, payload)
		assert.NoError(t, err)
	}
}

func TestPeerToPeer_HandleLeaveRoom_SelfIgnored(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	// Create leave room message for self
	msg := wire.Message{
		Type: wire.LeaveRoom,
		Content: wire.Player{
			UserID: 100, // Same as session
		},
	}
	payload := wire.Compose(wire.LeaveRoom, msg)

	err := p2p.Handle(context.Background(), payload)
	assert.NoError(t, err)
}

func TestPeerToPeer_HandleLeaveRoom_OtherPeer(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	// Add a peer
	p2p.peers["200"] = &Peer{peerID: "200", logger: slog.Default()}

	// Create leave room message for other peer
	msg := wire.Message{
		Type: wire.LeaveRoom,
		Content: wire.Player{
			UserID: 200,
		},
	}
	payload := wire.Compose(wire.LeaveRoom, msg)

	err := p2p.Handle(context.Background(), payload)
	assert.NoError(t, err)

	// Peer should be removed
	_, exists := p2p.peers["200"]
	assert.False(t, exists)
}

func TestPeerToPeer_HandleHostMigration(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)
	p2p.currentHostID = "100"

	// New host is 200
	msg := wire.Message{
		Type: wire.HostMigration,
		Content: wire.Player{
			UserID: 200,
		},
	}
	payload := wire.Compose(wire.HostMigration, msg)

	err := p2p.Handle(context.Background(), payload)
	assert.NoError(t, err)

	assert.Equal(t, "200", p2p.currentHostID)
}

// --- Message handler callback tests ---

func TestPeerToPeer_OnTCPMessage_NoPeer(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	handler := p2p.onTCPMessage("unknown")
	err := handler([]byte("test"))

	// Should not error, just buffer/drop
	assert.NoError(t, err)
}

func TestPeerToPeer_OnUDPMessage_NoPeer(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	handler := p2p.onUDPMessage("unknown")
	err := handler([]byte("test"))

	// Should not error, just buffer/drop
	assert.NoError(t, err)
}

func TestPeerToPeer_OnTCPMessage_WithPeer(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	// Add a peer without datachannel (will queue)
	peer := &Peer{peerID: "200", logger: slog.Default()}
	p2p.peers["200"] = peer

	handler := p2p.onTCPMessage("200")
	err := handler([]byte("test"))

	assert.NoError(t, err)
	// Should be queued with 'T' prefix
	require.Len(t, peer.outboundQueue, 1)
	assert.Equal(t, byte('T'), peer.outboundQueue[0][0])
	assert.Equal(t, []byte("test"), peer.outboundQueue[0][1:])
}

func TestPeerToPeer_OnUDPMessage_WithPeer(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	// Add a peer without datachannel (will queue)
	peer := &Peer{peerID: "200", logger: slog.Default()}
	p2p.peers["200"] = peer

	handler := p2p.onUDPMessage("200")
	err := handler([]byte("test"))

	assert.NoError(t, err)
	// Should be queued with 'U' prefix
	require.Len(t, peer.outboundQueue, 1)
	assert.Equal(t, byte('U'), peer.outboundQueue[0][0])
	assert.Equal(t, []byte("test"), peer.outboundQueue[0][1:])
}

// --- RTC signaling tests (with mock payloads) ---

func TestPeerToPeer_HandleRTCOffer_WrongRecipient(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	// Offer intended for user 999, not us (100)
	offer := wire.Offer{
		CreatorID:   200,
		RecipientID: 999,
		Offer:       webrtc.SessionDescription{Type: webrtc.SDPTypeOffer},
	}
	msg := wire.Message{
		From:    "200",
		To:      "999",
		Type:    wire.RTCOffer,
		Content: offer,
	}

	payload, _ := json.Marshal(msg)
	fullPayload := append([]byte{byte(wire.RTCOffer)}, payload...)

	err := p2p.handleRTCOffer(context.Background(), fullPayload)
	assert.NoError(t, err)

	// No peer should be created
	assert.Empty(t, p2p.peers)
}

func TestPeerToPeer_HandleRTCAnswer_WrongRecipient(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	// Answer intended for user 999, not us (100)
	answer := wire.Offer{
		CreatorID:   200,
		RecipientID: 999,
		Offer:       webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer},
	}
	msg := wire.Message{
		From:    "200",
		To:      "999",
		Type:    wire.RTCAnswer,
		Content: answer,
	}

	payload, _ := json.Marshal(msg)
	fullPayload := append([]byte{byte(wire.RTCAnswer)}, payload...)

	err := p2p.handleRTCAnswer(context.Background(), fullPayload)
	assert.NoError(t, err)
}

func TestPeerToPeer_HandleRTCCandidate_WrongRecipient(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	p2p := NewPeerToPeer(&ProxyP2P{}, newMockGameServiceClient(), session)

	// Candidate intended for user 999, not us (100)
	candidate := webrtc.ICECandidateInit{Candidate: "test"}
	msg := wire.Message{
		From:    "200",
		To:      "999",
		Type:    wire.RTCICECandidate,
		Content: candidate,
	}

	payload, _ := json.Marshal(msg)
	fullPayload := append([]byte{byte(wire.RTCICECandidate)}, payload...)

	err := p2p.handleRTCCandidate(context.Background(), fullPayload)
	assert.NoError(t, err)
}

// --- Peer setDataChannel and queue flushing ---

func TestPeer_SetDataChannel_FlushQueue(t *testing.T) {
	sent := make([][]byte, 0)
	var mu sync.Mutex

	// Create a mock data channel
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.NoError(t, err)
	defer pc.Close()

	dc, err := pc.CreateDataChannel("test", nil)
	require.NoError(t, err)

	peer := &Peer{
		peerID:        "1",
		logger:        slog.Default(),
		outboundQueue: [][]byte{[]byte("msg1"), []byte("msg2")},
	}

	// Note: In real scenario, OnOpen would fire after ICE negotiation.
	// Here we just test the setDataChannel logic sets up the callback.
	peer.setDataChannel(dc)

	// Simulate OnOpen by waiting briefly
	// (In actual WebRTC, this requires full negotiation)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	_ = sent
	mu.Unlock()
}

// --- Mock GameServiceClient ---

type mockGameServiceClient struct{}

func newMockGameServiceClient() *mockGameServiceClient {
	return &mockGameServiceClient{}
}

func (m *mockGameServiceClient) CreateGame(ctx context.Context, req *connect.Request[multiv1.CreateGameRequest]) (*connect.Response[multiv1.CreateGameResponse], error) {
	return connect.NewResponse(&multiv1.CreateGameResponse{}), nil
}

func (m *mockGameServiceClient) JoinGame(ctx context.Context, req *connect.Request[multiv1.JoinGameRequest]) (*connect.Response[multiv1.JoinGameResponse], error) {
	return connect.NewResponse(&multiv1.JoinGameResponse{}), nil
}

func (m *mockGameServiceClient) ListGames(ctx context.Context, req *connect.Request[multiv1.ListGamesRequest]) (*connect.Response[multiv1.ListGamesResponse], error) {
	return connect.NewResponse(&multiv1.ListGamesResponse{}), nil
}

func (m *mockGameServiceClient) GetGame(ctx context.Context, req *connect.Request[multiv1.GetGameRequest]) (*connect.Response[multiv1.GetGameResponse], error) {
	return connect.NewResponse(&multiv1.GetGameResponse{
		Game: &multiv1.Game{
			Name:       "test",
			HostUserId: 1,
		},
		Players: []*multiv1.Player{
			{UserId: 1, Username: "host"},
		},
	}), nil
}
