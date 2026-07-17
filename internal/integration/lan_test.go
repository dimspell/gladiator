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

// TestLANGameExchange proves a real, dockerized two-player session can
// connect through the lobby/room phase and exchange actual game packets
// (UDP :6113 + TCP :6114) over the LAN proxy.
//
// Topology: 1 console + 2 backend containers on a fixed-subnet network
// with static IPs. Each backend container also runs the mock client
// (via Exec, sharing its network namespace). The LAN game traffic is
// direct peer-to-peer: the host listens on its own container IP and the
// guest sends to it; the host learns the guest's address from the
// incoming handshake source and replies, so a full bidirectional
// exchange is verified.
func TestLANGameExchange(t *testing.T) {
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

	netName := "gladiator-lan-" + strings.ToLower(t.Name())
	net := newNetwork(t, ctx, netName)

	consoleC, consoleName := startConsole(t, ctx, net, fd, "lan", false)
	_ = consoleC

	backendA := startBackend(t, ctx, net, fd, consoleName, "lan", hostIP, false)
	backendB := startBackend(t, ctx, net, fd, consoleName, "lan", guestIP, false)

	hostEnv := map[string]string{
		"ROLE":         "host",
		"USERNAME":    "archer",
		"ROOM":         "room",
		"MY_IP":        hostIP,
		"BACKEND_ADDR":  "127.0.0.1:" + backendPort,
	}
	guestEnv := map[string]string{
		"ROLE":         "guest",
		"USERNAME":    "mage",
		"ROOM":         "room",
		"MY_IP":        guestIP,
		"PEER_IP":      hostIP,
		"BACKEND_ADDR":  "127.0.0.1:" + backendPort,
	}

	// Host runs in the background: it creates the room and then blocks
	// listening for the guest's game packets.
	var wg sync.WaitGroup
	var hostOut string
	var hostCode int
	wg.Add(1)
	go func() {
		defer wg.Done()
		hostOut, hostCode = runMockClient(t, ctx, backendA, hostEnv, 90*time.Second)
	}()

	// Let the host create the room and start listening before the guest joins.
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

// TestLAN4PlayerGameExchange proves a 4-player session (host + 3 guests
// simultaneously in the room) can exchange game packets (UDP :6113 + TCP
// :6114) over the LAN proxy. All guests join in parallel so the room
// holds 4 players; the host accepts connections from all of them.
func TestLAN4PlayerGameExchange(t *testing.T) {
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

	netName := "gladiator-lan4p-" + strings.ToLower(t.Name())
	net := newNetwork(t, ctx, netName)

	consoleC, consoleName := startConsole(t, ctx, net, fd, "lan", false)
	_ = consoleC

	backendHost := startBackend(t, ctx, net, fd, consoleName, "lan", hostIP, false)
	backendG1  := startBackend(t, ctx, net, fd, consoleName, "lan", guestIP, false)
	backendG2  := startBackend(t, ctx, net, fd, consoleName, "lan", guest2IP, false)
	backendG3  := startBackend(t, ctx, net, fd, consoleName, "lan", guest3IP, false)

	hostEnv := map[string]string{
		"ROLE":            "host",
		"USERNAME":        "archer",
		"ROOM":            "room",
		"MY_IP":           hostIP,
		"BACKEND_ADDR":    "127.0.0.1:" + backendPort,
		"MOCK_NUM_PLAYERS": "4",
	}
	guestEnv := func(name string) map[string]string {
		return map[string]string{
			"ROLE":         "guest",
			"USERNAME":     name,
			"ROOM":         "room",
			"MY_IP":        "127.0.0.1",
			"PEER_IP":      hostIP,
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
	// simultaneously connected to the host.
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
	}
	hostWg.Wait()

	require.Equal(t, 0, hostCode, "host mock client failed (code=%d):\n%s", hostCode, hostOut)
	require.Contains(t, hostOut, "GAME_PACKET_OK")
}
