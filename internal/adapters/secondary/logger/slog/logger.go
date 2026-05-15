package slog

import (
	"fmt"
	"io"
	stdslog "log/slog"
	"os"
	"path/filepath"

	"github.com/dnlopes/overseer/internal/shared/paths"
)

// New creates a slog.Logger that writes JSON to the XDG log file.
// The caller must defer the returned io.Closer.
// OVERSEER_LOG_LEVEL env var overrides the level parameter.
func New(level string) (*stdslog.Logger, io.Closer, error) {
	if envLevel := os.Getenv("OVERSEER_LOG_LEVEL"); envLevel != "" {
		level = envLevel
	}

	lvl := stdslog.LevelInfo
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = stdslog.LevelInfo
	}

	logPath := paths.LogFile()
	if err := paths.EnsureDir(filepath.Dir(logPath)); err != nil {
		return nil, nil, fmt.Errorf("logger: ensure dir: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("logger: open log file: %w", err)
	}

	handler := stdslog.NewJSONHandler(f, &stdslog.HandlerOptions{Level: lvl})
	return stdslog.New(handler), f, nil
}
