package relay

import (
	"context"
	"net"
	"testing"
	"time"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logger.SetDiscardLogger()
}

// --- Utility function tests ---

func TestRemoteID(t *testing.T) {
	assert.Equal(t, "123", remoteID(123))
	assert.Equal(t, "0", remoteID(0))
	assert.Equal(t, "9999999", remoteID(9999999))
}

// --- ProxyRelay Factory tests ---

func TestProxyRelay_Mode(t *testing.T) {
	p := &ProxyRelay{}
	assert.Equal(t, model.RunModeRelay, p.Mode())
}

func TestProxyRelay_Create(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 123,
	}

	proxyConfig := &ProxyRelay{
		RelayServerAddr: "localhost:9999",
		IPPrefix:        net.IPv4(127, 0, 1, 0),
	}

	client := newMockGameServiceClient()
	proxyClient := proxyConfig.Create(session, client)

	assert.NotNil(t, proxyClient)
	relay, ok := proxyClient.(*Relay)
	require.True(t, ok)
	assert.Equal(t, "123", relay.router.SelfID())
	assert.NotNil(t, relay.router.Transport())
}

// --- Relay tests ---

func TestNewRelay(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 456,
	}

	config := &ProxyRelay{
		RelayServerAddr: "relay.example.com:8080",
		IPPrefix:        net.IPv4(127, 0, 2, 0),
	}

	client := newMockGameServiceClient()
	relay := NewRelay(config, client, session)

	assert.NotNil(t, relay)
	assert.Equal(t, session, relay.session)
	assert.NotNil(t, relay.router)
	assert.Equal(t, "456", relay.router.SelfID())
	assert.NotNil(t, relay.router.Transport())
}

func TestNewRelay_DefaultIPPrefix(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 789,
	}

	config := &ProxyRelay{
		RelayServerAddr: "localhost:9999",
		// No IPPrefix set
	}

	client := newMockGameServiceClient()
	relay := NewRelay(config, client, session)

	assert.NotNil(t, relay.router.Manager())
}

func TestRelay_Close(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)
	relay.router.SetRoomID("test-room")
	relay.router.SetCurrentHostID("123")

	relay.Close()

	assert.Empty(t, relay.router.RoomID())
	assert.Empty(t, relay.router.CurrentHostID())
}

func TestRelay_Close_Idempotent(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)

	// Should not panic
	relay.Close()
	relay.Close()
	relay.Close()
}

// --- PacketRouter tests ---

func TestPacketRouter_Reset(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)
	relay.router.SetRoomID("test-room")
	relay.router.SetCurrentHostID("123")

	relay.router.Reset()

	assert.Empty(t, relay.router.RoomID())
	assert.Empty(t, relay.router.CurrentHostID())
}

// --- Handle tests ---

func TestPacketRouter_Handle_UnknownEventType(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)

	// Unknown event type should not error
	err := relay.Handle(context.Background(), []byte{0xFF})
	assert.NoError(t, err)
}

func TestPacketRouter_HandleLeaveRoom_SelfIgnored(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)

	// Create leave room message for self
	msg := wire.Message{
		Type: wire.LeaveRoom,
		Content: wire.Player{
			UserID: 100, // Same as session
		},
	}
	payload := wire.Compose(wire.LeaveRoom, msg)

	err := relay.Handle(context.Background(), payload)
	assert.NoError(t, err)
}

func TestPacketRouter_HandleLeaveRoom_OtherPeer(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)

	// Assign IP to peer so it exists in manager
	ip, err := relay.router.Manager().AssignIP("200")
	require.NoError(t, err)

	// Verify IP was assigned
	_, exists := relay.router.Manager().GetPeerIP("200")
	require.True(t, exists, "IP should be assigned")

	// Create leave room message for other peer
	msg := wire.Message{
		Type: wire.LeaveRoom,
		Content: wire.Player{
			UserID: 200,
		},
	}
	payload := wire.Compose(wire.LeaveRoom, msg)

	err = relay.Handle(context.Background(), payload)
	assert.NoError(t, err)

	// RemoveByRemoteID is called, but since there's no host started,
	// only the PeerHosts entry would be removed (which doesn't exist)
	// The PeerIPs entry remains - this is expected behavior
	_, stillExists := relay.router.Manager().GetPeerIP("200")
	assert.True(t, stillExists, "IP remains if no host was started")
	_ = ip
}

func TestPacketRouter_HandleHostMigration_NonSelf_NonBlocking(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)
	relay.router.SetCurrentHostID("100")
	relay.router.SetRoomID("test-room")

	// New host is 200 (not us)
	msg := wire.Message{
		Type: wire.HostMigration,
		Content: wire.Player{
			UserID: 200,
		},
	}
	payload := wire.Compose(wire.HostMigration, msg)

	start := time.Now()
	err := relay.Handle(context.Background(), payload)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, elapsed, 100*time.Millisecond,
		"Handle should not block for 3s when host migration is for another peer")

	// Verify currentHostID was still updated synchronously
	assert.Equal(t, "200", relay.router.CurrentHostID())
}

func TestPacketRouter_HandleHostMigration(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
		Conn:   &mockConn{}, // Need a conn for SendToGame
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)
	relay.router.SetCurrentHostID("100")
	relay.router.SetRoomID("test-room")

	// New host is 200 (not us, so we shouldn't send HostMigration packet)
	msg := wire.Message{
		Type: wire.HostMigration,
		Content: wire.Player{
			UserID: 200,
		},
	}
	payload := wire.Compose(wire.HostMigration, msg)

	err := relay.Handle(context.Background(), payload)
	assert.NoError(t, err)

	assert.Equal(t, "200", relay.router.CurrentHostID())
}

func TestPacketRouter_HandleJoinRoom(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)

	// Create join room message
	msg := wire.Message{
		Type: wire.JoinRoom,
		Content: wire.Player{
			UserID:   200,
			Username: "guest",
		},
	}
	payload := wire.Compose(wire.JoinRoom, msg)

	// Should not error (handleJoinRoom is currently a no-op)
	err := relay.Handle(context.Background(), payload)
	assert.NoError(t, err)
}

// --- Error path tests ---

func TestRelay_CreateRoom_InvalidRelayAddr(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "invalid:9999"}, newMockGameServiceClient(), session)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := relay.CreateRoom(ctx, proxy.CreateParams{GameID: "test-room"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed connect to the relay server")
}

func TestRelay_ListGames(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	client := &mockGameServiceClientWithGames{
		games: []*multiv1.Game{
			{Name: "game1", Password: ""},
			{Name: "game2", Password: "secret"},
		},
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, client, session)

	games, err := relay.ListGames(context.Background())
	require.NoError(t, err)
	assert.Len(t, games, 2)
	assert.Equal(t, "game1", games[0].Name)
	assert.Equal(t, "game2", games[1].Name)
}

// --- Message callbacks ---

func TestPacketRouter_OnTCPMessage_NoStream(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)
	relay.router.SetRoomID("test-room")

	handler := relay.router.OnTCPMessage("test-room", "200")

	// Without a stream, should return an error
	err := handler([]byte("test"))
	assert.Error(t, err, "expected error when stream is nil")
	assert.Contains(t, err.Error(), "stream is nil")
}

func TestPacketRouter_OnUDPMessage_NoStream(t *testing.T) {
	session := &bsession.Session{
		ID:     "test-session",
		UserID: 100,
	}

	relay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), session)
	relay.router.SetRoomID("test-room")

	handler := relay.router.OnUDPMessage("test-room", "200")

	// Without a stream, should return an error
	err := handler([]byte("test"))
	assert.Error(t, err, "expected error when stream is nil")
	assert.Contains(t, err.Error(), "stream is nil")
}

// --- Mock implementations ---

type mockConn struct {
	written []byte
}

func (m *mockConn) Read(b []byte) (n int, err error) { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error) {
	m.written = append(m.written, b...)
	return len(b), nil
}
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

type mockGameServiceClientWithGames struct {
	games []*multiv1.Game
}

func (m *mockGameServiceClientWithGames) CreateGame(ctx context.Context, req *connect.Request[multiv1.CreateGameRequest]) (*connect.Response[multiv1.CreateGameResponse], error) {
	return connect.NewResponse(&multiv1.CreateGameResponse{}), nil
}

func (m *mockGameServiceClientWithGames) JoinGame(ctx context.Context, req *connect.Request[multiv1.JoinGameRequest]) (*connect.Response[multiv1.JoinGameResponse], error) {
	return connect.NewResponse(&multiv1.JoinGameResponse{}), nil
}

func (m *mockGameServiceClientWithGames) ListGames(ctx context.Context, req *connect.Request[multiv1.ListGamesRequest]) (*connect.Response[multiv1.ListGamesResponse], error) {
	return connect.NewResponse(&multiv1.ListGamesResponse{
		Games: m.games,
	}), nil
}

func (m *mockGameServiceClientWithGames) GetGame(ctx context.Context, req *connect.Request[multiv1.GetGameRequest]) (*connect.Response[multiv1.GetGameResponse], error) {
	return connect.NewResponse(&multiv1.GetGameResponse{}), nil
}
