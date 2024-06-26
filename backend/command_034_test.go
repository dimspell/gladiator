package backend

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/dimspell/gladiator/backend/proxy"
	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/model"
	"github.com/stretchr/testify/assert"
)

func TestBackend_HandleJoinGame(t *testing.T) {
	b := &Backend{
		Proxy: proxy.NewLAN(),
		gameClient: &mockGameClient{
			GetGameResponse: connect.NewResponse(&v1.GetGameResponse{
				Game: &v1.Game{
					GameId:        100,
					Name:          "retreat",
					Password:      "",
					HostIpAddress: "192.168.121.212",
					MapId:         2,
				},
			}),
			JoinGameResponse: connect.NewResponse(&v1.JoinGameResponse{}),
			ListPlayersResponse: connect.NewResponse(&v1.ListPlayersResponse{
				Players: []*v1.Player{
					{
						CharacterName: "archer",
						ClassType:     int64(model.ClassTypeArcher),
						IpAddress:     "192.168.121.212",
						Username:      "archer",
					},
					{
						CharacterName: "mage",
						ClassType:     int64(model.ClassTypeMage),
						IpAddress:     "192.168.121.169",
						Username:      "mage",
					},
				},
			}),
		},
	}
	conn := &mockConn{}
	session := &model.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP", LocalIpAddress: "127.0.100.1"}

	assert.NoError(t, b.HandleJoinGame(session, JoinGameRequest{
		103, 97, 109, 101, 82, 111, 111, 109, 0, // Game name
		0, // Password
	}))

	assert.Equal(t, []byte{255, 34, 34, 0}, conn.Written[0:4]) // Header
	assert.Equal(t, []byte{2, 0}, conn.Written[4:6])           // Game State

	firstPlayer := []byte{'a', 'r', 'c', 'h', 'e', 'r', 0}
	assert.Equal(t, []byte{2, 0, 0, 0}, conn.Written[6:10])            // Class type (archer)
	assert.Equal(t, []byte{192, 168, 121, 212}, conn.Written[10:14])   // IP Address
	assert.Equal(t, firstPlayer, conn.Written[14:14+len(firstPlayer)]) // Player name

	start := 14 + len(firstPlayer)
	secondPlayer := []byte{'m', 'a', 'g', 'e', 0}
	assert.Equal(t, []byte{3, 0, 0, 0}, conn.Written[start:start+4])           // Class type (mage)
	assert.Equal(t, []byte{192, 168, 121, 169}, conn.Written[start+4:start+8]) // IP Address
	assert.Equal(t, secondPlayer, conn.Written[start+8:])                      // Player name
}
