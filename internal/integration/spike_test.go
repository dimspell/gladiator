//go:build integration

package integration

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// findRepoRoot walks up from the current working directory to the directory
// containing go.mod (the module root), so the Docker build context is correct
// regardless of where `go test` is invoked from.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not find repo root (go.mod) from %s", dir)
	return ""
}

// runMockClient executes the mock client inside the given backend container
// (sharing its network namespace) and returns its combined output.
func runMockClient(t *testing.T, ctx context.Context, c testcontainers.Container, args ...string) string {
	t.Helper()
	cmd := append([]string{"/mockclient"}, args...)
	code, reader, err := c.Exec(ctx, cmd)
	if err != nil {
		dumpLogs(t, ctx, c, "mockclient")
		t.Fatalf("exec mockclient: %v", err)
	}
	out, _ := io.ReadAll(reader)
	t.Logf("mockclient exit=%d: %s", code, string(out))
	return string(out)
}

// dumpLogs prints a container's logs to the test log.
func dumpLogs(t *testing.T, ctx context.Context, c testcontainers.Container, label string) {
	t.Helper()
	r, err := c.Logs(ctx)
	if err != nil {
		t.Logf("[%s] cannot read logs: %v", label, err)
		return
	}
	defer r.Close()
	data, _ := io.ReadAll(r)
	t.Logf("[%s] logs:\n%s", label, string(data))
}

// TestSpike validates the multi-docker topology: a console container and a
// backend container, with the mock client executed inside the backend container
// (so it shares the backend's network namespace and can reach loopback fake
// hosts). It proves the image builds, containers start, the backend reaches the
// console, and the mock client reaches both the backend and a loopback address.
func TestSpike(t *testing.T) {
	if os.Getenv("SKIP_DOCKER") != "" {
		t.Skip("SKIP_DOCKER set")
	}
	ctx := context.Background()
	repoRoot := findRepoRoot(t)

	netName := "gladiator-it-" + strings.ToLower(t.Name())
	network, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{Name: netName},
	})
	if err != nil {
		t.Fatalf("create network: %v", err)
	}
	t.Cleanup(func() { _ = network.Remove(ctx) })

	fromDockerfile := testcontainers.FromDockerfile{
		Context:    repoRoot,
		Dockerfile: "Dockerfile.integration",
		KeepImage:  true,
	}

	// --- Console container (relay enabled) ---
	consoleC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: fromDockerfile,
			ExposedPorts:   []string{"2137/tcp", "9999/udp"},
			Networks:       []string{netName},
			Cmd:            []string{"console", "--console-addr=0.0.0.0:2137", "--relay-addr=0.0.0.0:9999"},
			WaitingFor:     wait.ForHTTP("/.well-known/console.json").WithPort("2137/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start console: %v", err)
	}
	t.Cleanup(func() { _ = consoleC.Terminate(ctx) })

	consoleName, err := consoleC.Name(ctx)
	if err != nil {
		t.Fatalf("console name: %v", err)
	}
	consoleName = strings.TrimPrefix(consoleName, "/")
	t.Logf("console container: %s", consoleName)

	// --- Backend container (relay mode, points at console) ---
	backendC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: fromDockerfile,
			ExposedPorts:   []string{"6112/tcp"},
			Networks:       []string{netName},
			Env:            map[string]string{"BACKEND_ADDR": "127.0.0.1:6112"},
			Cmd: []string{
				"backend",
				"--console-addr=http://" + consoleName + ":2137",
				"--backend-addr=0.0.0.0:6112",
				"--proxy=relay-beta",
				"--relay-addr=" + consoleName + ":9999",
			},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start backend: %v", err)
	}
	t.Cleanup(func() { _ = backendC.Terminate(ctx) })

	// Give the backend a moment to fetch console metadata + pass the mode check.
	time.Sleep(3 * time.Second)
	dumpLogs(t, ctx, backendC, "backend-startup")

	// --- Mock client executed inside the backend container ---
	out := runMockClient(t, ctx, backendC)
	if !strings.Contains(out, "INTEGRATION_OK") {
		t.Fatalf("mock client did not report INTEGRATION_OK; output:\n%s", out)
	}

	// Sanity: console metadata is reachable from the host too.
	host, err := consoleC.Host(ctx)
	if err != nil {
		t.Fatalf("console host: %v", err)
	}
	port, err := consoleC.MappedPort(ctx, "2137")
	if err != nil {
		t.Fatalf("mapped port: %v", err)
	}
	resp, err := http.Get(fmt.Sprintf("http://%s:%s/.well-known/console.json", host, port.Port()))
	if err != nil {
		t.Fatalf("console metadata: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("console metadata status: %d", resp.StatusCode)
	}
	_ = bufio.NewReader(resp.Body)

	t.Log("spike OK: image builds, console+backend+mockclient topology validated")
}
