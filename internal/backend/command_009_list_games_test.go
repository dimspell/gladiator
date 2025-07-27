package backend

import (
	"context"
	"github.com/dimspell/gladiator/internal/backend/proxy/relay"
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
		tt := []struct {
			name         string
			proxyFactory ProxyFactory
		}{
			{"lan", &direct.ProxyLAN{"127.0.100.1"}},
			{"relay", &relay.ProxyRelay{RelayServerAddr: "127.0.0.1:9999"}},
		}

		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				b := &Backend{SessionManager: NewSessionManager(tc.proxyFactory, gameClient)}
				conn := &mockConn{}
				session := b.SessionManager.Add(conn)
				session.SetLogonData(&v1.User{UserId: 2137, Username: "mage"})

				assert.NoError(t, b.HandleListGames(context.Background(), session, ListGamesRequest{}))
				assert.Len(t, conn.Written, 8)
				assert.Equal(t, []byte{255, 9, 8, 0}, conn.Written[0:4]) // Header
				assert.Equal(t, []byte{0, 0, 0, 0}, conn.Written[4:8])   // Number of games
			})
		}
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
		tt := []struct {
			name         string
			proxyFactory ProxyFactory
			expectedIP   []byte
		}{
			{"lan", &direct.ProxyLAN{"127.0.100.1"}, []byte{127, 0, 21, 37}},
			{"relay", &relay.ProxyRelay{RelayServerAddr: "127.0.0.1:9999"}, []byte{127, 0, 0, 2}},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				b := &Backend{SessionManager: NewSessionManager(tc.proxyFactory, gameClient)}
				conn := &mockConn{}
				session := b.SessionManager.Add(conn)
				session.SetLogonData(&v1.User{UserId: 2137, Username: "mage"})

				assert.NoError(t, b.HandleListGames(context.Background(), session, ListGamesRequest{}))
				assert.Len(t, conn.Written, 21)

				assert.Equal(t, []byte{255, 9, 21, 0}, conn.Written[0:4])                           // Header
				assert.Equal(t, []byte{1, 0, 0, 0}, conn.Written[4:8])                              // Number of games
				assert.Equal(t, tc.expectedIP, conn.Written[8:12])                                  // Host IP address
				assert.Equal(t, []byte{'r', 'e', 't', 'r', 'e', 'a', 't', 0, 0}, conn.Written[12:]) // Room name and no password
			})
		}
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

		tt := []struct {
			name                 string
			proxyFactory         ProxyFactory
			expectedIPFirstGame  []byte
			expectedIPSecondGame []byte
		}{
			{
				name:                 "lan",
				proxyFactory:         &direct.ProxyLAN{"127.0.100.1"},
				expectedIPFirstGame:  []byte{127, 0, 21, 37},
				expectedIPSecondGame: []byte{127, 0, 13, 37},
			},
			{
				name:                 "relay",
				proxyFactory:         &relay.ProxyRelay{RelayServerAddr: "127.0.0.1:9999"},
				expectedIPFirstGame:  []byte{127, 0, 0, 2},
				expectedIPSecondGame: []byte{127, 0, 0, 2},
			},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				b := &Backend{SessionManager: NewSessionManager(tc.proxyFactory, gameClient)}
				conn := &mockConn{}
				session := b.SessionManager.Add(conn)
				session.SetLogonData(&v1.User{UserId: 2137, Username: "mage"})

				assert.NoError(t, b.HandleListGames(context.Background(), session, ListGamesRequest{}))
				assert.Len(t, conn.Written, 39)
				assert.Equal(t, []byte{255, 9, 39, 0}, conn.Written[0:4])     // Header
				assert.Equal(t, []byte{2, 0, 0, 0}, conn.Written[4:8])        // Number of games
				assert.Equal(t, tc.expectedIPFirstGame, conn.Written[8:12])   // Host IP Address
				assert.Equal(t, []byte("RoomName\x00"), conn.Written[12:21])  // Room name
				assert.Equal(t, []byte("secret\x00"), conn.Written[21:28])    // Password
				assert.Equal(t, tc.expectedIPSecondGame, conn.Written[28:32]) // Host IP Address
				assert.Equal(t, []byte("Other\x00"), conn.Written[32:38])     // Room name
				assert.Equal(t, []byte("\x00"), conn.Written[38:39])          // Password
			})
		}
	})
}
