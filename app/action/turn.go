package action

import (
	"context"

	"github.com/dimspell/gladiator/console/signalserver"
	"github.com/urfave/cli/v3"
)

func TurnCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "turn",
		Description: "Start signalling and TURN server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "turn-addr",
				Value: "192.168.121.169:38",
				Usage: "IP address of the game server hosting a game",
			},
			&cli.StringFlag{
				Name:  "turn-realm",
				Value: "dispel-multi",
				Usage: "Realm to use for TURN server",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		s, err := signalserver.NewServer()
		if err != nil {
			return err
		}
		s.Run()
		return nil
	}

	return cmd
}
