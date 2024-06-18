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
					HostIpAddress: "127.0.0.28",
					MapId:         2,
				},
			}),
			JoinGameResponse: connect.NewResponse(&v1.JoinGameResponse{}),
			ListPlayersResponse: connect.NewResponse(&v1.ListPlayersResponse{
				Players: []*v1.Player{
					{
						CharacterName: "archer",
						ClassType:     int64(model.ClassTypeArcher),
						IpAddress:     "127.0.0.28",
						Username:      "archer",
					},
					{
						CharacterName: "mage",
						ClassType:     int64(model.ClassTypeMage),
						IpAddress:     "127.0.1.100",
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

	t.Log(conn.Written)

	assert.Len(t, conn.Written, 34)
	assert.Equal(t, []byte{255, 34}, conn.Written[0:2]) // Command code
	assert.Equal(t, []byte{34, 0}, conn.Written[2:4])   // Packet length
	//
	//assert.Equal(t, []byte{2, 0}, conn.Written[4:6]) // Map ID
	//
	//// First Player
	//assert.Equal(t, []byte{3, 0, 0, 0}, conn.Written[6:10])     // Class = Magician
	//assert.Equal(t, []byte{127, 0, 0, 28}, conn.Written[10:14]) // IP Address
	//assert.Equal(t, []byte("archer\x00"), conn.Written[14:22])  // Character name
	//
	//// Second Player
	//assert.Equal(t, []byte{1, 0, 0, 0}, conn.Written[22:26])    // Class = Warrior
	//assert.Equal(t, []byte{127, 0, 0, 34}, conn.Written[26:30]) // IP Address
	//assert.Equal(t, []byte("mage\x00"), conn.Written[30:38])    // Character name
}
