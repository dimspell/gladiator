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
		myIpAddr := c.String("my-ip-addr")

		backend.PacketLogger = slog.New(packetlogger.New(os.Stderr, &packetlogger.Options{
			Level: slog.LevelDebug,
		}))

		bd := backend.NewBackend(backendAddr, consoleAddr, proxy.NewLAN(myIpAddr))

		if err := bd.Start(); err != nil {
			return err
		}
		defer bd.Shutdown()
		bd.Listen()
		return nil
	}
	return cmd
}
