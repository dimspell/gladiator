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

// TestWebRTCGameExchange proves a real, dockerized two-player session can
// connect through the lobby/room phase and exchange actual game packets
// (UDP :6113 + TCP :6114) over the webrtc-beta (P2P) proxy.
//
// Topology: 1 console (webrtc-beta mode, no relay server) + 2 backend
// containers on a fixed-subnet network with static IPs. Each backend
// connects to the console's WebSocket lobby for signaling. Mock clients
// run inside the backend containers (sharing their network namespace).
// Game traffic flows through WebRTC data channels:
//
//   client -> 127.0.0.2:6113/6114 (fake-host) -> backend webrtc client
//   -> WebRTC data channel (T/U prefix) -> peer's backend
//   -> peer's 127.0.0.1:6113/6114 (guest's game server)
//
// Both sides listen on 127.0.0.1 and send to 127.0.0.2 (the first
// HostManager-assigned fake-host IP, which is symmetric in a 2-player
// game).
func TestWebRTCGameExchange(t *testing.T) {
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

	netName := "gladiator-webrtc-" + strings.ToLower(t.Name())
	net := newNetwork(t, ctx, netName)

	// Console in webrtc-beta mode. No --relay-addr (no QUIC relay server).
	consoleC, consoleName := startConsole(t, ctx, net, fd, "webrtc-beta", false)
	_ = consoleC

	// Backends: --proxy=webrtc-beta (no --relay-addr).
	backendA := startBackend(t, ctx, net, fd, consoleName, "webrtc-beta", hostIP, false)
	backendB := startBackend(t, ctx, net, fd, consoleName, "webrtc-beta", guestIP, false)

	hostEnv := map[string]string{
		"ROLE":         "host",
		"USERNAME":     "archer",
		"ROOM":         "room",
		"MY_IP":        "127.0.0.1",
		"PEER_IP":      "127.0.0.2",
		"RELAY_MODE":   "1",
		"BACKEND_ADDR": "127.0.0.1:" + backendPort,
	}
	guestEnv := map[string]string{
		"ROLE":         "guest",
		"USERNAME":     "mage",
		"ROOM":         "room",
		"MY_IP":        "127.0.0.1",
		"PEER_IP":      "127.0.0.2",
		"RELAY_MODE":   "1",
		"BACKEND_ADDR": "127.0.0.1:" + backendPort,
	}

	// Host runs in the background: creates the room, then listens and
	// exchanges game packets through the WebRTC data channel.
	var wg sync.WaitGroup
	var hostOut string
	var hostCode int
	wg.Add(1)
	go func() {
		defer wg.Done()
		hostOut, hostCode = runMockClient(t, ctx, backendA, hostEnv, 90*time.Second)
	}()

	// Let the host create the room and the WebRTC offer propagate before
	// the guest joins.
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
