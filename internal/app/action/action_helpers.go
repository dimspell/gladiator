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
	proxyTypeLAN    = model.RunModeLAN.String()
	proxyTypeWebRTC = model.RunModeWebRTC.String()
	proxyTypeRelay  = model.RunModeRelay.String()
)

func selectProxy(c *cli.Command) (p backend.Proxy, err error) {
	switch c.String("proxy") {
	case proxyTypeLAN:
		myIPAddr := c.String("lan-my-ip-addr")
		if ip := net.ParseIP(myIPAddr); ip == nil {
			return nil, fmt.Errorf("invalid lan-my-ip-addr: %q", myIPAddr)
		}
		return &direct.ProxyLAN{myIPAddr}, nil
	case proxyTypeWebRTC:
		return &p2p.ProxyP2P{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
				{
					URLs:       []string{"turn:127.0.0.1:3478"},
					Username:   "username2",
					Credential: "password2",
				},
			},
		}, nil
	case proxyTypeRelay:
		relayAddr := c.String("relay-addr")
		return &relay.ProxyRelay{RelayServerAddr: relayAddr}, nil
	default:
		return nil, fmt.Errorf("unknown proxy: %q", c.String("proxy"))
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

	return options, nil
}

func fallbackString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
