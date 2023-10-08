package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteCharacterRequest_UsernameAndCharacterName(t *testing.T) {
	t.Run("packet parsing", func(t *testing.T) {
		packet := []byte{
			255, 61, // Command code
			14, 0, // Packet length
			117, 115, 101, 114, 0, // User name
			99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
			0, // Unknown (slot?)
		}
		req := DeleteCharacterRequest(packet[4:])
		data, err := req.Parse()

		assert.NoError(t, err)
		assert.Equal(t, "user", data.Username)
		assert.Equal(t, "character", data.CharacterName)
	})

	t.Run("valid names", func(t *testing.T) {
		// Arrange
		input := []byte("user\x00character\x00\x00")
		req := DeleteCharacterRequest(input)

		// Act
		data, err := req.Parse()

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "user", data.Username)
		assert.Equal(t, "character", data.CharacterName)
	})

	t.Run("missing null byte", func(t *testing.T) {
		// Arrange
		input := []byte("usercharacter\x00")
		req := DeleteCharacterRequest(input)

		// Act
		data, err := req.Parse()

		// Assert
		assert.Error(t, err)
		assert.Empty(t, data.Username)
		assert.Empty(t, data.CharacterName)
	})

	t.Run("empty data", func(t *testing.T) {
		// Arrange
		input := []byte{}
		req := DeleteCharacterRequest(input)

		// Act
		data, err := req.Parse()

		// Assert
		assert.Error(t, err)
		assert.Empty(t, data.Username)
		assert.Empty(t, data.CharacterName)
	})
}
