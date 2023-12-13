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
	app := &cli.Command{
		Name: appName,
		Version: fmt.Sprintf(
			"%s (revision: %s) built on %s",
			version,
			vcsRevision(commit, "0000000")[:7],
			buildDate,
		),
	}

	// Root flags
	app.Flags = append(app.Flags,
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug mode",
		},
		&cli.StringFlag{
			Name:  "log-level",
			Value: "debug",
			Usage: "Log level (debug, info, warn, error)",
		},
		&cli.StringFlag{
			Name:  "log-format",
			Value: "text",
			Usage: "Log format (text, json)",
		},
		&cli.StringFlag{
			Name:  "log-file",
			Usage: "Log file path",
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable colors in log output",
		},
	)

	// Setup function
	var closers []CleanupFunc
	app.Before = func(_ context.Context, c *cli.Command) error {
		closer, err := initDefaultLogger(c)
		if err != nil {
			return err
		}
		closers = append(closers, closer)
		return nil
	}

	// Cleanup function
	app.After = func(_ context.Context, _ *cli.Command) error {
		for _, closer := range closers {
			closer()
		}
		return nil
	}

	// Assign commands
	app.Commands = append(app.Commands,
		action.ConsoleCommand(),
		action.BackendCommand(),
		action.ServeCommand(),
	)

	// Start the app
	if err := app.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
