package backend

import (
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/stretchr/testify/assert"
)

func TestBackend_HandleSelectGame(t *testing.T) {
	t.Run("Sample mocked game", func(t *testing.T) {
		b, _ := helperNewBackend(t)
		b.gameClient = &mockGameClient{
			GetGameResponse: connect.NewResponse(&v1.GetGameResponse{
				Game: &v1.Game{
					GameId:        "gameId",
					Name:          "retreat",
					Password:      "",
					HostIpAddress: "192.168.121.212",
					MapId:         2,
				},
				Players: []*v1.Player{
					{
						// CharacterName: "archer",
						ClassType: v1.ClassType_Archer,
						IpAddress: "192.168.121.212",
						Username:  "archer",
					},
					// {
					//	CharacterName: "mage",
					//	ClassType:     int64(model.ClassTypeMage),
					//	IpAddress:     "192.168.121.169",
					//	Username:      "mage",
					// },
				},
			}),
		}

		conn := &mockConn{}
		session := &Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "mage"}

		assert.NoError(t, b.HandleSelectGame(session, SelectGameRequest{
			'r', 'e', 't', 'r', 'e', 'a', 'a', 't', 0, // Game name
			0, // Password
		}))

		t.Log(conn.Written, string(conn.Written))

		assert.Equal(t, []byte{255, 69, 23, 0}, conn.Written[0:4])                  // Header
		assert.Equal(t, []byte{2, 0, 0, 0}, conn.Written[4:8])                      // Map ID
		assert.Equal(t, []byte{2, 0, 0, 0}, conn.Written[8:12])                     // Class type
		assert.Equal(t, []byte{192, 168, 121, 212}, conn.Written[12:16])            // IP Address
		assert.Equal(t, []byte{'a', 'r', 'c', 'h', 'e', 'r', 0}, conn.Written[16:]) // Player name
	})

	t.Run("HostRoom only", func(t *testing.T) {
		b, _ := helperNewBackend(t)
		b.gameClient = &mockGameClient{
			GetGameResponse: connect.NewResponse(&v1.GetGameResponse{
				Game: &v1.Game{
					GameId:        "gameId",
					Name:          "gameRoom",
					Password:      "",
					HostIpAddress: "127.0.0.28",
					MapId:         2,
				},
				Players: []*v1.Player{
					{
						// CharacterName: "hostMagician",
						ClassType: v1.ClassType_Mage,
						IpAddress: "127.0.0.28",
						Username:  "playerA",
					},
				},
			}),
		}
		conn := &mockConn{}
		session := &Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

		assert.NoError(t, b.HandleSelectGame(session, SelectGameRequest{
			103, 97, 109, 101, 82, 111, 111, 109, 0, // Game name
			0, // Password
		}))

		assert.Len(t, conn.Written, 24)
		assert.Equal(t, []byte{255, 69}, conn.Written[0:2]) // Command code
		assert.Equal(t, []byte{24, 0}, conn.Written[2:4])   // Packet length

		assert.Equal(t, []byte{2, 0, 0, 0}, conn.Written[4:8]) // Map ID

		// First Player
		assert.Equal(t, []byte{3, 0, 0, 0}, conn.Written[8:12])     // Class = Magician
		assert.Equal(t, []byte{127, 0, 0, 28}, conn.Written[12:16]) // IP Address
		assert.Equal(t, []byte("playerA\x00"), conn.Written[16:24]) // Character name
	})
}
