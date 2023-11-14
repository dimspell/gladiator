package backend

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/stretchr/testify/assert"
)

type mockConn struct {
	ReadError  error
	Written    []byte
	WriteError error
	CloseError error

	LocalAddress  net.Addr
	RemoteAddress net.Addr
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	// Return injected error
	m.Written = append(m.Written, b...)
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
	return m.LocalAddress
}

func (m *mockConn) RemoteAddr() net.Addr {
	return m.RemoteAddress
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

func testDB(t *testing.T) *database.Queries {
	db, err := database.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	queries, err := db.Queries()
	if err != nil {
		t.Fatal(err)
	}
	return queries
}

func Test_splitMultiPacket(t *testing.T) {
	t.Run("non-compatible packet", func(t *testing.T) {
		packets := splitMultiPacket([]byte{1})

		assert.Equal(t, 1, len(packets))
		assert.True(t, bytes.Equal([]byte{1}, packets[0]))
	})

	t.Run("single packet", func(t *testing.T) {
		packets := splitMultiPacket([]byte{255, 1, 4, 0})

		assert.Equal(t, 1, len(packets))
		assert.True(t, bytes.Equal([]byte{255, 1, 4, 0}, packets[0]))
	})

	t.Run("two packets", func(t *testing.T) {
		packets := splitMultiPacket([]byte{
			255, 1, 8, 0, 1, 0, 0, 0,
			255, 2, 4, 0,
		})

		assert.Equal(t, 2, len(packets))
		assert.True(t, bytes.Equal([]byte{255, 1, 8, 0, 1, 0, 0, 0}, packets[0]))
		assert.True(t, bytes.Equal([]byte{255, 2, 4, 0}, packets[1]))
	})

	t.Run("three packets", func(t *testing.T) {
		packets := splitMultiPacket([]byte{
			255, 1, 8, 0, 1, 0, 0, 0,
			255, 2, 4, 0,
			255, 3, 6, 0, 1, 0,
		})

		assert.Equal(t, 3, len(packets))
		assert.True(t, bytes.Equal([]byte{255, 1, 8, 0, 1, 0, 0, 0}, packets[0]))
		assert.True(t, bytes.Equal([]byte{255, 2, 4, 0}, packets[1]))
		assert.True(t, bytes.Equal([]byte{255, 3, 6, 0, 1, 0}, packets[2]))
	})
}
