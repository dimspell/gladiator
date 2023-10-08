package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientHostAndUsernameRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 30,
		26, 0,
		68, 69, 83, 75, 84, 79, 80, 45, 49, 51, 51, 55, 73, 83, 72, 0,
		85, 115, 101, 114, 0,
	}

	// Act
	req := ClientHostAndUsernameRequest(packet[4:])
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "DESKTOP-1337ISH", data.ComputerHostname)
	assert.Equal(t, "User", data.ComputerUsername)
}
