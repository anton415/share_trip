package app

import (
	"log/slog"
	"os"
)

func NewLogger() (*slog.Logger, *os.File, error) {
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(
		"logs/app.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, nil, err
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, file, nil
}
