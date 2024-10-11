package action

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dimspell/gladiator/backend"
	"github.com/dimspell/gladiator/backend/packetlogger"
	"github.com/dimspell/gladiator/console"
	"github.com/dimspell/gladiator/console/database"
	"github.com/dimspell/gladiator/internal/proxy"
	"github.com/dimspell/gladiator/internal/signalserver"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

func ServeP2PCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "serve-p2p",
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
				Name:  "signaling-addr",
				Value: "127.0.0.1:5050",
				Usage: "Address of the signaling server",
			},
			&cli.StringFlag{
				Name:  "turn-public-ip",
				Value: "127.0.0.1",
				Usage: "",
			},
			&cli.IntFlag{
				Name:  "turn-port",
				Value: 3478,
				Usage: "",
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
		singalingAddr := c.String("signaling-addr")
		turnPublicIP := c.String("turn-public-ip")
		turnPort := c.Int("turn-port")

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

		bd := backend.NewBackend(backendAddr, consoleAddr, proxy.NewPeerToPeer(singalingAddr))
		bd.PacketLogger = slog.New(packetlogger.New(os.Stderr, &packetlogger.Options{
			Level: slog.LevelDebug,
		}))

		con := console.NewConsole(db, consoleAddr)

		startConsole, stopConsole := con.Handlers()

		signalling, err := signalserver.NewServer()
		if err != nil {
			return err
		}
		startTURN, stopTURN := signalling.Run(singalingAddr, turnPublicIP, int(turnPort))

		group, groupContext := errgroup.WithContext(ctx)
		group.Go(func() error {
			return startConsole(groupContext)
		})
		group.Go(func() error {
			return startTURN(ctx)
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
			stopTURN(ctx)
			_ = stopConsole(context.TODO())
		}
		return nil
	}

	return cmd
}
