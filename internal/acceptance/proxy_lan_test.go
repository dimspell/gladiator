package acceptance

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/backend/proxy/direct"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestProxyLAN_CreatesAndJoinRoom(t *testing.T) {
	logger.SetDiscardLogger()

	db, err := database.NewMemory()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
		return
	}
	defer db.Close()

	if err := database.Seed(db.Write); err != nil {
		t.Fatalf("failed to seed database: %v", err)
		return
	}

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cs := console.NewConsole(db)
	ts := httptest.NewServer(cs.HttpRouter())
	defer ts.Close()

	// Remove the HTTP schema prefix
	_ = console.WithConsoleAddr(ts.URL[len("http://"):], ts.URL)(cs)

	proxy1 := &direct.ProxyLAN{"198.51.100.1"}
	bd1 := backend.NewBackend("", ts.URL, proxy1)
	bd1.SignalServerURL = "ws://" + cs.ConsoleBindAddr + "/lobby"

	conn1 := &mockConn{}
	session1 := bd1.AddSession(conn1)

	t.Run("Host user has signs in and selects the character", func(t *testing.T) {
		assert.NoError(t, bd1.HandleClientAuthentication(ctx, session1, backend.ClientAuthenticationRequest{
			2, 0, 0, 0, // Unknown
			't', 'e', 's', 't', 0, // Password
			'a', 'r', 'c', 'h', 'e', 'r', 0, // Username
		}))
		if !bytes.Equal([]byte{255, 41, 8, 0, 1, 0, 0, 0}, conn1.Written) {
			t.Errorf("Not logged in, got: %v", conn1.Written)
			return
		}
		t.Log("Host user authenticated")

		// Select character
		assert.NoError(t, bd1.HandleSelectCharacter(ctx, session1, backend.SelectCharacterRequest{
			'a', 'r', 'c', 'h', 'e', 'r', 0, // User name
			'a', 'r', 'c', 'h', 'e', 'r', 0, // Character name
		}))
		err = session1.JoinLobby(ctx)
		if err != nil {
			t.Errorf("failed to join lobby: %v", err)
			return
		}
		err = bd1.RegisterNewObserver(ctx, session1)
		if err != nil {
			t.Errorf("failed to register new observer: %v", err)
			return
		}

		t.Log("Host has selected the character")
	})

	t.Run("Host creates a game room", func(t *testing.T) {
		// Create new game room
		assert.NoError(t, bd1.HandleCreateGame(ctx, session1, backend.CreateGameRequest{
			0, 0, 0, 0, // State
			byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0, // Map ID
			'r', 'o', 'o', 'm', 0, // Game room name
			0, // Password
		}))
		assert.NoError(t, bd1.HandleCreateGame(ctx, session1, backend.CreateGameRequest{
			1, 0, 0, 0, // State
			byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0, // Map ID
			'r', 'o', 'o', 'm', 0, // Game room name
			0, // Password
		}))

		if !handleMultiplayerMessage(ctx, cs) {
			t.Error("Failed to handle a message")
		}

		room, ok := cs.Multiplayer.GetRoom("room")
		if !ok {
			t.Errorf("failed to find room")
			return
		}
		if !room.Ready {
			t.Errorf("failed to create new room - it is unready")
			return
		}
		assert.Equal(t, "room", room.Name)
		assert.Equal(t, session1.UserID, room.CreatedBy.UserID)
		assert.Equal(t, session1.UserID, room.HostPlayer.UserID)
		assert.Equal(t, 1, len(room.Players))
		assert.Equal(t, session1.UserID, room.Players[1].UserID)
		assert.Equal(t, "archer", room.Players[1].User.Username)
		assert.Equal(t, byte(v1.ClassType_Archer), room.Players[1].Character.ClassType)

		t.Log("Host has created a game room")
	})

	// Other user
	conn2 := &mockConn{}

	proxy2 := &direct.ProxyLAN{"198.51.100.2"}
	bd2 := backend.NewBackend("", ts.URL, proxy2)
	bd2.SignalServerURL = "ws://" + cs.ConsoleBindAddr + "/lobby"

	session2 := bd2.AddSession(conn2)

	t.Run("Guest user signs in and selects the character", func(t *testing.T) {

		// Sign-in by player2
		assert.NoError(t, bd2.HandleClientAuthentication(ctx, session2, backend.ClientAuthenticationRequest{
			2, 0, 0, 0, // Unknown
			't', 'e', 's', 't', 0, // Password
			'm', 'a', 'g', 'e', 0, // Username
		}))
		if !bytes.Equal([]byte{255, 41, 8, 0, 1, 0, 0, 0}, conn2.Written) {
			t.Errorf("Not logged in, got: %v", conn2.Written)
			return
		}

		t.Log("Guest user authenticated")

		// Select character by player2
		assert.NoError(t, bd2.HandleSelectCharacter(ctx, session2, backend.SelectCharacterRequest{
			'm', 'a', 'g', 'e', 0, // User name
			'm', 'a', 'g', 'e', 0, // Character name
		}))
		err = session2.JoinLobby(ctx)
		if err != nil {
			t.Errorf("failed to join lobby: %v", err)
			return
		}
		err = bd2.RegisterNewObserver(ctx, session2)
		if err != nil {
			t.Errorf("failed to register new observer: %v", err)
			return
		}

		t.Log("Guest user has selected the character")
	})

	t.Run("Guest user joins the game room", func(t *testing.T) {
		// List games
		conn2.Written = nil // Truncate
		assert.NoError(t, bd2.HandleListGames(ctx, session2, backend.ListGamesRequest{}))

		// Check if user has received the game list with corresponding payload
		assert.Equal(t, []byte{
			1, 0, 0, 0, // Number of games
			198, 51, 100, 1, // IP address of host
			'r', 'o', 'o', 'm', 0, // Room name
			0, // Password
		}, findPacket(conn2.Written, packet.ListGames))

		// Select game
		conn2.Written = nil // Truncate
		assert.NoError(t, bd2.HandleSelectGame(ctx, session2, backend.SelectGameRequest{
			'r', 'o', 'o', 'm', 0, // Game name
			0, // Password
		}))

		// Check if the game is correct
		assert.Equal(t, []byte{
			byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0, // Map ID
			byte(v1.ClassType_Archer), 0, 0, 0, // Host's character class type
			198, 51, 100, 1, // IP address of host
			'a', 'r', 'c', 'h', 'e', 'r', 0, // Player name
		}, findPacket(conn2.Written, packet.SelectGame))

		conn2.Written = nil // Truncate

		// Join to host
		assert.NoError(t, bd2.HandleJoinGame(ctx, session2, backend.JoinGameRequest{
			'r', 'o', 'o', 'm', 0, // Game name
			0, // Password
		}))

		// Ensure the response is correct
		assert.Equal(t, []byte{
			model.GameStateStarted, 0, // Game state
			byte(v1.ClassType_Archer), 0, 0, 0, // Host's character class type
			198, 51, 100, 1, // IP address of host
			'a', 'r', 'c', 'h', 'e', 'r', 0, // Player name
		}, findPacket(conn2.Written, packet.JoinGame))

		t.Log("Guest user has joined the game")
	})

	t.Run("Ensure the response is correct", func(t *testing.T) {
		// Room contains all data
		room, ok := cs.Multiplayer.GetRoom("room")
		if !ok {
			t.Errorf("failed to find room")
			return
		}
		if !room.Ready {
			t.Errorf("failed to join room - it is unready")
			return
		}
		assert.Equal(t, "room", room.Name)
		assert.Equal(t, session1.UserID, room.CreatedBy.UserID)
		assert.Equal(t, session1.UserID, room.HostPlayer.UserID)
		assert.Equal(t, 2, len(room.Players))
		assert.Equal(t, session1.UserID, room.Players[1].UserID)
		assert.Equal(t, "archer", room.Players[1].User.Username)
		assert.Equal(t, byte(v1.ClassType_Archer), room.Players[1].Character.ClassType)
		assert.Equal(t, session2.UserID, room.Players[2].UserID)
		assert.Equal(t, "mage", room.Players[2].User.Username)
		assert.Equal(t, byte(v1.ClassType_Mage), room.Players[2].Character.ClassType)

		mpSession1, ok := cs.Multiplayer.GetUserSession(1)
		assert.True(t, ok)
		assert.Equal(t, session1.UserID, mpSession1.UserID)
		assert.Equal(t, "room", mpSession1.GameID)

		mpSession2, ok := cs.Multiplayer.GetUserSession(2)
		assert.True(t, ok)
		assert.Equal(t, session2.UserID, mpSession2.UserID)
		assert.Equal(t, "room", mpSession2.GameID)

		// Host user has correct data
		assert.Equal(t, int64(1), mpSession1.UserID)
		assert.Equal(t, "archer", mpSession1.User.Username)
		assert.Equal(t, "198.51.100.1", mpSession1.IPAddress)

		// Joining user has also the same data
		assert.Equal(t, int64(2), mpSession2.UserID)
		assert.Equal(t, "mage", mpSession2.User.Username)
		assert.Equal(t, "198.51.100.2", mpSession2.IPAddress)
	})

	t.Run("Ensure there are no unhandled messages", func(t *testing.T) {
		close(cs.Multiplayer.Messages)
		for message := range cs.Multiplayer.Messages {
			t.Error("unhandled message", message)
		}
	})
}

func findPacket(buf []byte, packetType packet.Code) []byte {
	for _, payload := range packet.Split(buf) {
		if len(payload) == 0 {
			// TODO: Why it happens?
			slog.Error("failed to split packet", "buffer", buf)
			return nil
		}
		pt := packet.Code(payload[1])
		if pt == packetType {
			return payload[4:]
		}
	}
	panic("not found")
}

func handleMultiplayerMessage(ctx context.Context, cs *console.Console) bool {
	timeout := time.After(time.Second)
	select {
	case <-ctx.Done():
		return false
	case <-timeout:
		return false
	case msg := <-cs.Multiplayer.Messages:
		cs.Multiplayer.HandleIncomingMessage(ctx, msg)
		return true
	}
}
