package backend

import (
	"testing"

	"github.com/dispel-re/dispel-multi/model"
	"github.com/stretchr/testify/assert"
)

func TestClientHostAndUsernameRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 30, // Command code
		26, 0, // Packet length
		68, 69, 83, 75, 84, 79, 80, 45, 49, 51, 51, 55, 73, 83, 72, 0, // Host name
		85, 115, 101, 114, 0, // User name
	}

	// Act
	req := ClientHostAndUsernameRequest(packet[4:])
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "DESKTOP-1337ISH", data.ComputerHostname)
	assert.Equal(t, "User", data.ComputerUsername)
}

func TestBackend_HandleClientHostAndUsername(t *testing.T) {
	b := &Backend{}
	conn := &mockConn{}
	session := &model.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	assert.NoError(t, b.HandleClientHostAndUsername(session, ClientHostAndUsernameRequest{
		255, 30, // Command code
		26, 0, // Packet length
		68, 69, 83, 75, 84, 79, 80, 45, 49, 51, 51, 55, 73, 83, 72, 0, // Host name
		85, 115, 101, 114, 0, // User name
	}))
	assert.Equal(t, []byte{255, 30, 6, 0}, conn.Written[0:4]) // Header
	assert.Equal(t, []byte{1, 0}, conn.Written[4:6])          // Accepted state
	assert.Len(t, conn.Written, 6)
}
