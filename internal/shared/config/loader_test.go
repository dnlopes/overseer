package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dnlopes/overseer/internal/shared/config"
	"github.com/dnlopes/overseer/internal/shared/errs"
)

func TestDefault_ReturnsCorrectValues(t *testing.T) {
	cfg := config.Default()

	if cfg.Dashboard.MinWidth != 60 {
		t.Errorf("MinWidth: want 60, got %d", cfg.Dashboard.MinWidth)
	}
	if cfg.Dashboard.MinHeight != 15 {
		t.Errorf("MinHeight: want 15, got %d", cfg.Dashboard.MinHeight)
	}
	if cfg.Dashboard.FocusOnStart != "sessions" {
		t.Errorf("FocusOnStart: want sessions, got %s", cfg.Dashboard.FocusOnStart)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level: want info, got %s", cfg.Logging.Level)
	}
	if cfg.Storage.Path != "" {
		t.Errorf("Storage.Path: want empty, got %s", cfg.Storage.Path)
	}
}

func TestLoad_MissingFile_ReturnsDefaults(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}

	def := config.Default()
	if cfg != def {
		t.Errorf("expected default config, got: %+v", cfg)
	}
}

func TestLoad_InvalidYAML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("dashboard: [\ninvalid yaml {"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput in error chain, got: %v", err)
	}
	if !strings.Contains(err.Error(), "line") {
		t.Errorf("expected error to contain 'line', got: %v", err)
	}
}

func TestLoad_PartialYAML_FillsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := "logging:\n  level: debug\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level: want debug, got %s", cfg.Logging.Level)
	}
	if cfg.Dashboard.MinWidth != 60 {
		t.Errorf("MinWidth: want 60, got %d", cfg.Dashboard.MinWidth)
	}
	if cfg.Dashboard.FocusOnStart != "sessions" {
		t.Errorf("FocusOnStart: want sessions, got %s", cfg.Dashboard.FocusOnStart)
	}
}

func TestLoad_InvalidFocusOnStart_ReturnsValidationError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := "dashboard:\n  focusOnStart: invalid_value\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput in error chain, got: %v", err)
	}
}

func TestLoad_ValidFullConfig_AllFieldsSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `dashboard:
  minWidth: 100
  minHeight: 30
  focusOnStart: status
logging:
  level: warn
storage:
  path: /tmp/overseer
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Dashboard.MinWidth != 100 {
		t.Errorf("MinWidth: want 100, got %d", cfg.Dashboard.MinWidth)
	}
	if cfg.Dashboard.MinHeight != 30 {
		t.Errorf("MinHeight: want 30, got %d", cfg.Dashboard.MinHeight)
	}
	if cfg.Dashboard.FocusOnStart != "status" {
		t.Errorf("FocusOnStart: want status, got %s", cfg.Dashboard.FocusOnStart)
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("Logging.Level: want warn, got %s", cfg.Logging.Level)
	}
	if cfg.Storage.Path != "/tmp/overseer" {
		t.Errorf("Storage.Path: want /tmp/overseer, got %s", cfg.Storage.Path)
	}
}

func TestValidate_InvalidMinWidth_ReturnsError(t *testing.T) {
	cfg := config.Default()
	cfg.Dashboard.MinWidth = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for MinWidth=0, got nil")
	}
}

func TestValidate_AllValidFocusValues(t *testing.T) {
	for _, focus := range []string{"sessions", "status", "preview"} {
		cfg := config.Default()
		cfg.Dashboard.FocusOnStart = focus

		if err := cfg.Validate(); err != nil {
			t.Errorf("FocusOnStart=%q: unexpected error: %v", focus, err)
		}
	}
}
