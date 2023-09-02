package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendLobbyMessageRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 14, // Command code
		17, 0, // Packet length
		84, 101, 120, 116, 32, 109, 101, 115, 115, 97, 103, 101, 0, // Text message
	}

	// Act
	req := SendLobbyMessageRequest(packet[4:])
	message, err := req.Parse()

	assert.NoError(t, err)
	assert.Equal(t, "Text message", message)
}

func TestSendLobbyMessageRequest_Parse(t *testing.T) {
	t.Run("valid message", func(t *testing.T) {
		// Arrange
		req := SendLobbyMessageRequest("Hello\x00")

		// Act
		message, err := req.Parse()

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Hello", message)
	})

	t.Run("missing null terminator", func(t *testing.T) {
		// Arrange
		req := SendLobbyMessageRequest("Hello")

		// Act
		message, err := req.Parse()

		// Assert
		assert.Error(t, err)
		assert.Empty(t, message)
	})

	t.Run("empty message", func(t *testing.T) {
		// Arrange
		req := SendLobbyMessageRequest("\x00")

		// Act
		message, err := req.Parse()

		// Assert
		assert.NoError(t, err)
		assert.Empty(t, message)
	})

	t.Run("extra null terminator", func(t *testing.T) {
		// Arrange
		req := SendLobbyMessageRequest("\x00\x00")

		// Act
		message, err := req.Parse()

		// Assert
		assert.NoError(t, err)
		assert.Empty(t, message)
	})
}
