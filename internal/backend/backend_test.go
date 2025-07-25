package backend

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/backend/proxy/direct"
	"github.com/dimspell/gladiator/internal/console"
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

	// Optionally close any channels, etc.
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

type mockGameClient struct {
	multiv1connect.UnimplementedGameServiceHandler

	GetGameResponse    *connect.Response[v1.GetGameResponse]
	JoinGameResponse   *connect.Response[v1.JoinGameResponse]
	CreateGameResponse *connect.Response[v1.CreateGameResponse]
	ListGamesResponse  *connect.Response[v1.ListGamesResponse]
}

func (m *mockGameClient) GetGame(context.Context, *connect.Request[v1.GetGameRequest]) (*connect.Response[v1.GetGameResponse], error) {
	return m.GetGameResponse, nil
}

func (m *mockGameClient) JoinGame(context.Context, *connect.Request[v1.JoinGameRequest]) (*connect.Response[v1.JoinGameResponse], error) {
	return m.JoinGameResponse, nil
}

func (m *mockGameClient) CreateGame(context.Context, *connect.Request[v1.CreateGameRequest]) (*connect.Response[v1.CreateGameResponse], error) {
	return m.CreateGameResponse, nil
}

func (m *mockGameClient) ListGames(context.Context, *connect.Request[v1.ListGamesRequest]) (*connect.Response[v1.ListGamesResponse], error) {
	return m.ListGamesResponse, nil
}

type mockCharacterClient struct {
	multiv1connect.UnimplementedCharacterServiceHandler

	GetCharacterResponse   *connect.Response[v1.GetCharacterResponse]
	ListCharactersResponse *connect.Response[v1.ListCharactersResponse]
}

func (m *mockCharacterClient) GetCharacter(context.Context, *connect.Request[v1.GetCharacterRequest]) (*connect.Response[v1.GetCharacterResponse], error) {
	return m.GetCharacterResponse, nil
}

func (m *mockCharacterClient) ListCharacters(context.Context, *connect.Request[v1.ListCharactersRequest]) (*connect.Response[v1.ListCharactersResponse], error) {
	return m.ListCharactersResponse, nil
}

func helperNewBackend(tb testing.TB, gameClient multiv1connect.GameServiceClient) (bd *Backend, px *direct.ProxyLAN, cs *console.Console) {
	tb.Helper()

	cs = &console.Console{
		RoomService: console.NewRoomService(),
	}
	ts := httptest.NewServer(http.HandlerFunc(cs.HandleWebSocket))

	// Use bogon IP addressing for tests (https://datatracker.ietf.org/doc/rfc6752/).
	px = &direct.ProxyLAN{MyIPAddress: "198.51.100.1"}

	bd = &Backend{
		// Replace the HTTP schema prefix for websocket connection.
		SignalServerURL: "ws://" + ts.URL[len("http://"):],
		SessionManager:  NewSessionManager(px, gameClient),
	}

	tb.Cleanup(func() {
		ts.Close()
	})

	return bd, px, cs
}
