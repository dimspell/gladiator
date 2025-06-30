package action

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend"
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
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		consoleAddr := c.String("console-addr")
		backendAddr := c.String("backend-addr")

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

		if err := bd.Start(); err != nil {
			return err
		}
		defer bd.Shutdown()
		bd.Listen()
		return nil
	}
	return cmd
}
