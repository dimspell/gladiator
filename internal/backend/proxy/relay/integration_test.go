package relay

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
)

const integrationRelayAddr = "127.0.0.1:9911"

// clusterPlayer bundles a backend session, its lobby user session, the Relay
// proxy client, and (optionally) a capture sink for everything the player's
// fake hosts receive.
type clusterPlayer struct {
	session    *bsession.Session
	userSession *console.UserSession
	relay      *Relay
	cap         *captureRedirect
}

func newClusterPlayer(t *testing.T, mp *console.RoomService, client *console.GameService, userID int64, username string, capture bool) *clusterPlayer {
	t.Helper()

	session := &bsession.Session{
		ID:          username + "-session",
		UserID:      userID,
		Username:    username,
		CharacterID: userID,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}
	us := &console.UserSession{
		UserID:      userID,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: userID, Username: username},
		Character:   wire.Character{CharacterID: userID, ClassType: byte(model.ClassTypeKnight)},
	}
	mp.AddUserSession(userID, us)

	var relay *Relay
	if capture {
		cap := &captureRedirect{}
		relay = NewRelay(&ProxyRelay{
			RelayServerAddr: integrationRelayAddr,
			ManagerOptions: []func(*redirect.HostManager){
				redirect.WithProxyFactory(&captureFactory{shared: cap}),
				redirect.WithDisabledLogger(),
			},
		}, client, session)
		p := &clusterPlayer{session: session, userSession: us, relay: relay, cap: cap}
		session.Proxy = relay
		return p
	}

	relay = NewRelay(&ProxyRelay{RelayServerAddr: integrationRelayAddr}, client, session)
	session.Proxy = relay
	return &clusterPlayer{session: session, userSession: us, relay: relay}
}

func waitFor(t *testing.T, msg string, cond func() bool) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		if cond() {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for: %s", msg)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// TestCluster drives the full multi-user relay flow end-to-end against a real
// RelayServer (loopback QUIC, no game binary, no external services):
//   host creates room -> 3 guests join -> message exchange ->
//   one guest leaves (cleanup) -> host leaves (host migration).
func TestCluster(t *testing.T) {
	logger.SetPlainTextLogger(os.Stderr, slog.LevelWarn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomID := "clusterRoom"
	mp := console.NewRoomService()
	relayServer, err := console.NewQUICRelay(integrationRelayAddr, mp)
	require.NoError(t, err)
	go mp.Run(ctx)
	go relayServer.Start(ctx)

	client := &console.GameService{RoomService: mp}

	// --- Host creates the room ---
	host := newClusterPlayer(t, mp, client, 7001, "host", false)
	require.NoError(t, host.relay.CreateRoom(ctx, proxy.CreateParams{GameID: roomID}))
	mp.SetRoomReady(wire.Message{Content: roomID})

	// --- Three guests join ---
	guests := make([]*clusterPlayer, 3)
	guestNames := []string{"guest1", "guest2", "guest3"}
	for i, name := range guestNames {
		g := newClusterPlayer(t, mp, client, int64(7101+i), name, true)
		if _, _, err := g.relay.GetGame(ctx, roomID); err != nil {
			t.Fatalf("%s failed to get game: %v", name, err)
		}
		if _, err := g.relay.JoinGame(ctx, roomID, ""); err != nil {
			t.Fatalf("%s failed to join: %v", name, err)
		}
		guests[i] = g
	}

	// 1) All four players are present and the host is the host.
	room, ok := mp.GetRoom(roomID)
	require.True(t, ok, "room should exist")
	require.Len(t, room.Players, 4, "expected 4 players in room")
	assert.Equal(t, host.session.UserID, room.HostPlayer.UserID, "host should be the host")

	// 2) Message exchange: host -> guest1 is relayed through the server.
	require.NoError(t, host.relay.router.sendPacket(RelayPacket{
		Type:    "tcp",
		RoomID:  roomID,
		ToID:    remoteID(guests[0].session.UserID),
		Payload: []byte("hi-guest1"),
	}))
	waitFor(t, "guest1 receives host message", func() bool {
		return bytes.Contains(guests[0].cap.Bytes(), []byte("hi-guest1"))
	})

	// 3) One guest leaves: host stays host, room shrinks, resources cleaned.
	mp.LeaveRoom(ctx, guests[2].userSession)
	room, ok = mp.GetRoom(roomID)
	require.True(t, ok)
	assert.Len(t, room.Players, 3, "expected 3 players after one leave")
	assert.Equal(t, host.session.UserID, room.HostPlayer.UserID, "host unchanged after guest leave")

	// 4) Host leaves: RoomService migrates to a remaining guest.
	mp.LeaveRoom(ctx, host.userSession)
	room, ok = mp.GetRoom(roomID)
	require.True(t, ok, "room should survive with remaining players")
	assert.Len(t, room.Players, 2, "expected 2 players after host leave")
	assert.NotEqual(t, host.session.UserID, room.HostPlayer.UserID, "host should have migrated")

	cancel()
	time.Sleep(100 * time.Millisecond)
	host.relay.Close()
	for _, g := range guests {
		g.relay.Close()
	}
}
