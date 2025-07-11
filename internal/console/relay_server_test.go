package console

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/assert"
)

type MockStream struct {
	Writer bytes.Buffer
	Reader bytes.Buffer
}

func (ms *MockStream) CancelWrite(code quic.StreamErrorCode) {}
func (ms *MockStream) CancelRead(code quic.StreamErrorCode)  {}

func (ms *MockStream) Write(p []byte) (int, error) {
	return ms.Writer.Write(p)
}
func (ms *MockStream) Read(p []byte) (int, error) {
	return ms.Reader.Read(p)
}

type MockConn struct{}

func (mc *MockConn) AcceptStream(ctx context.Context) (*quic.Stream, error) {
	// TODO implement me
	panic("implement me")
}

func (mc *MockConn) RemoteAddr() net.Addr {
	return &net.UDPAddr{}
}

func (mc *MockConn) CloseWithError(code quic.ApplicationErrorCode, msg string) error {
	return nil
}

func TestRelayServer_LeaveRoom_RemovesPeerAndRoom(t *testing.T) {
	rs := &RelayServer{
		rooms:         make(map[string]*Room),
		peerToRoomIDs: map[string]string{"peer1": "room1"},
		Events:        make(chan RelayEvent, 2),
		logger:        logger.NewDiscardLogger(),
	}

	mockStream := &MockStream{}
	mockConn := &MockConn{}

	rs.rooms["room1"] = &Room{
		ID: "room1",
		Peers: map[string]*PeerConn{
			"peer1": {
				ID:     "peer1",
				Conn:   mockConn,
				Stream: mockStream,
			},
		},
	}

	rs.leaveRoom("peer1", "room1")

	_, exists := rs.peerToRoomIDs["peer1"]
	assert.False(t, exists)

	_, ok := rs.rooms["room1"]
	assert.False(t, ok, "Room should be deleted")

	var events []RelayEvent
	for i := 0; i < 2; i++ {
		select {
		case ev := <-rs.Events:
			events = append(events, ev)
		case <-time.After(time.Second):
			t.Fatal("expected event")
		}
	}

	assert.ElementsMatch(t, []string{"leave", "delete"}, []string{events[0].Type, events[1].Type})
}

func TestRelayServer_JoinRoom_NewRoom(t *testing.T) {
	rs := &RelayServer{
		rooms:         make(map[string]*Room),
		peerToRoomIDs: make(map[string]string),
		Events:        make(chan RelayEvent, 1),
		logger:        logger.NewDiscardLogger(),
	}

	mockStream := &MockStream{}
	mockConn := &MockConn{}

	pc := rs.joinRoom("room1", "peer1", mockConn, mockStream)

	assert.Equal(t, "peer1", pc.ID)
	assert.Contains(t, rs.rooms["room1"].Peers, "peer1")
	assert.Equal(t, "room1", rs.peerToRoomIDs["peer1"])

	select {
	case ev := <-rs.Events:
		assert.Equal(t, "join", ev.Type)
		assert.Equal(t, "peer1", ev.PeerID)
		assert.Equal(t, "room1", ev.RoomID)
	case <-time.After(time.Second):
		t.Fatal("expected join event")
	}
}
