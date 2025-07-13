package action

import "fmt"

var (
	// Backend
	defaultBackendAddr = "127.0.0.1:6112"

	// Proxy config
	defaultProxyType = "lan"

	// For LAN Proxy
	defaultMyIPAddr = "127.0.0.1"

	// For Relay Proxy
	defaultRelayAddr = "127.0.0.1:9999"
)

var (
	// Console
	defaultConsoleAddr       = "127.0.0.1:2137"
	defaultPublicConsoleAddr = fmt.Sprintf("http://%s", defaultConsoleAddr)
	defaultLobbyAddr         = fmt.Sprintf("ws://%s/lobby", defaultConsoleAddr)

	// SQLite config
	defaultDatabasePath = "dispel-multi.sqlite"
	defaultDatabaseType = "memory"
)

var (
	// For TURN server
	defaultTurnPublicIP = "127.0.0.1"
)
