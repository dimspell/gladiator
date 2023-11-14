package backend

import (
	"net"
	"testing"

	"github.com/dispel-re/dispel-multi/model"
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
	db := testDB(t)
	b := &Backend{DB: db}
	conn := &mockConn{RemoteAddress: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12137}}
	session := &model.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

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
