package logger_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/dnlopes/overseer/internal/shared/logger"
)

func TestLogger_WritesJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OVERSEER_LOG_LEVEL", "")
	logPath := tmp + "/overseer/overseer.log"

	log, closer, err := logger.New(logPath, "info")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer closer.Close()

	log.Info("hello")
	closer.Close()

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
	t.Setenv("OVERSEER_LOG_LEVEL", "debug")
	logPath := tmp + "/overseer/overseer.log"

	log, closer, err := logger.New(logPath, "info")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer closer.Close()

	log.Debug("debug-line")
	closer.Close()

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
