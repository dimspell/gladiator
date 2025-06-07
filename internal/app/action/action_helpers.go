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
	"github.com/dimspell/gladiator/internal/console/database"
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

func selectProxy(c *cli.Command) (p backend.Proxy, err error) {
	switch c.String("proxy") {
	case "lan":
		myIPAddr := c.String("lan-my-ip-addr")
		if ip := net.ParseIP(myIPAddr); ip == nil || len(ip) != 4 {
			return nil, fmt.Errorf("invalid lan-my-ip-addr: %q", myIPAddr)
		}
		return &direct.ProxyLAN{myIPAddr}, nil
	case "webrtc-beta":
		return &p2p.ProxyP2P{}, nil
	case "relay-beta":
		relayAddr := c.String("relay-addr")
		return &relay.ProxyRelay{RelayAddr: relayAddr}, nil
	default:
		return nil, fmt.Errorf("unknown proxy: %q", c.String("proxy"))
	}
}
