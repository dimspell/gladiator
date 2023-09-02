package app

import (
	"context"
	"fmt"
	"os"

	"github.com/dispel-re/dispel-multi/app/action"
	"github.com/urfave/cli/v3"
)

const appName = "dispel-multi"

func NewApp(version, commit, buildDate string) {
	initLogger()

	app := &cli.Command{
		Name: appName,
		Version: fmt.Sprintf(
			"%s (revision: %s) built on %s",
			version,
			vcsRevision(commit, "0000000")[:7],
			buildDate,
		),
	}

	app.Commands = append(app.Commands,
		action.ServeCommand(),
		action.ConsoleCommand(),
	)

	if err := app.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
