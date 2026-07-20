//go:build integration

package integration

import (
	"context"
	"fmt"
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

// TestRelay4PlayerGameExchange proves a 4-player session (1 host + 3 guests)
// can exchange game packets (UDP :6113 + TCP :6114) over the relay-beta
// proxy. Guests join concurrently to test that the relay proxy handles
// parallel StartGuest dialer setup and concurrent game-packet exchange.
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

	var wg sync.WaitGroup
	var hostOut string
	var hostCode int
	wg.Add(1)
	go func() {
		defer wg.Done()
		hostOut, hostCode = runMockClient(t, ctx, backendHost, hostEnv, 120*time.Second)
	}()
	time.Sleep(5 * time.Second)

	guests := []struct {
		name     string
		backend  testcontainers.Container
	}{
		{"mage", backendG1},
		{"warrior", backendG2},
		{"necro", backendG3},
	}
	var guestWg sync.WaitGroup
	var guestFailures []string
	var guestMu sync.Mutex
	for _, g := range guests {
		guestWg.Add(1)
		go func(name string, backend testcontainers.Container) {
			defer guestWg.Done()
			out, code := runMockClient(t, ctx, backend, guestEnv(name), 90*time.Second)
			if code != 0 {
				guestMu.Lock()
				guestFailures = append(guestFailures, fmt.Sprintf("guest %s (code=%d):\n%s", name, code, out))
				guestMu.Unlock()
			}
		}(g.name, g.backend)
	}
	guestWg.Wait()

	if len(guestFailures) > 0 {
		dumpLogs(t, ctx, consoleC, "console-relay")
		dumpLogs(t, ctx, backendHost, "host-backend")
		for _, g := range guests {
			dumpLogs(t, ctx, g.backend, "guest-"+g.name)
		}
		for _, f := range guestFailures {
			t.Logf("FAIL: %s", f)
		}
		t.Fatal("one or more guests failed")
	}
	wg.Wait()

	require.Equal(t, 0, hostCode, "host mock client failed (code=%d):\n%s", hostCode, hostOut)
	require.Contains(t, hostOut, "GAME_PACKET_OK")
}

// TestRelayFullLifecycleWithMigration proves a full relay-proxy lifecycle
// with host migration. Topology: 1 console (relay-beta) + 4 backends
// (relay-beta). Host A (archer) creates the room, guests B (mage), C (warrior),
// and D (necro) join. B leaves mid-game, then A leaves, triggering host
// migration to C. Survivors (C, D) re-exchange after migration.
func TestRelayFullLifecycleWithMigration(t *testing.T) {
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

	netName := "gladiator-lifecycle-" + strings.ToLower(t.Name())
	net := newNetwork(t, ctx, netName)

	// Phase 1: Console + 4 backends (relay-beta, all with relay enabled)
	consoleC, consoleName := startConsole(t, ctx, net, fd, "relay-beta", true)
	_ = consoleC

	backendA := startBackend(t, ctx, net, fd, consoleName, "relay-beta", hostIP, true)
	backendB := startBackend(t, ctx, net, fd, consoleName, "relay-beta", guestIP, true)
	backendC := startBackend(t, ctx, net, fd, consoleName, "relay-beta", guest2IP, true)
	backendD := startBackend(t, ctx, net, fd, consoleName, "relay-beta", guest3IP, true)

	// Phase 2: Host A (archer) — background goroutine, stays ~90s then leaves.
	hostEnv := map[string]string{
		"ROLE":             "host",
		"USERNAME":         "archer",
		"ROOM":             "room",
		"MY_IP":            "127.0.0.1",
		"RELAY_MODE":       "1",
		"BACKEND_ADDR":     "127.0.0.1:" + backendPort,
		"MOCK_NUM_PLAYERS": "4",
		"LEAVE_AFTER":      "90",
	}
	var wg sync.WaitGroup
	var hostOut string
	var hostCode int
	wg.Add(1)
	go func() {
		defer wg.Done()
		hostOut, hostCode = runMockClient(t, ctx, backendA, hostEnv, 180*time.Second)
	}()

	// Phase 3: Wait for room ready on the console.
	time.Sleep(5 * time.Second)

	// Phase 4: Guests B, C, D join concurrently.
	guestBEnv := map[string]string{
		"ROLE":        "guest",
		"USERNAME":    "mage",
		"ROOM":        "room",
		"MY_IP":       "127.0.0.1",
		"PEER_IPS":    "127.0.0.2",
		"RELAY_MODE":  "1",
		"BACKEND_ADDR": "127.0.0.1:" + backendPort,
		"LEAVE_AFTER": "40",
	}
	guestCEnv := map[string]string{
		"ROLE":        "guest",
		"USERNAME":    "warrior",
		"ROOM":        "room",
		"MY_IP":       "127.0.0.1",
		"PEER_IPS":    "127.0.0.2",
		"RELAY_MODE":  "1",
		"BACKEND_ADDR": "127.0.0.1:" + backendPort,
	}
	guestDEnv := map[string]string{
		"ROLE":        "guest",
		"USERNAME":    "necro",
		"ROOM":        "room",
		"MY_IP":       "127.0.0.1",
		"PEER_IPS":    "127.0.0.2",
		"RELAY_MODE":  "1",
		"BACKEND_ADDR": "127.0.0.1:" + backendPort,
	}

	var (
		guestBOut  string
		guestBCode int
		guestCOut  string
		guestCCode int
		guestDOut  string
		guestDCode int
	)

	// B uses its own WaitGroup so we can wait for LEAVE_AFTER=40 to fire.
	var bWg sync.WaitGroup
	bWg.Add(1)
	go func() {
		defer bWg.Done()
		guestBOut, guestBCode = runMockClient(t, ctx, backendB, guestBEnv, 60*time.Second)
	}()

	// C and D run until their 180s timeout (survivors).
	var guestWg sync.WaitGroup
	guestWg.Add(1)
	go func() {
		defer guestWg.Done()
		guestCOut, guestCCode = runMockClient(t, ctx, backendC, guestCEnv, 180*time.Second)
	}()
	guestWg.Add(1)
	go func() {
		defer guestWg.Done()
		guestDOut, guestDCode = runMockClient(t, ctx, backendD, guestDEnv, 180*time.Second)
	}()

	// Phase 5: B leaves after ~40s (LEAVE_AFTER=40).
	bWg.Wait()
	if guestBCode != 0 || !strings.Contains(guestBOut, "GAME_PACKET_OK") {
		dumpLogs(t, ctx, backendB, "guest-mage")
		dumpLogs(t, ctx, consoleC, "console-relay")
		dumpLogs(t, ctx, backendA, "host-archer")
		t.Fatalf("guest B (mage) failed (code=%d):\n%s", guestBCode, guestBOut)
	}
	require.Contains(t, guestBOut, "GAME_PACKET_EXCHANGED_UDP", "B exchanged UDP before leaving")
	require.Contains(t, guestBOut, "GAME_PACKET_EXCHANGED_TCP", "B exchanged TCP before leaving")

	// Let the leave propagate through the relay.
	time.Sleep(5 * time.Second)

	// Phase 6: Survivors (C, D) keep exchanging with A — no explicit check yet.

	// Phase 7: A leaves after ~90s → the console migrates host to C.
	wg.Wait()
	require.Equal(t, 0, hostCode, "host A (archer) failed:\n%s", hostOut)
	require.Contains(t, hostOut, "GAME_PACKET_OK", "host A should have exchanged before leaving")

	// Sleep for migration to propagate to survivors.
	time.Sleep(5 * time.Second)

	// Dump C and D container logs for migration-event visibility.
	dumpLogs(t, ctx, backendC, "guest-warrior")
	dumpLogs(t, ctx, backendD, "guest-necro")

	// Wait for C and D to finish (180s timeout from Phase 4).
	guestWg.Wait()

	// Phase 8: Assertions on all outputs.
	var failures []string

	// Survivors must have exited 0 and exchanged successfully.
	if guestCCode != 0 || !strings.Contains(guestCOut, "GAME_PACKET_OK") {
		failures = append(failures, fmt.Sprintf("guest C (warrior) code=%d:\n%s", guestCCode, guestCOut))
	}
	if guestDCode != 0 || !strings.Contains(guestDOut, "GAME_PACKET_OK") {
		failures = append(failures, fmt.Sprintf("guest D (necro) code=%d:\n%s", guestDCode, guestDOut))
	}

	// Survivors must have exchanged UDP and TCP (initial round or after migration).
	if !strings.Contains(guestCOut, "GAME_PACKET_EXCHANGED_UDP") {
		failures = append(failures, "C did not exchange UDP")
	}
	if !strings.Contains(guestCOut, "GAME_PACKET_EXCHANGED_TCP") {
		failures = append(failures, "C did not exchange TCP")
	}
	if !strings.Contains(guestDOut, "GAME_PACKET_EXCHANGED_UDP") {
		failures = append(failures, "D did not exchange UDP")
	}
	if !strings.Contains(guestDOut, "GAME_PACKET_EXCHANGED_TCP") {
		failures = append(failures, "D did not exchange TCP")
	}

	// Host migration must be reported by at least one survivor.
	cMig := strings.Contains(guestCOut, "HOST_MIGRATION_TO") ||
		strings.Contains(guestCOut, "HOST_MIGRATION_SELF")
	dMig := strings.Contains(guestDOut, "HOST_MIGRATION_TO") ||
		strings.Contains(guestDOut, "HOST_MIGRATION_SELF")
	if !cMig && !dMig {
		failures = append(failures,
			"neither C nor D reported host migration (HOST_MIGRATION_TO or HOST_MIGRATION_SELF)")
	}

	if len(failures) > 0 {
		dumpLogs(t, ctx, consoleC, "console-relay")
		dumpLogs(t, ctx, backendA, "host-archer")
		dumpLogs(t, ctx, backendB, "guest-mage")
		dumpLogs(t, ctx, backendC, "guest-warrior")
		dumpLogs(t, ctx, backendD, "guest-necro")
		for _, f := range failures {
			t.Logf("FAIL: %s", f)
		}
		t.Fatal("survivor check failed; see FAIL lines above")
	}
}
