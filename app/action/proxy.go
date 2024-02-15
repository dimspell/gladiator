package action

import (
	"context"

	"github.com/dispel-re/dispel-multi/proxy"
	"github.com/urfave/cli/v3"
)

func ProxyCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "proxy",
		Description: "Start proxy server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "addr",
				Value: "192.168.121.212",
				Usage: "IP address of the game server hosting a game",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		return proxy.NewClientProxy(c.String("addr")).Start(ctx)
	}

	return cmd
}
