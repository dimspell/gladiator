package app

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	colorable "github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v3"
)

// LogLevels maps log level names to slog.Level values.
var LogLevels = map[string]slog.Level{
	"trace":   slog.LevelDebug,
	"debug":   slog.LevelDebug,
	"info":    slog.LevelInfo,
	"warn":    slog.LevelWarn,
	"warning": slog.LevelWarn,
	"error":   slog.LevelError,
	"fatal":   slog.LevelError,
}

// CleanupFunc is a function that can be deferred to cleanup resources.
type CleanupFunc func()

// initDefaultLogger initializes the default logger.
func initDefaultLogger(app *cli.Command) (CleanupFunc, error) {
	deferred := func() {
		// noop
	}

	logLevel, ok := LogLevels[strings.ToLower(app.String("log-level"))]
	if !ok {
		return deferred, fmt.Errorf("invalid log level: %s", app.String("log-level"))
	}

	var w *os.File = os.Stderr
	if app.String("log-file") != "" {
		var err error
		w, err = os.OpenFile(app.String("log-file"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return deferred, err
		}
		deferred = func() {
			fmt.Println("Closing log file")
			w.Close()
		}
	}

	switch strings.ToLower(app.String("log-format")) {
	case "text":
		slog.SetDefault(slog.New(
			tint.NewHandler(
				colorable.NewColorable(w),
				&tint.Options{
					Level:      logLevel,
					TimeFormat: time.TimeOnly,
					NoColor:    !isatty.IsTerminal(w.Fd()) || os.Getenv("NO_COLOR") != "" || app.Bool("no-color"),
				},
			),
		))
	default:
		slog.SetDefault(slog.New(
			slog.NewJSONHandler(w, &slog.HandlerOptions{
				AddSource: logLevel == slog.LevelDebug,
				Level:     logLevel,
			}),
		))
	}

	return deferred, nil
}
