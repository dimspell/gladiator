package action

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

func ServeCommand(version string) *cli.Command {
	cmd := &cli.Command{
		Name:        "serve",
		Description: "Start the backend and console server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "console-addr",
				Aliases: []string{"console-bind"},
				Value:   defaultConsoleAddr,
				Usage:   "Bind address for the console server",
				Sources: cli.NewValueSourceChain(cli.EnvVar("CONSOLE_ADDR"), cli.EnvVar("CONSOLE_BIND")),
			},
			&cli.StringFlag{
				Name:    "console-public-addr",
				Value:   defaultPublicConsoleAddr,
				Usage:   "Public address to the console server (with http:// or https://)",
				Sources: cli.NewValueSourceChain(cli.EnvVar("CONSOLE_PUBLIC_ADDR")),
			},
			&cli.StringFlag{
				Name:    "backend-addr",
				Value:   defaultBackendAddr,
				Usage:   "Bind address for the backend server",
				Sources: cli.NewValueSourceChain(cli.EnvVar("BACKEND_ADDR")),
			},
			&cli.StringFlag{
				Name:    "proxy",
				Value:   defaultProxyType,
				Usage:   fmt.Sprintf("Proxy type to use. Possible values are: %q, %q, %q", proxyTypeLAN, proxyTypeWebRTC, proxyTypeRelay),
				Sources: cli.NewValueSourceChain(cli.EnvVar("PROXY")),
			},
			&cli.StringFlag{
				Name:    "lan-my-ip-addr",
				Value:   defaultMyIPAddr,
				Usage:   "IP address used in intercommunication between the users (only in lan proxy)",
				Sources: cli.NewValueSourceChain(cli.EnvVar("LAN_MY_IP_ADDR")),
			},
			&cli.StringFlag{
				Name:    "relay-addr",
				Aliases: []string{"relay-bind"},
				Value:   defaultRelayAddr,
				Usage:   "Bind address for the relay server",
				Sources: cli.NewValueSourceChain(cli.EnvVar("RELAY_ADDR"), cli.EnvVar("RELAY_BIND")),
			},
			&cli.StringFlag{
				Name:    "relay-public-addr",
				Usage:   "Public address to the relay server",
				Sources: cli.NewValueSourceChain(cli.EnvVar("RELAY_PUBLIC_ADDR")),
			},
			&cli.StringFlag{
				Name:    "lobby-addr",
				Value:   defaultLobbyAddr,
				Sources: cli.NewValueSourceChain(cli.EnvVar("LOBBY_ADDR")),
			},
			&cli.StringFlag{
				Name:    "database-type",
				Value:   defaultDatabaseType,
				Usage:   "Database type (memory, sqlite)",
				Sources: cli.NewValueSourceChain(cli.EnvVar("DATABASE_TYPE")),
			},
			&cli.StringFlag{
				Name:    "sqlite-path",
				Value:   defaultDatabasePath,
				Usage:   "Path to sqlite database file",
				Sources: cli.NewValueSourceChain(cli.EnvVar("SQLITE_PATH")),
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		consolePublicAddr := c.String("console-public-addr")
		backendAddr := c.String("backend-addr")
		lobbyAddr := c.String("lobby-addr")

		db, err := selectDatabaseType(c)
		if err != nil {
			return err
		}
		defer func() {
			if err := db.Close(); err != nil {
				slog.Error("Failed to close database", logging.Error(err))
			}
		}()

		px, err := selectProxy(c)
		if err != nil {
			return err
		}

		bd := backend.NewBackend(backendAddr, consolePublicAddr, px)
		bd.SignalServerURL = lobbyAddr

		co, err := selectConsoleOptions(c, version)
		if err != nil {
			return err
		}
		con := console.NewConsole(db, co...)

		startConsole, stopConsole := con.Handlers()

		group, groupContext := errgroup.WithContext(ctx)
		group.Go(func() error {
			return startConsole(groupContext)
		})
		group.Go(func() error {
			if err := bd.Start(); err != nil {
				return err
			}
			bd.Listen()
			return nil
		})

		if err := group.Wait(); err != nil {
			bd.Shutdown()
			return errors.Join(err, stopConsole(ctx))
		}
		return nil
	}

	return cmd
}
