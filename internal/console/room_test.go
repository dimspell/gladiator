package console

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/stretchr/testify/require"
)

// --- Mock UserSession with Send ---
type mockSession struct {
	*UserSession
	sendFunc func(ctx context.Context, payload []byte)
}

func (m *mockSession) Send(ctx context.Context, payload []byte) {
	if m.sendFunc != nil {
		m.sendFunc(ctx, payload)
	}
}

type mockWsConn struct {
	writeFunc func(ctx context.Context, messageType websocket.MessageType, payload []byte) error
}

func (m *mockWsConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return websocket.MessageText, []byte{}, nil
}
func (m *mockWsConn) Write(ctx context.Context, messageType websocket.MessageType, payload []byte) error {
	if m.writeFunc != nil {
		return m.writeFunc(ctx, messageType, payload)
	}
	return nil
}
func (m *mockWsConn) CloseNow() error { return nil }

func newTestSession(id int64, sendFunc func(ctx context.Context, payload []byte)) *UserSession {
	return &UserSession{
		UserID:    id,
		User:      wire.User{UserID: id, Username: "user"},
		Character: wire.Character{CharacterID: id, ClassType: 1},
		WebSocket: &mockWsConn{
			writeFunc: func(ctx context.Context, messageType websocket.MessageType, payload []byte) error {
				if sendFunc != nil {
					sendFunc(ctx, payload)
				}
				return nil
			},
		},
	}
}

func TestAddGetDeleteUserSession(t *testing.T) {
	mp := NewRoomService()
	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)

	got, ok := mp.GetUserSession(sess.UserID)
	require.True(t, ok)
	require.Equal(t, sess, got)

	mp.DeleteUserSession(sess.UserID)
	_, ok = mp.GetUserSession(sess.UserID)
	require.False(t, ok)
}

// TestRoomService_Reset_NoRace verifies that Reset() does not race with
// concurrent readers of rooms/sessions maps. It starts Run, spawns readers,
// cancels the context to trigger Reset, and relies on -race to flag any
// unsynchronized access. Regression guard for the pre-existing race in Reset().
func TestRoomService_Reset_NoRace(t *testing.T) {
	mp := NewRoomService()

	// Populate some sessions and rooms.
	for i := int64(1); i <= 10; i++ {
		mp.AddUserSession(i, newTestSession(i, nil))
		_, _ = mp.CreateRoom(i, fmt.Sprintf("room-%d", i), "", 0, "127.0.0.1")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go mp.Run(ctx)

	var wg sync.WaitGroup

	// Reader 1: repeatedly list rooms
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_ = mp.ListRooms()
		}
	}()

	// Reader 2: repeatedly look up sessions
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := int64(1); ; i = (i % 10) + 1 {
			select {
			case <-ctx.Done():
				return
			default:
			}
			mp.GetUserSession(i)
		}
	}()

	// Reader 3: repeatedly get known rooms
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; ; i = (i % 10) + 1 {
			select {
			case <-ctx.Done():
				return
			default:
			}
			mp.GetRoom(fmt.Sprintf("room-%d", i))
		}
	}()

	// Let readers warm up, then cancel to trigger Reset.
	time.Sleep(5 * time.Millisecond)
	cancel()
	wg.Wait()
}

func TestCreateRoomAndJoinRoom(t *testing.T) {
	mp := NewRoomService()
	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)

	room, err := mp.CreateRoom(sess.UserID, "room1", "", 0, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, "room1", room.ID)

	// Join with another user
	sess2 := newTestSession(2, nil)
	mp.AddUserSession(sess2.UserID, sess2)
	joinedRoom, err := mp.JoinRoom("room1", sess2.UserID, "127.0.0.2")
	require.NoError(t, err)
	require.Equal(t, 2, len(joinedRoom.Players))
}

func TestLeaveRoomAndHostMigration(t *testing.T) {
	mp := NewRoomService()
	sess1 := newTestSession(1, nil)
	sess2 := newTestSession(2, nil)
	mp.AddUserSession(sess1.UserID, sess1)
	mp.AddUserSession(sess2.UserID, sess2)
	room, _ := mp.CreateRoom(sess1.UserID, "room1", "", 0, "127.0.0.1")
	_, _ = mp.JoinRoom("room1", sess2.UserID, "127.0.0.2")

	// Host leaves, guest should become host
	mp.LeaveRoom(context.Background(), sess1)
	roomAfter, _ := mp.GetRoom("room1")
	require.Equal(t, sess2.UserID, roomAfter.HostPlayer.UserID)
	require.Equal(t, room.ID, roomAfter.ID)
}

func TestGetNextHost(t *testing.T) {
	mp := NewRoomService()
	sess1 := newTestSession(1, nil)
	sess2 := newTestSession(2, nil)
	sess1.JoinedAt = time.Now().Add(-time.Minute)
	sess2.JoinedAt = time.Now()
	room := &GameRoom{Players: map[int64]*UserSession{1: sess1, 2: sess2}}
	host := mp.GetNextHost(room)
	require.Equal(t, sess1, host)
}

func TestSetRoomReady(t *testing.T) {
	mp := NewRoomService()
	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)
	room, _ := mp.CreateRoom(sess.UserID, "room1", "", 0, "127.0.0.1")
	msg := wire.Message{Content: "room1"}
	mp.SetRoomReady(msg)
	require.True(t, room.Ready)
}

func TestJoinRoomErrors(t *testing.T) {
	mp := NewRoomService()
	_, err := mp.JoinRoom("room1", 1, "127.0.0.1")
	require.Error(t, err, "should error if user or room missing")

	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)
	_, err = mp.CreateRoom(sess.UserID, "room1", "", 0, "127.0.0.1")
	require.NoError(t, err)
	_, err = mp.JoinRoom("room1", 2, "127.0.0.2")
	require.Error(t, err, "should error if user missing")
	mp.AddUserSession(2, newTestSession(2, nil))
	_, err = mp.JoinRoom("room1", 1, "127.0.0.1")
	require.Error(t, err, "should error if already joined")
}

func TestDestroyRoom(t *testing.T) {
	mp := NewRoomService()
	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)
	room, _ := mp.CreateRoom(sess.UserID, "room1", "", 0, "127.0.0.1")
	mp.DestroyRoom("room1")
	_, found := mp.GetRoom("room1")
	require.False(t, found)
	require.NotNil(t, room)
}

func TestBroadcastMessage(t *testing.T) {
	mp := NewRoomService()
	var sent []int64
	mockSess := &mockSession{newTestSession(1, func(ctx context.Context, payload []byte) { sent = append(sent, 1) }), nil}
	mp.AddUserSession(1, mockSess.UserSession)
	mockSess = &mockSession{newTestSession(2, func(ctx context.Context, payload []byte) { sent = append(sent, 2) }), nil}
	mp.AddUserSession(2, mockSess.UserSession)
	mockSess = &mockSession{newTestSession(3, func(ctx context.Context, payload []byte) { sent = append(sent, 3) }), nil}
	mp.AddUserSession(3, mockSess.UserSession)
	mp.BroadcastMessage(context.Background(), []byte("hi"))
	require.ElementsMatch(t, []int64{1, 2, 3}, sent)
}

func TestAnnounceJoin(t *testing.T) {
	mp := NewRoomService()
	var sentTo []int64
	mockSess := &mockSession{newTestSession(1, func(ctx context.Context, payload []byte) { sentTo = append(sentTo, 1) }), nil}
	mp.AddUserSession(1, mockSess.UserSession)
	mockSess = &mockSession{newTestSession(2, func(ctx context.Context, payload []byte) { sentTo = append(sentTo, 2) }), nil}
	mp.AddUserSession(2, mockSess.UserSession)
	mockSess = &mockSession{newTestSession(3, func(ctx context.Context, payload []byte) { sentTo = append(sentTo, 3) }), nil}
	mp.AddUserSession(3, mockSess.UserSession)
	room, _ := mp.CreateRoom(1, "room1", "", 0, "127.0.0.1")
	room.Players[2] = mp.sessions[2]
	room.Players[3] = mp.sessions[3]
	mp.AnnounceJoin(*room, 2)
	// Should send to 1 and 3, not 2
	require.ElementsMatch(t, []int64{1, 3}, sentTo)
}

func TestListRoomsAndGetRoom(t *testing.T) {
	mp := NewRoomService()
	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)
	_, _ = mp.CreateRoom(sess.UserID, "room1", "", 0, "127.0.0.1")
	rooms := mp.ListRooms()
	require.Contains(t, rooms, "room1")
	got, found := mp.GetRoom("room1")
	require.True(t, found)
	require.Equal(t, "room1", got.ID)
}

func TestSetPlayerConnectedDisconnected(t *testing.T) {
	mp := NewRoomService()
	sess := newTestSession(1, nil)

	// SetPlayerConnected should add the user and send them a LobbyUsers message
	mp.SetPlayerConnected(sess)

	// Verify session was added
	_, ok := mp.GetUserSession(sess.UserID)
	require.True(t, ok)

	// SetPlayerDisconnected should remove the session
	mp.SetPlayerDisconnected(sess)
	_, ok = mp.GetUserSession(sess.UserID)
	require.False(t, ok)
}

func TestForEachSessionAndListSessions(t *testing.T) {
	mp := NewRoomService()
	for i := int64(1); i <= 2; i++ {
		mp.AddUserSession(i, newTestSession(i, nil))
	}
	var ids []int64
	mp.forEachSession(func(s *UserSession) bool { ids = append(ids, s.UserID); return true })
	require.ElementsMatch(t, []int64{1, 2}, ids)
	players := mp.listSessions()
	require.Len(t, players, 2)
}

func TestResetClearsSessionsAndRooms(t *testing.T) {
	mp := NewRoomService()
	mp.AddUserSession(1, newTestSession(1, nil))
	mp.Rooms["room1"] = &GameRoom{ID: "room1", Players: map[int64]*UserSession{1: mp.sessions[1]}}
	mp.Reset()
	require.Empty(t, mp.sessions)
	require.Empty(t, mp.Rooms)
}

func TestRegisterRelayHooks(t *testing.T) {
	mp := NewRoomService()
	relay := &RelayServer{}
	mp.RegisterRelayHooks(relay)
	require.NotNil(t, relay.OnJoin)
	require.NotNil(t, relay.OnLeave)
	require.NotNil(t, relay.OnDelete)
}

func TestHandleRelayLeaveRemovesUser(t *testing.T) {
	mp := NewRoomService()
	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)
	room, _ := mp.CreateRoom(sess.UserID, "room1", "", 0, "127.0.0.1")
	mp.HandleRelayLeave("leave", "1", "room1")
	_, found := room.Players[1]
	require.False(t, found)
}

// TestSendWriteErrorTearsDownSession verifies that a failed websocket write
// triggers the OnWriteError callback (wired like SetPlayerConnected does) and
// that the dead session is removed from the session map. Regression test for
// the silent-drop TODO in UserSession.Send.
func TestSendWriteErrorTearsDownSession(t *testing.T) {
	mp := NewRoomService()
	sess := newTestSession(1, nil)
	// Mirror SetPlayerConnected's wiring of the teardown callback.
	sess.OnWriteError = func() { mp.SetPlayerDisconnected(sess) }
	mp.AddUserSession(sess.UserID, sess)

	// Make the socket write fail.
	sess.WebSocket = &mockWsConn{
		writeFunc: func(ctx context.Context, messageType websocket.MessageType, payload []byte) error {
			return fmt.Errorf("connection reset")
		},
	}

	sess.Send(context.Background(), []byte{byte(wire.LobbyUsers), 1, 2, 3})

	// The teardown runs in a goroutine; wait for it to complete.
	require.Eventually(t, func() bool {
		_, ok := mp.GetUserSession(1)
		return !ok
	}, time.Second, 10*time.Millisecond, "dead session should be removed after a write error")

	// And the session must not be re-added.
	_, ok := mp.GetUserSession(1)
	require.False(t, ok, "dead session should be removed after a write error")
	require.Nil(t, sess.WebSocket, "WebSocket should be nil after teardown")
}

// TestSendWriteErrorFiresOnce ensures concurrent failed sends trigger the
// teardown callback exactly once (guarded by the atomic disconnecting flag).
func TestSendWriteErrorFiresOnce(t *testing.T) {
	var calls atomic.Int64
	sess := newTestSession(1, nil)
	sess.OnWriteError = func() { calls.Add(1) }
	sess.WebSocket = &mockWsConn{
		writeFunc: func(ctx context.Context, messageType websocket.MessageType, payload []byte) error {
			return fmt.Errorf("boom")
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess.Send(context.Background(), []byte{byte(wire.LobbyUsers), 1})
		}()
	}
	wg.Wait()

	require.Eventually(t, func() bool {
		return calls.Load() == 1
	}, time.Second, 10*time.Millisecond, "OnWriteError must fire exactly once")
}
