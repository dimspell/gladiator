//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
)

const (
	consolePort = "2137"
	relayPort   = "9999"
	backendPort = "6112"

	// Static IPs on the integration subnet. The backend's --lan-my-ip-addr
	// must be the container's own Docker IP (so the peer can reach it), but
	// that IP is only known after the container starts. We pin static IPs on
	// a fixed subnet so the value is known before start.
	subnet    = "172.28.0.0/16"
	hostIP    = "172.28.0.20"
	guestIP   = "172.28.0.21"
	guest2IP  = "172.28.0.22"
	guest3IP  = "172.28.0.23"
)

// newNetwork creates a user-defined bridge network with a fixed subnet so we
// can assign static container IPs. It returns the network name; cleanup is
// registered via t.Cleanup.
func newNetwork(t *testing.T, ctx context.Context, name string) string {
	t.Helper()
	net, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name: name,
			IPAM: &network.IPAM{
				Config: []network.IPAMConfig{{Subnet: netip.MustParsePrefix(subnet)}},
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = net.Remove(ctx) })
	return name
}

// startConsole starts a console container on the network. runMode is the
// advertised run mode (e.g. "lan"); withRelay starts the QUIC relay server.
func startConsole(t *testing.T, ctx context.Context, netName string, fd testcontainers.FromDockerfile, runMode string, withRelay bool) (testcontainers.Container, string) {
	t.Helper()

	args := []string{"console", "--console-addr=0.0.0.0:" + consolePort, "--database-type=memory"}
	if withRelay {
		args = append(args, "--relay-addr=0.0.0.0:"+relayPort)
	}
	if runMode != "" {
		args = append(args, "--run-mode="+runMode)
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: fd,
			ExposedPorts:  []string{consolePort + "/tcp", relayPort + "/udp"},
			Networks:       []string{netName},
			Cmd:            args,
			WaitingFor:     wait.ForHTTP("/.well-known/console.json").WithPort(consolePort + "/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Terminate(ctx) })

	name, err := c.Name(ctx)
	require.NoError(t, err)
	return c, strings.TrimPrefix(name, "/")
}

// startBackend starts a backend container on the network with a static IP.
// proxy is the --proxy value (e.g. "lan"); myIP is the static IP used for
// --lan-my-ip-addr; consoleName is the console container name (Docker DNS).
// The backend's run mode is derived from --proxy (it has no --run-mode flag).
func startBackend(t *testing.T, ctx context.Context, netName string, fd testcontainers.FromDockerfile, consoleName, proxy, myIP string, withRelay bool) testcontainers.Container {
	t.Helper()

	args := []string{
		"backend",
		"--console-addr=http://" + consoleName + ":" + consolePort,
		"--lobby-addr=ws://" + consoleName + ":" + consolePort + "/lobby",
		"--backend-addr=0.0.0.0:" + backendPort,
		"--proxy=" + proxy,
	}
	if proxy == "lan" {
		args = append(args, "--lan-my-ip-addr="+myIP)
	}
	if withRelay {
		args = append(args, "--relay-addr="+consoleName+":"+relayPort)
	}

	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: fd,
			ExposedPorts:  []string{backendPort + "/tcp"},
			Networks:       []string{netName},
			Cmd:            args,
			WaitingFor:     wait.ForListeningPort(backendPort + "/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	}
	require.NoError(t, testcontainers.WithEndpointSettingsModifier(func(settings map[string]*network.EndpointSettings) {
		if ep, ok := settings[netName]; ok && ep != nil {
			ep.IPAMConfig = &network.EndpointIPAMConfig{IPv4Address: netip.MustParseAddr(myIP)}
		}
	}).Customize(&req))

	c, err := testcontainers.GenericContainer(ctx, req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Terminate(ctx) })
	return c
}

// runMockClient executes /mockclient inside the backend container (sharing its
// network namespace) with the given environment, returning combined output and
// the exit code.
func runMockClient(t *testing.T, ctx context.Context, c testcontainers.Container, env map[string]string, timeout time.Duration) (string, int) {
	t.Helper()

	envSlice := []string{"TIMEOUT_SECONDS=" + fmt.Sprint(int(timeout.Seconds()))}
	for k, v := range env {
		envSlice = append(envSlice, k+"="+v)
	}

	code, reader, err := c.Exec(ctx, []string{"/mockclient"}, tcexec.WithEnv(envSlice))
	require.NoError(t, err)
	out, _ := io.ReadAll(reader)
	return string(out), code
}
