package action

import (
	"context"

	"github.com/dimspell/gladiator/internal/proxy/signalserver"
	"github.com/urfave/cli/v3"
)

func TurnCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "turn",
		Description: "Start signalling and TURN server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "websocket-addr",
				Value: "ws://localhost:5050",
			},
			&cli.StringFlag{
				Name:  "turn-public-ip",
				Value: "127.0.0.1",
			},
			&cli.IntFlag{
				Name:  "turn-port",
				Value: 3478,
			},
			// &cli.StringFlag{
			// 	Name:  "turn-realm",
			// 	Value: "dispel-multi",
			// 	Usage: "Realm to use for TURN server",
			// },
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		s, err := signalserver.NewServer()
		if err != nil {
			return err
		}
		start, stop := s.Run(c.String("websocket-addr"), c.String("turn-public-ip"), int(c.Int("turn-port")))
		defer stop(ctx)
		return start(ctx)
	}

	return cmd
}
