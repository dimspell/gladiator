package action

var (
	defaultConsoleAddr = "127.0.0.1:2137"
	defaultBackendAddr = "127.0.0.1:6112"
	defaultLobbyAddr   = "ws://127.0.0.1:2137/lobby"

	// SQLite config
	defaultDatabasePath = "dispel-multi.sqlite"
	defaultDatabaseType = "memory"

	// Proxy config
	defaultProxyType = "lan"

	// For LAN Proxy
	defaultMyIPAddr = "127.0.0.1"

	// For Relay Proxy
	defaultRelayAddr = "127.0.0.1:9999"

	// For TURN server
	defaultTurnPublicIP = "127.0.0.1"
)
