package app

import (
	"log/slog"
	"os"
)

func initLogger() {
	logger := slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		}),
	)
	slog.SetDefault(logger)
}
