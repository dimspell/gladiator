package action

import (
	"github.com/dispel-re/dispel-multi/console/client"
	"github.com/urfave/cli/v3"
)

func ConsoleCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "console",
		Description: "Choose command and execute it in interactive interface",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "console-addr",
				Value: defaultConsoleAddr,
				Usage: "Port for the console server",
			},
		},
	}

	cmd.Action = func(c *cli.Context) error {
		consoleAddr := c.String("console-addr")

		_ = client.New(consoleAddr)

		// Run fzf-like interface to select the action and name the parameters

		return nil
	}

	return cmd
}
