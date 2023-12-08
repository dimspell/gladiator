package backend

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/dispel-re/dispel-multi/console/database"
	v1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
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

type mockGameClient struct {
	multiv1connect.UnimplementedGameServiceHandler

	ListGamesResponse *connect.Response[v1.ListGamesResponse]
}

func (m *mockGameClient) ListGames(context.Context, *connect.Request[v1.ListGamesRequest]) (*connect.Response[v1.ListGamesResponse], error) {
	return m.ListGamesResponse, nil
}

type mockCharacterClient struct {
	multiv1connect.UnimplementedCharacterServiceHandler
}

func (m *mockGameClient) ListCharacters(context.Context, *connect.Request[v1.ListCharactersRequest]) (*connect.Response[v1.ListCharactersResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("multi.v1.CharacterService.ListCharacters is not implemented"))
}
