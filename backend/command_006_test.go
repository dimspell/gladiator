package backend

import (
	"errors"
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
		req := HandleAuthorizationHandshakeRequest{}

		// Act
		err := b.HandleAuthorizationHandshake(session, req)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, []byte("ENET\x00"), conn.Written)
	})

	t.Run("connection error", func(t *testing.T) {
		// Arrange
		b := &Backend{}
		session := &model.Session{Conn: &mockConn{
			WriteError: errors.New("write error"),
		}}
		req := HandleAuthorizationHandshakeRequest{}

		// Act
		err := b.HandleAuthorizationHandshake(session, req)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "write error")
	})
}
