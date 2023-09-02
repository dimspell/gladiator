package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCharactersRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 60, // Command code
		10, 0, // Packet length
		108, 111, 103, 105, 110, 0, // Username = login
	}

	// Act
	req := GetCharactersRequest(packet[4:])
	username, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "login", username)
}
