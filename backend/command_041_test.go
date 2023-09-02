package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientAuthenticationRequest(t *testing.T) {
	packet := []byte{
		255, 41, // Command code
		19, 0, // Packet length
		2, 0, 0, 0, // Unknown
		112, 97, 115, 115, 0, // Password
		108, 111, 103, 105, 110, 0, // Username
	}
	req := ClientAuthenticationRequest(packet[4:])
	username, password := req.UsernameAndPassword()

	assert.Equal(t, uint32(2), req.Unknown())
	assert.Equal(t, "pass", password)
	assert.Equal(t, "login", username)
}
