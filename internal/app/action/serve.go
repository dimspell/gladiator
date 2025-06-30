package action

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

func ServeCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "serve",
		Description: "Start the backend and console server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "console-addr",
				Value: defaultConsoleAddr,
				Usage: "Port for the console server",
			},
			&cli.StringFlag{
				Name:  "backend-addr",
				Value: defaultBackendAddr,
				Usage: "Port for the backend server",
			},
			&cli.StringFlag{
				Name:  "proxy",
				Value: defaultProxyType,
				Usage: fmt.Sprintf("Proxy type to use. Possible values are: %q, %q, %q", "lan", "webrtc-beta", "relay-beta"),
			},
			&cli.StringFlag{
				Name:  "lan-my-ip-addr",
				Value: defaultMyIPAddr,
				Usage: "IP address used in intercommunication between the users (only in lan proxy)",
			},
			&cli.StringFlag{
				Name:  "relay-addr",
				Value: defaultRelayAddr,
				Usage: "Address of the relay server (only in relay proxy)",
			},
			&cli.StringFlag{
				Name:  "lobby-addr",
				Value: defaultLobbyAddr,
			},
			&cli.StringFlag{
				Name:  "database-type",
				Value: defaultDatabaseType,
				Usage: "Database type (memory, sqlite)",
			},
			&cli.StringFlag{
				Name:  "sqlite-path",
				Value: defaultDatabasePath,
				Usage: "Path to sqlite database file",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		consoleAddr := c.String("console-addr")
		backendAddr := c.String("backend-addr")

		db, err := selectDatabaseType(c)
		if err != nil {
			return err
		}
		defer func() {
			if err := db.Close(); err != nil {
				slog.Error("Failed to close database", logging.Error(err))
			}
		}()

		// logger.PacketLogger = slog.New(packetlogger.New(os.Stderr, &packetlogger.Options{
		//	Level: slog.LevelDebug,
		// }))
		logger.PacketLogger = slog.New(
			tint.NewHandler(
				os.Stderr,
				&tint.Options{
					Level:      slog.LevelDebug,
					TimeFormat: time.TimeOnly,
					AddSource:  true,
				},
			),
		)

		px, err := selectProxy(c)
		if err != nil {
			return err
		}

		bd := backend.NewBackend(backendAddr, consoleAddr, px)
		bd.SignalServerURL = c.String("lobby-addr")

		co, err := selectConsoleOptions(c)
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
			return errors.Join(err, stopConsole(context.TODO()))
		}
		return nil
	}

	return cmd
}
