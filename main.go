package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"github.com/dispel-re/dispel-multi/action"
	"github.com/urfave/cli/v3"
)

// Version stores what is a current version and git revision of the build.
// See more by using `go version -m ./path/to/binary` command.
var (
	version = "(devel)"
	commit  = ""
	date    = time.Now().UTC().Format(time.RFC3339)
)

const appName = "dispel-multi"

func main() {
	initLogger()

	app := &cli.Command{
		Name: appName,
		Version: fmt.Sprintf(
			"%s (revision: %s) built on %s",
			version,
			vcsRevision(commit, "0000000")[:7],
			date,
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

func initLogger() {
	logger := slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		}),
	)
	slog.SetDefault(logger)
}
