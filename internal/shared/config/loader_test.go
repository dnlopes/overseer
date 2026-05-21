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
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level: want info, got %s", cfg.Logging.Level)
	}
	if cfg.Storage.DataDir != "" {
		t.Errorf("Storage.DataDir: want empty, got %s", cfg.Storage.DataDir)
	}
}

func TestDefault_ShipsOpencodeAndClaudeLaunchers(t *testing.T) {
	cfg := config.Default()

	if len(cfg.Launchers) != 2 {
		t.Fatalf("Launchers: want 2 entries, got %d", len(cfg.Launchers))
	}
	if cfg.Launchers[0].DisplayName != "OpenCode" || cfg.Launchers[0].Command != "opencode" {
		t.Errorf("Launchers[0]: want {OpenCode, opencode}, got %+v", cfg.Launchers[0])
	}
	if cfg.Launchers[1].DisplayName != "Claude Code" || cfg.Launchers[1].Command != "claude" {
		t.Errorf("Launchers[1]: want {Claude Code, claude}, got %+v", cfg.Launchers[1])
	}
}

func TestDefault_ShipsVSCodeEditor(t *testing.T) {
	cfg := config.Default()

	if len(cfg.Editors) != 1 {
		t.Fatalf("Editors: want 1 entry, got %d", len(cfg.Editors))
	}
	if cfg.Editors[0].DisplayName != "VSCode" || cfg.Editors[0].Command != "code" {
		t.Errorf("Editors[0]: want {VSCode, code}, got %+v", cfg.Editors[0])
	}
}

func TestLoad_MissingFile_ReturnsDefaults(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}

	def := config.Default()
	if cfg.Dashboard != def.Dashboard {
		t.Errorf("Dashboard: want %+v, got %+v", def.Dashboard, cfg.Dashboard)
	}
	if cfg.Logging != def.Logging {
		t.Errorf("Logging: want %+v, got %+v", def.Logging, cfg.Logging)
	}
	if cfg.Storage != def.Storage {
		t.Errorf("Storage: want %+v, got %+v", def.Storage, cfg.Storage)
	}
	if len(cfg.Launchers) != len(def.Launchers) {
		t.Errorf("Launchers length: want %d, got %d", len(def.Launchers), len(cfg.Launchers))
	}
	if len(cfg.Editors) != len(def.Editors) {
		t.Errorf("Editors length: want %d, got %d", len(def.Editors), len(cfg.Editors))
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
	if len(cfg.Launchers) != 2 {
		t.Errorf("Launchers: want 2 defaults retained, got %d", len(cfg.Launchers))
	}
}

func TestLoad_ValidFullConfig_AllFieldsSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `dashboard:
  minWidth: 100
  minHeight: 30
logging:
  level: warn
storage:
  dataDir: /tmp/overseer
launchers:
  - displayName: Custom Agent
    command: my-agent --foo
  - displayName: Plain Bash
    command: bash
editors:
  - displayName: Cursor
    command: cursor
  - displayName: Neovim
    command: nvim
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
	if cfg.Logging.Level != "warn" {
		t.Errorf("Logging.Level: want warn, got %s", cfg.Logging.Level)
	}
	if cfg.Storage.DataDir != "/tmp/overseer" {
		t.Errorf("Storage.DataDir: want /tmp/overseer, got %s", cfg.Storage.DataDir)
	}
	if len(cfg.Launchers) != 4 {
		t.Fatalf("Launchers: want 4 entries, got %d", len(cfg.Launchers))
	}
	if cfg.Launchers[0].DisplayName != "OpenCode" || cfg.Launchers[0].Command != "opencode" {
		t.Errorf("Launchers[0]: want {OpenCode, opencode}, got %+v", cfg.Launchers[0])
	}
	if cfg.Launchers[1].DisplayName != "Claude Code" || cfg.Launchers[1].Command != "claude" {
		t.Errorf("Launchers[1]: want {Claude Code, claude}, got %+v", cfg.Launchers[1])
	}
	if cfg.Launchers[2].DisplayName != "Custom Agent" || cfg.Launchers[2].Command != "my-agent --foo" {
		t.Errorf("Launchers[2]: want {Custom Agent, my-agent --foo}, got %+v", cfg.Launchers[2])
	}
	if cfg.Launchers[3].DisplayName != "Plain Bash" || cfg.Launchers[3].Command != "bash" {
		t.Errorf("Launchers[3]: want {Plain Bash, bash}, got %+v", cfg.Launchers[3])
	}
	if len(cfg.Editors) != 3 {
		t.Fatalf("Editors: want 3 entries, got %d", len(cfg.Editors))
	}
	if cfg.Editors[0].DisplayName != "VSCode" || cfg.Editors[0].Command != "code" {
		t.Errorf("Editors[0]: want {VSCode, code}, got %+v", cfg.Editors[0])
	}
	if cfg.Editors[1].DisplayName != "Cursor" || cfg.Editors[1].Command != "cursor" {
		t.Errorf("Editors[1]: want {Cursor, cursor}, got %+v", cfg.Editors[1])
	}
	if cfg.Editors[2].DisplayName != "Neovim" || cfg.Editors[2].Command != "nvim" {
		t.Errorf("Editors[2]: want {Neovim, nvim}, got %+v", cfg.Editors[2])
	}
}

func TestLoad_RelativeDataDir_RejectedWithInvalidInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := "storage:\n  dataDir: ./relative/path\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for relative DataDir, got nil")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput in error chain, got: %v", err)
	}
}

func TestLoad_ExplicitEmptyLaunchers_KeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := "launchers: []\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("explicit empty launchers should load cleanly, got: %v", err)
	}
	if len(cfg.Launchers) != 2 {
		t.Errorf("Launchers: want 2 defaults, got %d", len(cfg.Launchers))
	}
}

func TestLoad_LauncherFieldOmitted_KeepsDefaults(t *testing.T) {
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
	if len(cfg.Launchers) != 2 {
		t.Errorf("Launchers: omitting field should preserve 2 defaults, got %d", len(cfg.Launchers))
	}
}

func TestLoad_LauncherMissingCommand_RejectedWithInvalidInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `launchers:
  - displayName: NoCommand
    command: ""
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for empty launcher command, got nil")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput in error chain, got: %v", err)
	}
}

func TestLoad_ExplicitEmptyEditors_KeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := "editors: []\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("explicit empty editors should load cleanly, got: %v", err)
	}
	if len(cfg.Editors) != 1 {
		t.Errorf("Editors: want 1 default, got %d", len(cfg.Editors))
	}
}

func TestLoad_EditorFieldOmitted_KeepsDefaults(t *testing.T) {
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
	if len(cfg.Editors) != 1 {
		t.Errorf("Editors: omitting field should preserve 1 default, got %d", len(cfg.Editors))
	}
}

func TestLoad_EditorMissingCommand_RejectedWithInvalidInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `editors:
  - displayName: NoCommand
    command: ""
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for empty editor command, got nil")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput in error chain, got: %v", err)
	}
}

func TestValidate_InvalidMinWidth_ReturnsError(t *testing.T) {
	cfg := config.Default()
	cfg.Dashboard.MinWidth = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for MinWidth=0, got nil")
	}
}

func TestValidate_InvalidMinHeight_ReturnsError(t *testing.T) {
	cfg := config.Default()
	cfg.Dashboard.MinHeight = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for MinHeight=0, got nil")
	}
}

func TestDomainLaunchers_ConvertsValidEntries(t *testing.T) {
	cfg := config.Config{
		Launchers: []config.LauncherConfig{
			{DisplayName: "OpenCode", Command: "opencode"},
			{DisplayName: "Claude", Command: "claude --debug"},
		},
	}

	got, err := cfg.DomainLaunchers()
	if err != nil {
		t.Fatalf("DomainLaunchers() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("DomainLaunchers() length = %d, want 2", len(got))
	}
	if got[0].DisplayName != "OpenCode" || got[0].Command != "opencode" {
		t.Errorf("DomainLaunchers()[0] = %+v, want {OpenCode, opencode}", got[0])
	}
	if got[1].DisplayName != "Claude" || got[1].Command != "claude --debug" {
		t.Errorf("DomainLaunchers()[1] = %+v, want {Claude, claude --debug}", got[1])
	}
}

func TestDomainLaunchers_InvalidEntry_ReturnsInvalidInput(t *testing.T) {
	cfg := config.Config{
		Launchers: []config.LauncherConfig{
			{DisplayName: "OK", Command: "ok"},
			{DisplayName: "", Command: "x"},
		},
	}

	_, err := cfg.DomainLaunchers()
	if err == nil {
		t.Fatal("expected error for empty display name, got nil")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput in error chain, got: %v", err)
	}
}

func TestDomainEditors_ConvertsValidEntries(t *testing.T) {
	cfg := config.Config{
		Editors: []config.EditorConfig{
			{DisplayName: "VSCode", Command: "code"},
			{DisplayName: "Cursor", Command: "cursor --wait"},
		},
	}

	got, err := cfg.DomainEditors()
	if err != nil {
		t.Fatalf("DomainEditors() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("DomainEditors() length = %d, want 2", len(got))
	}
	if got[0].DisplayName != "VSCode" || got[0].Command != "code" {
		t.Errorf("DomainEditors()[0] = %+v, want {VSCode, code}", got[0])
	}
	if got[1].DisplayName != "Cursor" || got[1].Command != "cursor --wait" {
		t.Errorf("DomainEditors()[1] = %+v, want {Cursor, cursor --wait}", got[1])
	}
}

func TestDomainEditors_InvalidEntry_ReturnsInvalidInput(t *testing.T) {
	cfg := config.Config{
		Editors: []config.EditorConfig{
			{DisplayName: "OK", Command: "ok"},
			{DisplayName: "", Command: "x"},
		},
	}

	_, err := cfg.DomainEditors()
	if err == nil {
		t.Fatal("expected error for empty display name, got nil")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput in error chain, got: %v", err)
	}
}
