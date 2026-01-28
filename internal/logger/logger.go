package logger

import (
	"log/slog"
	"os"
)

// Setup initializes the global slog logger with JSON output and source location.
func Setup(level slog.Level) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	})
	slog.SetDefault(slog.New(handler))
}
