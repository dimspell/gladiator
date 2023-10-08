package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCharacterInventoryRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 68, // Command code
		20, 0, // Packet length
		117, 115, 101, 114, 0, // User name
		99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
		0, // Unknown
	}
	req := GetCharacterInventoryRequest(packet[4:])

	// Act
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "user", data.Username)
	assert.Equal(t, "character", data.CharacterName)
	assert.Equal(t, []byte{0}, data.Unknown)
}
