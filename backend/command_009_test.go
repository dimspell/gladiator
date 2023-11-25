package backend

import (
	"context"
	"database/sql"
	"testing"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
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
		db := testDB(t)
		b := &Backend{DB: db}
		conn := &mockConn{}
		session := &model.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

		assert.NoError(t, b.HandleListGames(session, ListGamesRequest{}))
		assert.Len(t, conn.Written, 8)
		assert.Equal(t, []byte{255, 9, 8, 0}, conn.Written[0:4]) // Header
		assert.Equal(t, []byte{0, 0, 0, 0}, conn.Written[4:8])   // Number of games
	})

	t.Run("with games", func(t *testing.T) {
		db := testDB(t)
		db.CreateGameRoom(context.TODO(), database.CreateGameRoomParams{
			Name:          "RoomName",
			Password:      sql.NullString{Valid: true, String: "secret"},
			HostIpAddress: "127.0.0.1",
		})

		b := &Backend{DB: db}
		conn := &mockConn{}
		session := &model.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

		assert.NoError(t, b.HandleListGames(session, ListGamesRequest{}))
		assert.Len(t, conn.Written, 28)
		assert.Equal(t, []byte{255, 9, 28, 0}, conn.Written[0:4])    // Header
		assert.Equal(t, []byte{1, 0, 0, 0}, conn.Written[4:8])       // Number of games
		assert.Equal(t, []byte{127, 0, 0, 1}, conn.Written[8:12])    // Host IP Address
		assert.Equal(t, []byte("RoomName\x00"), conn.Written[12:21]) // Room name
		assert.Equal(t, []byte("secret\x00"), conn.Written[21:28])   // Password
	})
}
