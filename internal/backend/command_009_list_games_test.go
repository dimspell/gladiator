package backend

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/proxy/direct"
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

func TestBackend_HandleListGames(t *testing.T) {
	t.Run("no games", func(t *testing.T) {
		gameClient := &mockGameClient{
			ListGamesResponse: connect.NewResponse(&v1.ListGamesResponse{Games: []*v1.Game{}}),
		}
		b := &Backend{SessionManager: NewSessionManager(&direct.ProxyLAN{"127.0.100.1"}, gameClient)}
		conn := &mockConn{}
		session := b.SessionManager.Add(conn)
		session.SetLogonData(&v1.User{UserId: 2137, Username: "mage"})

		assert.NoError(t, b.HandleListGames(context.Background(), session, ListGamesRequest{}))
		assert.Len(t, conn.Written, 8)
		assert.Equal(t, []byte{255, 9, 8, 0}, conn.Written[0:4]) // Header
		assert.Equal(t, []byte{0, 0, 0, 0}, conn.Written[4:8])   // Number of games
	})

	t.Run("with one game", func(t *testing.T) {
		gameClient := &mockGameClient{
			ListGamesResponse: connect.NewResponse(&v1.ListGamesResponse{Games: []*v1.Game{
				{
					GameId:        "gameId",
					Name:          "retreat",
					Password:      "",
					HostIpAddress: "127.0.21.37",
					MapId:         v1.GameMap_UnderworldRetreat,
				},
			}}),
		}
		b := &Backend{SessionManager: NewSessionManager(&direct.ProxyLAN{"127.0.100.1"}, gameClient)}
		conn := &mockConn{}
		session := b.SessionManager.Add(conn)
		session.SetLogonData(&v1.User{UserId: 2137, Username: "mage"})

		assert.NoError(t, b.HandleListGames(context.Background(), session, ListGamesRequest{}))
		assert.Len(t, conn.Written, 21)

		assert.Equal(t, []byte{255, 9, 21, 0}, conn.Written[0:4])                           // Header
		assert.Equal(t, []byte{1, 0, 0, 0}, conn.Written[4:8])                              // Number of games
		assert.Equal(t, []byte{127, 0, 21, 37}, conn.Written[8:12])                         // Host IP address
		assert.Equal(t, []byte{'r', 'e', 't', 'r', 'e', 'a', 't', 0, 0}, conn.Written[12:]) // Room name and no password

	})

	t.Run("with games", func(t *testing.T) {
		gameClient := &mockGameClient{
			ListGamesResponse: connect.NewResponse(&v1.ListGamesResponse{Games: []*v1.Game{
				{
					GameId:        "gameId",
					Name:          "RoomName",
					Password:      "secret",
					HostIpAddress: "127.0.21.37",
					MapId:         v1.GameMap_UnderworldRetreat,
				},
				{
					GameId:        "gameId",
					Name:          "Other",
					Password:      "",
					HostIpAddress: "127.0.13.37",
					MapId:         v1.GameMap_AbandonedRealm,
				},
			}}),
		}

		b := &Backend{SessionManager: NewSessionManager(&direct.ProxyLAN{"127.0.100.1"}, gameClient)}
		conn := &mockConn{}
		session := b.SessionManager.Add(conn)
		session.SetLogonData(&v1.User{UserId: 2137, Username: "mage"})

		assert.NoError(t, b.HandleListGames(context.Background(), session, ListGamesRequest{}))
		assert.Len(t, conn.Written, 39)
		assert.Equal(t, []byte{255, 9, 39, 0}, conn.Written[0:4])    // Header
		assert.Equal(t, []byte{2, 0, 0, 0}, conn.Written[4:8])       // Number of games
		assert.Equal(t, []byte{127, 0, 21, 37}, conn.Written[8:12])  // Host IP Address
		assert.Equal(t, []byte("RoomName\x00"), conn.Written[12:21]) // Room name
		assert.Equal(t, []byte("secret\x00"), conn.Written[21:28])   // Password
		assert.Equal(t, []byte{127, 0, 13, 37}, conn.Written[28:32]) // Host IP Address
		assert.Equal(t, []byte("Other\x00"), conn.Written[32:38])    // Room name
		assert.Equal(t, []byte("\x00"), conn.Written[38:39])         // Password
	})
}
