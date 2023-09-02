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
		user, character, err := req.Parse()

		assert.NoError(t, err)
		assert.Equal(t, "user", user)
		assert.Equal(t, "character", character)
	})

	t.Run("valid names", func(t *testing.T) {
		// Arrange
		data := []byte("user\x00character\x00\x00")
		req := DeleteCharacterRequest(data)

		// Act
		username, characterName, err := req.Parse()

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "user", username)
		assert.Equal(t, "character", characterName)
	})

	t.Run("missing null byte", func(t *testing.T) {
		// Arrange
		data := []byte("usercharacter\x00")
		req := DeleteCharacterRequest(data)

		// Act
		username, characterName, err := req.Parse()

		// Assert
		assert.Error(t, err)
		assert.Empty(t, username)
		assert.Empty(t, characterName)
	})

	t.Run("empty data", func(t *testing.T) {
		// Arrange
		data := []byte{}
		req := DeleteCharacterRequest(data)

		// Act
		username, characterName, err := req.Parse()

		// Assert
		assert.Error(t, err)
		assert.Empty(t, username)
		assert.Empty(t, characterName)
	})
}
