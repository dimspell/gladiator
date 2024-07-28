package action

import (
	"context"

	"github.com/dimspell/gladiator/proxy/redirect"
	"github.com/urfave/cli/v3"
)

func RedirectCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "redirect",
		Description: "Connect as a player and read all packets send by game server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "game-addr",
				Value: "192.168.121.169",
				Usage: "IP address of the game server hosting a game",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		return redirect.NewClientProxy(c.String("game-addr")).Start(ctx)
	}

	return cmd
}
