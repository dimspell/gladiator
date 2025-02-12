package action

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/dimspell/gladiator/internal/backend/proxy/p2p"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v3"
)

func BackendCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "backend",
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
				Name:  "my-ip-addr",
				Value: defaultMyIPAddr,
				Usage: "IP address used in intercommunication between the users",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		consoleAddr := c.String("console-addr")
		backendAddr := c.String("backend-addr")
		// myIpAddr := c.String("my-ip-addr")

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

		// bd := backend.NewBackend(backendAddr, consoleAddr, backend.NewLAN(myIpAddr))
		bd := backend.NewBackend(backendAddr, consoleAddr, p2p.NewPeerToPeer())

		// TODO: Name the URL in the parameters
		bd.SignalServerURL = defaultLobbyAddr

		if err := bd.Start(); err != nil {
			return err
		}
		defer bd.Shutdown()
		bd.Listen()
		return nil
	}
	return cmd
}
