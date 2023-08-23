package backend

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/dispel-re/dispel-multi/model"
	"github.com/stretchr/testify/assert"
)

type mockConn struct {
	ReadError  error
	Written    []byte
	WriteError error
	CloseError error
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	// Return injected error
	return 0, m.WriteError
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	// Implement read logic
	return 0, m.ReadError
}

func (m *mockConn) Close() error {
	// Implement close logic
	return m.CloseError
}

func (m *mockConn) LocalAddr() net.Addr {
	// Return mock local address
	return nil
}

func (m *mockConn) RemoteAddr() net.Addr {
	// Return mock remote address
	return nil
}

func (m *mockConn) SetDeadline(t time.Time) error {
	// Implement deadline logic
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	// Implement read deadline logic
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	// Implement write deadline logic
	return nil
}

func (m *mockConn) SetWriteErr(err error) {
	m.WriteError = err
}

func (m *mockConn) CloseWithError(err error) {
	// Set CloseError
	m.CloseError = err

	// Optionally close any channels, etc
	// to simulate closed connection
}

func (m *mockConn) SetReadData(data []byte) {
	// Save data to return on Read calls
}

func (m *mockConn) AddReadData(data []byte) {
	// Append data to internal buffer
	// Return data on subsequent Read calls
}

func (m *mockConn) AllDataRead() bool {
	// Check if all queued data has been read
	return true
}

func (m *mockConn) ClearReadData() {
	// Clear any queued read data
}

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
