package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCharactersRequest(t *testing.T) {
	packet := []byte{
		255, 60, // Command code
		10, 0, // Packet length
		108, 111, 103, 105, 110, 0, // Username = login
	}
	req := GetCharactersRequest(packet[4:])

	assert.Equal(t, "login", req.Username())
}
