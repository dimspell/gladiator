package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectChannelRequest(t *testing.T) {
	packet := []byte{
		255, 12,
		0, 0,
		99, 104, 97, 110, 110, 101, 108, 0,
		68, 73, 83, 80, 69, 76, 0,
	}
	req := SelectChannelRequest(packet[4:])

	assert.Equal(t, "channel", req.ChannelName())
}
