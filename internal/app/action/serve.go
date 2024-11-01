package action

import (
	"context"
	"errors"
	"fmt"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"os"
	"time"
)

const (
	defaultConsoleAddr = "127.0.0.1:2137"
	defaultBackendAddr = "127.0.0.1:6112"
	defaultLobbyAddr   = "ws://127.0.0.1:2137/lobby"
	defaultMyIPAddr    = "127.0.0.1"
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
				Name:  "my-ip-addr",
				Value: defaultMyIPAddr,
				Usage: "IP address used in intercommunication between the users",
			},
			&cli.StringFlag{
				Name:  "database-type",
				Value: "memory",
				Usage: "Database type (memory, sqlite)",
			},
			&cli.StringFlag{
				Name:  "sqlite-path",
				Value: "dispel-multi.sqlite",
				Usage: "Path to sqlite database file",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		consoleAddr := c.String("console-addr")
		backendAddr := c.String("backend-addr")
		myIpAddr := c.String("my-ip-addr")

		var (
			db  *database.SQLite
			err error
		)
		switch c.String("database-type") {
		case "memory":
			db, err = database.NewMemory()
			if err != nil {
				return err
			}
		case "sqlite":
			db, err = database.NewLocal(c.String("sqlite-path"))
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown database type: %q", c.String("database-type"))
		}
		defer func() {
			if err := db.Close(); err != nil {
				slog.Error("Failed to close database", "error", err)
			}
		}()

		if err := database.Seed(db.Write); err != nil {
			slog.Warn("Seed queries failed", "error", err)
		}

		//logger.PacketLogger = slog.New(packetlogger.New(os.Stderr, &packetlogger.Options{
		//	Level: slog.LevelDebug,
		//}))
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

		bd := backend.NewBackend(backendAddr, consoleAddr, backend.NewLAN(myIpAddr))

		// TODO: Name the URL in the parameters
		bd.SignalServerURL = defaultLobbyAddr

		con := console.NewConsole(db, consoleAddr)

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
