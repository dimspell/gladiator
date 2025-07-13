package action

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/urfave/cli/v3"
)

func BackendCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "backend",
		Description: "Start backend server and join it to existing console server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "console-addr",
				Value:   defaultPublicConsoleAddr,
				Usage:   "Address to the console server (with http:// or https://)",
				Sources: cli.NewValueSourceChain(cli.EnvVar("CONSOLE_ADDR")),
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
				Value:   defaultRelayAddr,
				Usage:   "Address of the relay server (only in relay proxy)",
				Sources: cli.NewValueSourceChain(cli.EnvVar("RELAY_ADDR")),
			},
			&cli.StringFlag{
				Name:    "lobby-addr",
				Value:   defaultLobbyAddr,
				Sources: cli.NewValueSourceChain(cli.EnvVar("LOBBY_ADDR")),
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		consoleAddr := c.String("console-addr")
		backendAddr := c.String("backend-addr")
		lobbyAddr := c.String("lobby-addr")

		// logger.PacketLogger = slog.New(packetlogger.New(os.Stderr, &packetlogger.Options{
		//	Level: slog.LevelDebug,
		// }))
		logger.PacketLogger = slog.Default()

		px, err := selectProxy(c)
		if err != nil {
			return err
		}

		metadata, err := backend.GetMetadata(ctx, consoleAddr)
		if err != nil {
			return err
		}

		if px.Mode() != metadata.RunMode {
			return fmt.Errorf("incorrect run-mode - was %q; expected %q", px.Mode(), metadata.RunMode)
		}

		bd := backend.NewBackend(backendAddr, consoleAddr, px)
		bd.SignalServerURL = lobbyAddr

		if err := bd.Start(); err != nil {
			return err
		}
		defer bd.Shutdown()
		bd.Listen()
		return nil
	}
	return cmd
}
