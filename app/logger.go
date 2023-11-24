package app

import (
	"log/slog"
	"os"
)

func initLogger() {
	// f, err := os.Create("logfile.txt")
	// if err != nil {
	// 	panic(err)
	// }

	logger := slog.New(
		slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelDebug,
		}),
	)
	slog.SetDefault(logger)
}
