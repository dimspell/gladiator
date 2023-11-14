package backend

import (
	"testing"

	"github.com/dispel-re/dispel-multi/model"
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

func TestBackend_HandleListChannels(t *testing.T) {
	b := &Backend{}
	conn := &mockConn{}
	session := &model.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	assert.NoError(t, b.HandleListChannels(session, ListChannelsRequest{}))
	assert.Equal(t, []byte{255, 11, 11, 0}, conn.Written[0:4]) // Header
	assert.Equal(t, []byte("DISPEL\x00"), conn.Written[4:11])  // Channel name
	assert.Len(t, conn.Written, 11)
}
