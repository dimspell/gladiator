package action

var (
	// Backend
	defaultBackendAddr = "127.0.0.1:6112"
	defaultLobbyAddr   = "ws://127.0.0.1:2137/lobby"

	// Proxy config
	defaultProxyType = "lan"

	// For LAN Proxy
	defaultMyIPAddr = "127.0.0.1"

	// For Relay Proxy
	defaultRelayAddr = "127.0.0.1:9999"
)

var (
	// Console
	defaultConsoleAddr = "127.0.0.1:2137"

	// SQLite config
	defaultDatabasePath = "dispel-multi.sqlite"
	defaultDatabaseType = "memory"
)

var (
	// For TURN server
	defaultTurnPublicIP = "127.0.0.1"
)
