package action

import (
	"context"
	"log/slog"
	"os"

	"github.com/dimspell/gladiator/backend"
	"github.com/dimspell/gladiator/backend/packetlogger"
	"github.com/dimspell/gladiator/internal/proxy"
	"github.com/urfave/cli/v3"
)

func BackendP2PCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "backend-p2p",
		Description: "Start backend server and join it to existing console server",
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
				Name:  "turn-public-addr",
				Value: "127.0.0.1:3478",
				Usage: "",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		consoleAddr := c.String("console-addr")
		backendAddr := c.String("backend-addr")
		singalingAddr := c.String("signaling-addr")
		// turnPublicAddr := c.String("turn-public-addr")

		bd := backend.NewBackend(backendAddr, consoleAddr, "")
		bd.Proxy = proxy.NewPeerToPeer(singalingAddr)
		bd.PacketLogger = slog.New(packetlogger.New(os.Stderr, &packetlogger.Options{
			Level: slog.LevelDebug,
		}))

		if err := bd.Start(); err != nil {
			return err
		}
		defer bd.Shutdown()
		bd.Listen()
		return nil
	}
	return cmd
}
