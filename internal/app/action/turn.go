package action

import (
	"context"
	"strconv"

	"github.com/dimspell/gladiator/internal/turn"
	"github.com/urfave/cli/v3"
)

func TurnCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "turn",
		Description: "Start TURN server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "turn-public-ip",
				Value: "127.0.0.1",
			},
			&cli.IntFlag{
				Name:  "turn-port",
				Value: 3478,
			},
			&cli.StringFlag{
				Name:  "turn-realm",
				Value: "dispel-multi",
				Usage: "Realm to use for TURN server",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		turn.StartWithConfig(&turn.Config{
			PublicIPAddr: c.String("turn-public-ip"),
			PortNumber:   strconv.Itoa(int(c.Int("turn-port"))),
			Realm:        c.String("turn-realm"),
			Users:        `username1=password1,username2=password2`,
		})
		return nil
	}

	return cmd
}
