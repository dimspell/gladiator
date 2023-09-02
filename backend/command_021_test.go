package backend

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPingRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 21, // Command code
		8, 0, // Packet length
		232, 214, 133, 0, // Time in milliseconds
	}
	req := PingRequest(packet[4:])

	// Act
	ms, err := req.Parse()
	date, _ := req.ParseDate()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint32(8771304), ms)
	assert.Equal(t, "02:26:11", date.Format(time.TimeOnly))
}
