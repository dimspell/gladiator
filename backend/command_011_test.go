package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListChannelsRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 11,
		4, 0,
	}

	// Act
	req := ListChannelsRequest(packet[4:])

	// Assert
	assert.Empty(t, req)
}
