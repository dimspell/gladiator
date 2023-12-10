package action

import (
	"github.com/dispel-re/dispel-multi/backend"
	"github.com/urfave/cli/v3"
)

func BackendCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "backend",
		Description: "Start backend server and join it to existing server",
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
		},
	}

	cmd.Action = func(c *cli.Context) error {
		consoleAddr := c.String("console-addr")
		backendAddr := c.String("backend-addr")

		bd := backend.NewBackend(consoleAddr)
		bd.Listen(backendAddr)
		return nil
	}

	return cmd
}
