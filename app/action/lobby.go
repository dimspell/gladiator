package action

import (
	"context"
	"errors"

	"github.com/dimspell/gladiator/internal/lobby"
	"github.com/urfave/cli/v3"
)

func LobbyCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "lobby",
		Description: "Start the signalling server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "lobby-addr",
				Value: defaultLobbyAddr,
				Usage: "Address of the lobby & signaling server",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		lb := lobby.NewLobby(ctx)
		start, stop := lb.Prepare(c.String("lobby-addr"))

		var err error
		err = start(ctx)
		return errors.Join(err, stop(ctx))
	}

	return cmd
}
