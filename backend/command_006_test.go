package backend

import (
	"testing"

	"github.com/dispel-re/dispel-multi/model"
	"github.com/stretchr/testify/assert"
)

func TestBackend_HandleAuthorizationHandshake(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		// Arrange
		b := &Backend{}
		conn := &mockConn{}
		session := &model.Session{Conn: conn}
		req := AuthorizationHandshakeRequest("68XIPSID\x03\x00\x00\x00")

		// Act
		err := b.HandleAuthorizationHandshake(session, req)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, conn.Written, 9)
		assert.Equal(t, []byte{255, 6, 9, 0}, conn.Written[0:4]) // Header
		assert.Equal(t, []byte("ENET\x00"), conn.Written[4:])    // Response
	})

	t.Run("invalid request", func(t *testing.T) {
		// Arrange
		b := &Backend{}
		conn := &mockConn{}
		session := &model.Session{Conn: conn}
		req := AuthorizationHandshakeRequest("WRONG")

		// Act
		err := b.HandleAuthorizationHandshake(session, req)

		// Assert
		assert.Contains(t, err.Error(), "packet-6: malformed packet")
		assert.Len(t, conn.Written, 0) // No response
	})

	t.Run("invalid but padded request", func(t *testing.T) {
		// Arrange
		b := &Backend{}
		conn := &mockConn{}
		session := &model.Session{Conn: conn}
		req := AuthorizationHandshakeRequest("WRONGSID\x03\x00\x00\x00")

		// Act
		err := b.HandleAuthorizationHandshake(session, req)

		// Assert
		assert.Contains(t, err.Error(), `packet-6: wrong auth key: "WRONGSID"`)
		assert.Len(t, conn.Written, 8)
		assert.Equal(t, []byte{255, 6, 8, 0}, conn.Written[0:4]) // Header
		assert.Equal(t, []byte{0, 0, 0, 0}, conn.Written[4:])    // Response
	})
}

func TestAuthorizationHandshakeRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 6, // Command code
		16, 0, // Packet length
		54, 56, 88, 73, 80, 83, 73, 68, // "68XIPSID"
		3, 0, 0, 0, // Unknown counter
	}

	// Act
	req := AuthorizationHandshakeRequest(packet[4:])
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "68XIPSID", data.AuthKey)
	assert.Equal(t, uint32(3), data.VersionNumber)
}
