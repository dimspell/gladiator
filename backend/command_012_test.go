package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectChannelRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 12, // Command code
		19, 0, // Packet length
		99, 104, 97, 110, 110, 101, 108, 0, // "channel"
		68, 73, 83, 80, 69, 76, 0, // "DISPEL"
	}

	// Act
	req := SelectChannelRequest(packet[4:])
	channelName, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "channel", channelName)
}
