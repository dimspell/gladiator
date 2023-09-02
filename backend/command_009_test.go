package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListGamesRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 9,
		4, 0,
	}

	// Act
	req := ListChannelsRequest(packet[4:])

	// Assert
	assert.Empty(t, req)
}
