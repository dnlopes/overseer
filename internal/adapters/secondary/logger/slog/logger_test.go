package slog_test

import (
	"encoding/json"
	"os"
	"testing"

	slogadapter "github.com/dnlopes/overseer/internal/adapters/secondary/logger/slog"
)

func TestLogger_WritesJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)
	t.Setenv("OVERSEER_LOG_LEVEL", "")

	logger, closer, err := slogadapter.New("info")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer closer.Close()

	logger.Info("hello")
	closer.Close()

	logPath := tmp + "/overseer/overseer.log"
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("unmarshal JSON log line: %v\ncontent: %s", err, data)
	}

	if entry["msg"] != "hello" {
		t.Errorf("expected msg=hello, got %v", entry["msg"])
	}
	if entry["level"] != "INFO" {
		t.Errorf("expected level=INFO, got %v", entry["level"])
	}
}

func TestLogger_LevelEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)
	t.Setenv("OVERSEER_LOG_LEVEL", "debug")

	logger, closer, err := slogadapter.New("info")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer closer.Close()

	logger.Debug("debug-line")
	closer.Close()

	logPath := tmp + "/overseer/overseer.log"
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("unmarshal JSON log line: %v\ncontent: %s", err, data)
	}

	if entry["msg"] != "debug-line" {
		t.Errorf("expected msg=debug-line, got %v", entry["msg"])
	}
	if entry["level"] != "DEBUG" {
		t.Errorf("expected level=DEBUG, got %v", entry["level"])
	}
}
