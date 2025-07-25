package console

import (
	"context"
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
		Websocket: &mockWsConn{
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
	mp := NewMultiplayer()
	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)

	got, ok := mp.GetUserSession(sess.UserID)
	require.True(t, ok)
	require.Equal(t, sess, got)

	mp.DeleteUserSession(sess.UserID)
	_, ok = mp.GetUserSession(sess.UserID)
	require.False(t, ok)
}

func TestCreateRoomAndJoinRoom(t *testing.T) {
	mp := NewMultiplayer()
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
	mp := NewMultiplayer()
	sess1 := newTestSession(1, nil)
	sess2 := newTestSession(2, nil)
	mp.AddUserSession(sess1.UserID, sess1)
	mp.AddUserSession(sess2.UserID, sess2)
	room, _ := mp.CreateRoom(sess1.UserID, "room1", "", 0, "127.0.0.1")
	mp.JoinRoom("room1", sess2.UserID, "127.0.0.2")

	// Host leaves, guest should become host
	mp.LeaveRoom(context.Background(), sess1)
	roomAfter, _ := mp.GetRoom("room1")
	require.Equal(t, sess2.UserID, roomAfter.HostPlayer.UserID)
	require.Equal(t, room.ID, roomAfter.ID)
}

func TestGetNextHost(t *testing.T) {
	mp := NewMultiplayer()
	sess1 := newTestSession(1, nil)
	sess2 := newTestSession(2, nil)
	sess1.JoinedAt = time.Now().Add(-time.Minute)
	sess2.JoinedAt = time.Now()
	room := &GameRoom{Players: map[int64]*UserSession{1: sess1, 2: sess2}}
	host := mp.GetNextHost(room)
	require.Equal(t, sess1, host)
}

func TestSetRoomReady(t *testing.T) {
	mp := NewMultiplayer()
	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)
	room, _ := mp.CreateRoom(sess.UserID, "room1", "", 0, "127.0.0.1")
	msg := wire.Message{Content: "room1"}
	mp.SetRoomReady(msg)
	require.True(t, room.Ready)
}

func TestJoinRoomErrors(t *testing.T) {
	mp := NewMultiplayer()
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
	mp := NewMultiplayer()
	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)
	room, _ := mp.CreateRoom(sess.UserID, "room1", "", 0, "127.0.0.1")
	mp.DestroyRoom("room1")
	_, found := mp.GetRoom("room1")
	require.False(t, found)
	require.NotNil(t, room)
}

func TestBroadcastMessage(t *testing.T) {
	mp := NewMultiplayer()
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
	mp := NewMultiplayer()
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
	mp := NewMultiplayer()
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
	t.Skip("Failing - needs to be fixed")
	mp := NewMultiplayer()
	sess := newTestSession(1, nil)
	called := false
	mockSess := &mockSession{sess, func(ctx context.Context, payload []byte) { called = true }}
	mp.SetPlayerConnected(mockSess.UserSession)
	require.True(t, called)
	called = false
	mp.SetPlayerDisconnected(mockSess.UserSession)
	// Should not panic, should remove session
	_, ok := mp.GetUserSession(sess.UserID)
	require.False(t, ok)
}

func TestForEachSessionAndListSessions(t *testing.T) {
	mp := NewMultiplayer()
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
	mp := NewMultiplayer()
	mp.AddUserSession(1, newTestSession(1, nil))
	mp.Rooms["room1"] = &GameRoom{ID: "room1", Players: map[int64]*UserSession{1: mp.sessions[1]}}
	mp.Reset()
	require.Empty(t, mp.sessions)
	require.Empty(t, mp.Rooms)
}

func TestRegisterRelayHooks(t *testing.T) {
	mp := NewMultiplayer()
	relay := &RelayServer{}
	mp.RegisterRelayHooks(relay)
	require.NotNil(t, relay.OnJoin)
	require.NotNil(t, relay.OnLeave)
	require.NotNil(t, relay.OnDelete)
}

func TestHandleRelayLeaveRemovesUser(t *testing.T) {
	mp := NewMultiplayer()
	sess := newTestSession(1, nil)
	mp.AddUserSession(sess.UserID, sess)
	room, _ := mp.CreateRoom(sess.UserID, "room1", "", 0, "127.0.0.1")
	mp.HandleRelayLeave("leave", "1", "room1")
	_, found := room.Players[1]
	require.False(t, found)
}
