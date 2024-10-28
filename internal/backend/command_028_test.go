package backend

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/stretchr/testify/assert"
)

func TestCreateGameRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 28, // Command code
		18, 0, // Packet length
		1, 0, 0, 0, // State
		3, 0, 0, 0, // Map ID
		114, 111, 111, 109, 0, // Game room name
		0, // Password
	}

	// Act
	req := CreateGameRequest(packet[4:])
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), data.State)
	assert.Equal(t, uint32(3), data.MapID)
	assert.Equal(t, "room", data.RoomName)
	assert.Equal(t, "", data.Password)
}

func TestBackend_HandleCreateGame(t *testing.T) {
	b, _ := helperNewBackend(t)
	b.gameClient = &mockGameClient{
		CreateGameResponse: connect.NewResponse(&v1.CreateGameResponse{
			Game: &v1.Game{
				GameId:        "room",
				Name:          "room",
				Password:      "",
				HostIpAddress: "127.0.0.1",
				MapId:         v1.GameMap_FrozenLabyrinth,
				HostUserId:    2137,
			},
		}),
		GetGameResponse: connect.NewResponse(&v1.GetGameResponse{
			Game: &v1.Game{
				GameId:        "room",
				Name:          "room",
				Password:      "",
				HostIpAddress: "127.0.0.1",
				MapId:         v1.GameMap_FrozenLabyrinth,
				HostUserId:    2137,
			},
			Players: []*v1.Player{
				{
					UserId:      2137,
					Username:    "TEST",
					CharacterId: 0,
					ClassType:   0,
					IpAddress:   "127.0.0.1",
				},
			},
		}),
	}

	conn := &mockConn{}
	session := b.AddSession(conn)
	session.SetLogonData(&v1.User{UserId: 2137, Username: "JP"})
	session.ID = "TEST"

	ctx := context.Background()
	if err := b.ConnectToLobby(ctx, &v1.User{UserId: session.UserID, Username: session.Username}, session); err != nil {
		t.Error(err)
		return
	}
	if err := b.RegisterNewObserver(ctx, session); err != nil {
		t.Errorf("error registering observer: %v", err)
		return
	}

	// State = 0
	assert.NoError(t, b.HandleCreateGame(session, CreateGameRequest{
		0, 0, 0, 0, // State
		3, 0, 0, 0, // Map ID
		'r', 'o', 'o', 'm', 0, // Game room name
		0, // Password
	}))
	assert.Equal(t, []byte{255, 28, 8, 0}, conn.Written[0:4]) // Header
	assert.Equal(t, []byte{1, 0, 0, 0}, conn.Written[4:8])    // Next state
	assert.Len(t, conn.Written, 8)

	conn.Written = nil

	// State = 1
	assert.NoError(t, b.HandleCreateGame(session, CreateGameRequest{
		1, 0, 0, 0, // State
		3, 0, 0, 0, // Map ID
		'r', 'o', 'o', 'm', 0, // Game room name
		0, // Password
	}))
	assert.Equal(t, []byte{255, 28, 8, 0}, conn.Written[0:4]) // Header
	assert.Equal(t, []byte{2, 0, 0, 0}, conn.Written[4:8])    // Next state
	assert.Len(t, conn.Written, 8)
}
