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
				Name:  "game-addr",
				Value: "192.168.121.169",
				Usage: "IP address of the game server hosting a game",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		bindIP := "127.0.0.1"

		p := proxy.GlobalProxy{
			MaxActiveClients: 32,
			Games:            make(map[string]*proxy.Game),
			Connections:      make(map[string]*proxy.Client),
		}
		return p.Run(bindIP)
		// return proxy.NewClientProxy(c.String("game-addr")).Start(ctx)
	}

	return cmd
}
