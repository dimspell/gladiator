package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectCharacterRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 76, // Command code
		15, 0, // Packet length
		117, 115, 101, 114, 0, // User name
		99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
	}

	// Act
	req := SelectCharacterRequest(packet[4:])
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "user", data.Username)
	assert.Equal(t, "character", data.CharacterName)
}
