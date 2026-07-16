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
