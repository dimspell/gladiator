package app

import (
	"log/slog"
	"os"
)

func initLogger() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelDebug,
		}),
	)
	slog.SetDefault(logger)
}
