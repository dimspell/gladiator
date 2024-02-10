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
		Flags:       []cli.Flag{
			// &cli.StringFlag{
			// 	Name:  "proxy-addr",
			// 	Value: defaultConsoleAddr,
			// 	Usage: "Port for the proxy server",
			// },
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		return proxy.NewProxy()
	}

	return cmd
}
