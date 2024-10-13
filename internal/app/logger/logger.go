package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v3"
)

var (
	PacketLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
)

// logLevels maps log level names to slog.Level values.
var logLevels = map[string]slog.Level{
	"trace":   slog.LevelDebug,
	"debug":   slog.LevelDebug,
	"info":    slog.LevelInfo,
	"warn":    slog.LevelWarn,
	"warning": slog.LevelWarn,
	"error":   slog.LevelError,
	"fatal":   slog.LevelError,
}

// CleanupFunc is a function that can be deferred to clean up resources.
type CleanupFunc func() error

// InitDefaultLogger initializes the default logger.
func InitDefaultLogger(app *cli.Command) (CleanupFunc, error) {
	deferred := io.NopCloser(nil).Close

	logLevel, ok := logLevels[strings.ToLower(app.String("log-level"))]
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
		deferred = func() error {
			fmt.Println("Closing log file")
			return w.Close()
		}
	}

	switch strings.ToLower(app.String("log-format")) {
	case "text":
		SetColoredLogger(w, logLevel, app.Bool("no-color"))
	default:
		SetDefaultJSONLogger(w, logLevel)
	}

	return deferred, nil
}

func SetColoredLogger(w *os.File, logLevel slog.Level, forceNoColor bool) {
	slog.SetDefault(slog.New(
		tint.NewHandler(
			colorable.NewColorable(w),
			&tint.Options{
				Level:      logLevel,
				TimeFormat: time.TimeOnly,
				NoColor:    !isatty.IsTerminal(w.Fd()) || os.Getenv("NO_COLOR") != "" || forceNoColor,
				AddSource:  true,
			},
		),
	))
}

func SetPlainTextLogger(w io.Writer, logLevel slog.Level) {
	slog.SetDefault(slog.New(
		tint.NewHandler(
			w,
			&tint.Options{
				Level:      logLevel,
				TimeFormat: time.TimeOnly,
				AddSource:  true,
			},
		),
	))
}

func SetDefaultJSONLogger(w io.Writer, logLevel slog.Level) {
	slog.SetDefault(slog.New(
		slog.NewJSONHandler(w, &slog.HandlerOptions{
			AddSource: logLevel == slog.LevelDebug,
			Level:     logLevel,
		}),
	))
}
