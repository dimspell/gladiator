package backend

import (
	"net"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/stretchr/testify/assert"
)

func TestCreateGameRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 28, // Command code
		18, 0, // Packet length
		1, 0, 0, 0, // State
		3, 0, 0, 0, // Map ID
		114, 111, 111, 109, 0, // Game room name
		0, // Password
	}

	// Act
	req := CreateGameRequest(packet[4:])
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), data.State)
	assert.Equal(t, uint32(3), data.MapID)
	assert.Equal(t, "room", data.RoomName)
	assert.Equal(t, "", data.Password)
}

func TestBackend_HandleCreateGame(t *testing.T) {
	b := &Backend{
		Proxy: NewLAN("127.0.0.1"),
		gameClient: &mockGameClient{
			CreateGameResponse: connect.NewResponse(&v1.CreateGameResponse{
				Game: &v1.Game{
					GameId:        "gameId",
					Name:          "room",
					Password:      "",
					HostIpAddress: "127.0.0.1",
					MapId:         3,
				},
			}),
		}}
	conn := &mockConn{RemoteAddress: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12137}}
	session := &Session{ID: "TEST",
		Conn:     conn,
		UserID:   2137,
		Username: "JP",
	}

	// State = 0
	assert.NoError(t, b.HandleCreateGame(session, CreateGameRequest{
		0, 0, 0, 0, // State
		3, 0, 0, 0, // Map ID
		114, 111, 111, 109, 0, // Game room name
		0, // Password
	}))
	assert.Equal(t, []byte{255, 28, 8, 0}, conn.Written[0:4]) // Header
	assert.Equal(t, []byte{1, 0, 0, 0}, conn.Written[4:8])    // Next state
	assert.Len(t, conn.Written, 8)

	conn.Written = nil

	// State = 1
	assert.NoError(t, b.HandleCreateGame(session, CreateGameRequest{
		1, 0, 0, 0, // State
		3, 0, 0, 0, // Map ID
		114, 111, 111, 109, 0, // Game room name
		0, // Password
	}))
	assert.Equal(t, []byte{255, 28, 8, 0}, conn.Written[0:4]) // Header
	assert.Equal(t, []byte{2, 0, 0, 0}, conn.Written[4:8])    // Next state
	assert.Len(t, conn.Written, 8)
}
