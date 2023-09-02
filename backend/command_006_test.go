package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestBackend_HandleAuthorizationHandshake(t *testing.T) {
// 	t.Run("valid request", func(t *testing.T) {
// 		// Arrange
// 		b := &Backend{}
// 		conn := &mockConn{}
// 		session := &model.Session{Conn: conn}
// 		req := AuthorizationHandshakeRequest{}
//
// 		// Act
// 		err := b.HandleAuthorizationHandshake(session, req)
//
// 		// Assert
// 		assert.NoError(t, err)
// 		assert.Equal(t, []byte("ENET\x00"), conn.Written)
// 	})
//
// 	t.Run("connection error", func(t *testing.T) {
// 		// Arrange
// 		b := &Backend{}
// 		session := &model.Session{Conn: &mockConn{
// 			WriteError: errors.New("write error"),
// 		}}
// 		req := AuthorizationHandshakeRequest{}
//
// 		// Act
// 		err := b.HandleAuthorizationHandshake(session, req)
//
// 		// Assert
// 		assert.Error(t, err)
// 		assert.Contains(t, err.Error(), "write error")
// 	})
// }

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
	authKey, unknown, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "68XIPSID", authKey)
	assert.Equal(t, uint32(3), unknown)
}
