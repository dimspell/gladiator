package action

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/dimspell/gladiator/internal/backend/proxy/direct"
	"github.com/dimspell/gladiator/internal/backend/proxy/p2p"
	"github.com/dimspell/gladiator/internal/backend/proxy/relay"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/pion/webrtc/v4"
	"github.com/urfave/cli/v3"
)

func selectDatabaseType(c *cli.Command) (db *database.SQLite, err error) {
	switch c.String("database-type") {
	case "memory":
		db, err = database.NewMemory()
		if err != nil {
			return nil, err
		}
	case "sqlite":
		db, err = database.NewLocal(c.String("sqlite-path"))
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown database type: %q", c.String("database-type"))
	}

	if err := database.Seed(db.Write); err != nil {
		slog.Warn("Seed queries failed", logging.Error(err))
	}

	return db, nil
}

var (
	proxyTypeLAN   = model.RunModeLAN.String()
	proxyTypeRelay = model.RunModeRelay.String()
)

// proxyTypeWebRTC and its implementation in selectProxy are disabled.
// The p2p and webrtc imports are kept for reference via blank identifiers.
var (
	_ = p2p.ProxyP2P{}
	_ = webrtc.ICEServer{}
)

func selectProxy(c *cli.Command) (p backend.ProxyFactory, err error) {
	switch c.String("proxy") {
	case proxyTypeLAN:
		myIPAddr := c.String("lan-my-ip-addr")
		if ip := net.ParseIP(myIPAddr); ip == nil {
			return nil, fmt.Errorf("invalid lan-my-ip-addr: %q", myIPAddr)
		}
		return &direct.ProxyLAN{MyIPAddress: myIPAddr}, nil
	case model.RunModeWebRTC.String():
		// WebRTC proxy is disabled. Source retained for reference.
		// Previously returned &p2p.ProxyP2P{
		//   ICEServers: []webrtc.ICEServer{
		//     {URLs: []string{"stun:stun.l.google.com:19302"}},
		//     {URLs: []string{"turn:127.0.0.1:3478"}, Username: "username2", Credential: "password2"},
		//   },
		// }
		return nil, fmt.Errorf("proxy %q is disabled; use %q or %q",
			model.RunModeWebRTC, proxyTypeLAN, proxyTypeRelay)
	case proxyTypeRelay:
		relayAddr := c.String("relay-addr")
		return &relay.ProxyRelay{RelayServerAddr: relayAddr}, nil
	default:
		return nil, fmt.Errorf("unknown proxy: %q (valid: %s, %s)",
			c.String("proxy"), proxyTypeLAN, proxyTypeRelay)
	}
}

func selectConsoleOptions(c *cli.Command, version string) ([]console.Option, error) {
	var options []console.Option

	options = append(options, console.WithVersion(version))

	consoleBindAddr := c.String("console-addr")
	consolePublicAddr := fallbackString(c.String("console-public-addr"), fmt.Sprintf("http://%s", consoleBindAddr))
	options = append(options, console.WithConsoleAddr(consoleBindAddr, consolePublicAddr))

	if relayBindAddr := c.String("relay-addr"); relayBindAddr != "" {
		relayPublicAddr := fallbackString(c.String("relay-public-addr"), relayBindAddr)
		options = append(options, console.WithRelayAddr(relayBindAddr, relayPublicAddr))
	}

	if runMode := c.String("run-mode"); runMode != "" {
		if !isValidRunMode(runMode) {
			return nil, fmt.Errorf("unknown run-mode: %q (valid: lan, relay-beta, single)", runMode)
		}
		options = append(options, console.WithRunMode(model.RunMode(runMode)))
	}

	return options, nil
}

// isValidRunMode reports whether s is one of the selectable model.RunMode values.
// WebRTC and libp2p are disabled; the constants exist but their modes are rejected.
func isValidRunMode(s string) bool {
	switch model.RunMode(s) {
	case model.RunModeSinglePlayer, model.RunModeLAN, model.RunModeRelay:
		return true
	}
	return false
}

func fallbackString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
