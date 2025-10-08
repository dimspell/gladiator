package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/dimspell/gladiator/internal/app/action"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/urfave/cli/v3"
)

const appName = "gladiator"

// Version stores what is a current version and git revision of the build.
// See more by using `go version -m ./path/to/binary` command.
var (
	version = "devel"
	commit  = ""
	date    = time.Now().UTC().Format(time.RFC3339)
)

func vcsRevision(value string, defaultValue string) string {
	if value != "" {
		return value
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return defaultValue
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return defaultValue
}

// NewApp creates a new CLI app with the given version, commit and build date.
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
		&cli.StringFlag{
			Name:    "log-level",
			Value:   "debug",
			Usage:   "Log level (debug, info, warn, error)",
			Sources: cli.NewValueSourceChain(cli.EnvVar("LOG_LEVEL")),
			Validator: func(s string) error {
				switch s {
				case "debug", "info", "warn", "error":
					return nil
				default:
					return fmt.Errorf("unknown log level: %s", s)
				}
			},
		},
		&cli.StringFlag{
			Name:    "log-format",
			Value:   "text",
			Usage:   "Log format (text, json, discard)",
			Sources: cli.NewValueSourceChain(cli.EnvVar("LOG_FORMAT")),
			Validator: func(s string) error {
				switch s {
				case "json", "text", "discard":
					return nil
				default:
					return fmt.Errorf("unknown log format: %s", s)
				}
			},
		},
		&cli.StringFlag{
			Name:    "log-file",
			Usage:   "Log file path",
			Sources: cli.NewValueSourceChain(cli.EnvVar("LOG_FILE")),
		},
		&cli.BoolFlag{
			Name:    "no-color",
			Usage:   "Disable colors in log output",
			Sources: cli.NewValueSourceChain(cli.EnvVar("NO_COLOR")),
		},
	)

	// Setup function
	var closers []logger.CleanupFunc
	app.Before = func(ctx context.Context, c *cli.Command) (context.Context, error) {
		closer, err := logger.InitDefaultLogger(c)
		if err != nil {
			return ctx, err
		}
		closers = append(closers, closer)
		return ctx, nil
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
		action.ConsoleCommand(version),
		action.BackendCommand(),
		action.ServeCommand(version),
		action.TurnCommand(),
	)
	if guiCmd := action.GUICommand(app.Version); guiCmd != nil {
		app.Commands = append(app.Commands, guiCmd)
		app.Action = guiCmd.Action
	}

	// Start the app
	if err := app.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	NewApp(version, commit, date)
}
