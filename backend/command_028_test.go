package backend

import (
	"testing"

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
	state, mapId, roomName, password, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), state)
	assert.Equal(t, uint32(3), mapId)
	assert.Equal(t, "room", roomName)
	assert.Equal(t, "", password)
}
