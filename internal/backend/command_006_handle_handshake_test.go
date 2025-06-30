package backend

import (
	"fmt"
	"testing"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/stretchr/testify/assert"
)

func TestBackend_HandleAuthorizationHandshake(t *testing.T) {
	t.Run("error cases", func(t *testing.T) {
		t.Run("send wrong auth key", func(t *testing.T) {
			// Arrange
			b := &Backend{}
			conn := &mockConn{}
			session := &bsession.Session{Conn: conn}
			req := AuthorizationHandshakeRequest("WRONGSID\x03\x00\x00\x00")

			// Act
			err := b.HandleAuthorizationHandshake(session, req)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "wrong auth key")
			assert.Len(t, conn.Written, 8)
			assert.Equal(t, []byte{255, 6, 8, 0}, conn.Written[0:4]) // Header
			assert.Equal(t, []byte{0, 0, 0, 0}, conn.Written[4:])    // Response
		})

		t.Run("send wrong version number", func(t *testing.T) {
			// Arrange
			b := &Backend{}
			conn := &mockConn{}
			session := &bsession.Session{Conn: conn}
			req := AuthorizationHandshakeRequest("68XIPSID\x04\x00\x00\x00")

			// Act
			err := b.HandleAuthorizationHandshake(session, req)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid version number")
			assert.Len(t, conn.Written, 8)
			assert.Equal(t, []byte{255, 6, 8, 0}, conn.Written[0:4]) // Header
			assert.Equal(t, []byte{0, 0, 0, 0}, conn.Written[4:])    // Response
		})

		t.Run("sending failed because of network error", func(t *testing.T) {
			// Arrange
			b := &Backend{}
			conn := &mockConn{WriteError: fmt.Errorf("network error")}
			session := &bsession.Session{Conn: conn}
			req := AuthorizationHandshakeRequest("68XIPSID\x03\x00\x00\x00")

			// Act
			err := b.HandleAuthorizationHandshake(session, req)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "network error")
		})
	})

	t.Run("valid request", func(t *testing.T) {
		// Arrange
		b := &Backend{}
		conn := &mockConn{}
		session := &bsession.Session{Conn: conn}
		req := AuthorizationHandshakeRequest("68XIPSID\x03\x00\x00\x00")

		// Act
		err := b.HandleAuthorizationHandshake(session, req)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, conn.Written, 9)
		assert.Equal(t, []byte{255, 6, 9, 0}, conn.Written[0:4]) // Header
		assert.Equal(t, []byte("ENET\x00"), conn.Written[4:])    // Response
	})
}

func TestAuthorizationHandshakeRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 6, // Command code
		16, 0, // Packet length
		54, 56, 88, 73, 80, 83, 73, 68, // "68XIPSID"
		3, 0, 0, 0, // Probably a version number
	}

	// Act
	req := AuthorizationHandshakeRequest(packet[4:])
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "68XIPSID", string(data.AuthKey))
	assert.Equal(t, uint32(3), data.VersionNumber)
}
