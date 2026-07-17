//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

// TestRelayGameExchange proves a real, dockerized two-player session can
// connect through the lobby/room phase and exchange actual game packets
// (UDP :6113 + TCP :6114) over the relay-beta proxy.
//
// Topology: 1 console (with built-in QUIC relay server) + 2 backend
// containers on a fixed-subnet network with static IPs. Each backend
// connects to the console's relay server via QUIC. Mock clients run
// inside the backend containers (sharing their network namespace).
// Game traffic flows through the relay:
//
//	client -> 127.0.0.2:6113/6114 (fake-host) -> backend relay client
//	-> QUIC relay server -> peer's backend -> peer's 127.0.0.1:6113/6114
//
// Both sides listen on 127.0.0.1 and send to 127.0.0.2 (the first
// HostManager-assigned fake-host IP, which is symmetric in a 2-player
// game).
func TestRelayGameExchange(t *testing.T) {
	if os.Getenv("SKIP_DOCKER") != "" {
		t.Skip("SKIP_DOCKER set")
	}
	ctx := context.Background()
	repoRoot := findRepoRoot(t)
	fd := testcontainers.FromDockerfile{
		Context:    repoRoot,
		Dockerfile: "Dockerfile.integration",
		KeepImage:  true,
	}

	netName := "gladiator-relay-" + strings.ToLower(t.Name())
	net := newNetwork(t, ctx, netName)

	// Console with built-in QUIC relay. --relay-addr implies --run-mode=relay-beta.
	consoleC, consoleName := startConsole(t, ctx, net, fd, "relay-beta", true)
	_ = consoleC

	// Backends: --proxy=relay-beta, --relay-addr=<console>:9999.
	// --lan-my-ip-addr is NOT passed (only used for LAN proxy).
	backendA := startBackend(t, ctx, net, fd, consoleName, "relay-beta", hostIP, true)
	backendB := startBackend(t, ctx, net, fd, consoleName, "relay-beta", guestIP, true)

	hostEnv := map[string]string{
		"ROLE":         "host",
		"USERNAME":    "archer",
		"ROOM":         "room",
		"MY_IP":        "127.0.0.1",
		"PEER_IP":      "127.0.0.2",
		"RELAY_MODE":   "1",
		"BACKEND_ADDR":  "127.0.0.1:" + backendPort,
	}
	guestEnv := map[string]string{
		"ROLE":         "guest",
		"USERNAME":    "mage",
		"ROOM":         "room",
		"MY_IP":        "127.0.0.1",
		"PEER_IP":      "127.0.0.2",
		"RELAY_MODE":   "1",
		"BACKEND_ADDR":  "127.0.0.1:" + backendPort,
	}

	// Host runs in the background: creates the room, then listens and
	// exchanges game packets through the relay.
	var wg sync.WaitGroup
	var hostOut string
	var hostCode int
	wg.Add(1)
	go func() {
		defer wg.Done()
		hostOut, hostCode = runMockClient(t, ctx, backendA, hostEnv, 90*time.Second)
	}()

	// Let the host create the room and the relay path set up before the
	// guest joins.
	time.Sleep(5 * time.Second)

	guestOut, guestCode := runMockClient(t, ctx, backendB, guestEnv, 90*time.Second)
	wg.Wait()

	if guestCode != 0 || !strings.Contains(guestOut, "GAME_PACKET_OK") {
		dumpLogs(t, ctx, backendB, "guest-backend")
		dumpLogs(t, ctx, backendA, "host-backend")
		t.Logf("HOST mock client output (code=%d):\n%s", hostCode, hostOut)
		t.Fatalf("guest mock client failed (code=%d):\n%s", guestCode, guestOut)
	}
	require.Contains(t, guestOut, "GAME_PACKET_EXCHANGED_UDP")
	require.Contains(t, guestOut, "GAME_PACKET_EXCHANGED_TCP")

	if hostCode != 0 || !strings.Contains(hostOut, "GAME_PACKET_OK") {
		dumpLogs(t, ctx, backendA, "host-backend")
		t.Fatalf("host mock client failed (code=%d):\n%s", hostCode, hostOut)
	}
}

// TestRelay4PlayerGameExchange proves a 4-player session (host + 3 guests
// simultaneously in the room) can exchange game packets (UDP :6113 + TCP
// :6114) over the relay-beta proxy. All guests join in parallel so the
// room holds 4 players; the host accepts connections from all of them
// through the QUIC relay server.
func TestRelay4PlayerGameExchange(t *testing.T) {
	if os.Getenv("SKIP_DOCKER") != "" {
		t.Skip("SKIP_DOCKER set")
	}
	ctx := context.Background()
	repoRoot := findRepoRoot(t)
	fd := testcontainers.FromDockerfile{
		Context:    repoRoot,
		Dockerfile: "Dockerfile.integration",
		KeepImage:  true,
	}

	netName := "gladiator-relay4p-" + strings.ToLower(t.Name())
	net := newNetwork(t, ctx, netName)

	consoleC, consoleName := startConsole(t, ctx, net, fd, "relay-beta", true)
	_ = consoleC

	backendHost := startBackend(t, ctx, net, fd, consoleName, "relay-beta", hostIP, true)
	backendG1   := startBackend(t, ctx, net, fd, consoleName, "relay-beta", guestIP, true)
	backendG2   := startBackend(t, ctx, net, fd, consoleName, "relay-beta", guest2IP, true)
	backendG3   := startBackend(t, ctx, net, fd, consoleName, "relay-beta", guest3IP, true)

	hostEnv := map[string]string{
		"ROLE":            "host",
		"USERNAME":        "archer",
		"ROOM":            "room",
		"MY_IP":           "127.0.0.1",
		"PEER_IP":         "127.0.0.2",
		"RELAY_MODE":      "1",
		"BACKEND_ADDR":    "127.0.0.1:" + backendPort,
		"MOCK_NUM_PLAYERS": "4",
	}
	guestEnv := func(name string) map[string]string {
		return map[string]string{
			"ROLE":         "guest",
			"USERNAME":     name,
			"ROOM":         "room",
			"MY_IP":        "127.0.0.1",
			"PEER_IP":      "127.0.0.2",
			"RELAY_MODE":   "1",
			"BACKEND_ADDR": "127.0.0.1:" + backendPort,
		}
	}

	var hostWg sync.WaitGroup
	var hostOut string
	var hostCode int
	hostWg.Add(1)
	go func() {
		defer hostWg.Done()
		hostOut, hostCode = runMockClient(t, ctx, backendHost, hostEnv, 120*time.Second)
	}()
	time.Sleep(5 * time.Second)

	// All guests join the room and exchange in parallel so they are
	// simultaneously connected to the host through the relay.
	var guestWg sync.WaitGroup
	type gres struct {
		name string
		out  string
		code int
	}
	results := make(chan gres, 3)
	guests := []struct {
		b    testcontainers.Container
		name string
	}{
		{backendG1, "mage"},
		{backendG2, "warrior"},
		{backendG3, "necro"},
	}
	for _, g := range guests {
		guestWg.Add(1)
		g := g
		go func() {
			defer guestWg.Done()
			out, code := runMockClient(t, ctx, g.b, guestEnv(g.name), 90*time.Second)
			results <- gres{g.name, out, code}
		}()
	}
	guestWg.Wait()
	close(results)

	for r := range results {
		require.Equalf(t, 0, r.code, "guest %s mock client failed (code=%d):\n%s", r.name, r.code, r.out)
		require.Containsf(t, r.out, "GAME_PACKET_OK", "guest %s did not exchange ok:\n%s", r.name, r.out)
		require.Containsf(t, r.out, "GAME_PACKET_EXCHANGED_UDP", "guest %s UDP failed:\n%s", r.name, r.out)
		require.Containsf(t, r.out, "GAME_PACKET_EXCHANGED_TCP", "guest %s TCP failed:\n%s", r.name, r.out)
	}
	hostWg.Wait()

	require.Equal(t, 0, hostCode, "host mock client failed (code=%d):\n%s", hostCode, hostOut)
	require.Contains(t, hostOut, "GAME_PACKET_OK")
}
