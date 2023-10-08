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
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), data.State)
	assert.Equal(t, uint32(3), data.MapID)
	assert.Equal(t, "room", data.RoomName)
	assert.Equal(t, "", data.Password)
}
