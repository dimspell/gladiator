package backend

import (
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
	"github.com/stretchr/testify/assert"
)

func TestBackend_HandleJoinGame(t *testing.T) {
	b := &Backend{
		GameClient: &mockGameClient{
			GetGameResponse: connect.NewResponse(&v1.GetGameResponse{
				Game: &v1.Game{
					GameId:        100,
					Name:          "gameRoom",
					Password:      "",
					HostIpAddress: "127.0.0.28",
					MapId:         2,
				},
			}),
			JoinGameResponse: connect.NewResponse(&v1.JoinGameResponse{}),
			ListPlayersResponse: connect.NewResponse(&v1.ListPlayersResponse{
				Players: []*v1.Player{
					{
						CharacterName: "hostMagician",
						ClassType:     int64(model.ClassTypeMage),
						IpAddress:     "127.0.0.28",
					},
					{
						CharacterName: "guestWarrior",
						ClassType:     int64(model.ClassTypeWarrior),
						IpAddress:     "127.0.0.34",
					},
				},
			}),
		},
	}
	conn := &mockConn{}
	session := &model.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	assert.NoError(t, b.HandleJoinGame(session, JoinGameRequest{
		103, 97, 109, 101, 82, 111, 111, 109, 0, // Game name
		0, // Password
	}))
	assert.Equal(t, []byte{255, 34}, conn.Written[0:2]) // Command code
	assert.Equal(t, []byte{50, 0}, conn.Written[2:4])   // Packet length
	assert.Len(t, conn.Written, 50)

	assert.Equal(t, []byte{2, 0, 0, 0}, conn.Written[4:8]) // Map ID

	// First Player
	assert.Equal(t, []byte{3, 0, 0, 0}, conn.Written[8:12])          // Class = Magician
	assert.Equal(t, []byte{127, 0, 0, 28}, conn.Written[12:16])      // IP Address
	assert.Equal(t, []byte("hostMagician\x00"), conn.Written[16:29]) // Character name

	// Second Player
	assert.Equal(t, []byte{1, 0, 0, 0}, conn.Written[29:33])         // Class = Warrior
	assert.Equal(t, []byte{127, 0, 0, 34}, conn.Written[33:37])      // IP Address
	assert.Equal(t, []byte("guestWarrior\x00"), conn.Written[37:50]) // Character name
}
