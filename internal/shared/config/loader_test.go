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

	if cfg.Theme != "dark" {
		t.Errorf("Theme: want dark, got %s", cfg.Theme)
	}
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
	if cfg.Theme != def.Theme {
		t.Errorf("Theme: want %s, got %s", def.Theme, cfg.Theme)
	}
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

func TestLoad_ThemeKey_OverridesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := "theme: tokyo-night\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Theme != "tokyo-night" {
		t.Errorf("Theme: want tokyo-night, got %s", cfg.Theme)
	}
}

func TestLoad_ThemeOmitted_KeepsDefault(t *testing.T) {
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

	if cfg.Theme != "dark" {
		t.Errorf("Theme: omitting field should preserve dark default, got %s", cfg.Theme)
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

	content := `theme: dracula
dashboard:
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

	if cfg.Theme != "dracula" {
		t.Errorf("Theme: want dracula, got %s", cfg.Theme)
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

func TestDefault_ShipsFiveBuiltInLabels(t *testing.T) {
	cfg := config.Default()

	wantCodes := []string{"WIP", "reviewing", "ready", "done", "draft"}
	if len(cfg.Labels) != len(wantCodes) {
		t.Fatalf("Labels: want %d entries, got %d", len(wantCodes), len(cfg.Labels))
	}
	for i, want := range wantCodes {
		if cfg.Labels[i].Code != want {
			t.Errorf("Labels[%d].Code = %q, want %q", i, cfg.Labels[i].Code, want)
		}
		if cfg.Labels[i].Color == "" {
			t.Errorf("Labels[%d].Color is empty", i)
		}
		if cfg.Labels[i].Glyph == "" {
			t.Errorf("Labels[%d].Glyph is empty, want non-empty default glyph", i)
		}
	}
}

func TestLoad_LabelsKey_ReplacesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `labels:
  - code: in-progress
    color: "#ff00ff"
    glyph: "🔥"
  - code: blocked
    color: red
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Labels) != 2 {
		t.Fatalf("Labels: want 2 entries (full replacement), got %d", len(cfg.Labels))
	}
	if cfg.Labels[0].Code != "in-progress" || cfg.Labels[0].Color != "#ff00ff" || cfg.Labels[0].Glyph != "🔥" {
		t.Errorf("Labels[0] = %+v, want {in-progress, #ff00ff, 🔥}", cfg.Labels[0])
	}
	if cfg.Labels[1].Code != "blocked" || cfg.Labels[1].Color != "red" || cfg.Labels[1].Glyph != "" {
		t.Errorf("Labels[1] = %+v, want {blocked, red, \"\"} (glyph omitted)", cfg.Labels[1])
	}
}

func TestLoad_LabelsOmitted_KeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := "theme: dark\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Labels) != 5 {
		t.Fatalf("Labels: omitting field should preserve 5 defaults, got %d", len(cfg.Labels))
	}
	if cfg.Labels[0].Code != "WIP" {
		t.Errorf("Labels[0].Code = %q, want %q", cfg.Labels[0].Code, "WIP")
	}
}

func TestLoad_LabelsEmptyList_ClearsAllLabels(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := "labels: []\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Labels) != 0 {
		t.Fatalf("Labels: explicit empty list should clear defaults, got %d entries", len(cfg.Labels))
	}
}

func TestValidate_LabelInvalidCode_ReturnsInvalidInput(t *testing.T) {
	cfg := config.Default()
	cfg.Labels = []config.LabelConfig{
		{Code: "", Color: "#fff"},
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("Validate() error = nil, want error for empty label code")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("Validate() error = %v, want ErrInvalidInput", err)
	}
}

func TestValidate_LabelInvalidColor_ReturnsInvalidInput(t *testing.T) {
	cfg := config.Default()
	cfg.Labels = []config.LabelConfig{
		{Code: "WIP", Color: ""},
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("Validate() error = nil, want error for empty label color")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("Validate() error = %v, want ErrInvalidInput", err)
	}
}

func TestValidate_LabelDuplicateCodes_ReturnsInvalidInput(t *testing.T) {
	cfg := config.Default()
	cfg.Labels = []config.LabelConfig{
		{Code: "WIP", Color: "#f00"},
		{Code: "WIP", Color: "#0f0"},
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("Validate() error = nil, want error for duplicate label codes")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("Validate() error = %v, want ErrInvalidInput", err)
	}
}

func TestDomainLabels_BuildsValidatedDomainObjects(t *testing.T) {
	cfg := config.Config{
		Labels: []config.LabelConfig{
			{Code: "WIP", Color: "#F59E0B", Glyph: "🚧"},
			{Code: "ready", Color: "#60A5FA"},
		},
	}

	got, err := cfg.DomainLabels()
	if err != nil {
		t.Fatalf("DomainLabels() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("DomainLabels() len = %d, want 2", len(got))
	}
	if got[0].Code != "WIP" || got[0].Color != "#F59E0B" || got[0].Glyph != "🚧" {
		t.Errorf("DomainLabels()[0] = %+v, want {WIP, #F59E0B, 🚧}", got[0])
	}
	if got[1].Code != "ready" || got[1].Color != "#60A5FA" || got[1].Glyph != "" {
		t.Errorf("DomainLabels()[1] = %+v, want {ready, #60A5FA, \"\"}", got[1])
	}
}

func TestDomainLabels_InvalidEntry_ReturnsInvalidInput(t *testing.T) {
	cfg := config.Config{
		Labels: []config.LabelConfig{
			{Code: "OK", Color: "#000"},
			{Code: "", Color: "#fff"},
		},
	}

	_, err := cfg.DomainLabels()
	if err == nil {
		t.Fatal("DomainLabels() error = nil, want error for empty code")
	}
	if !errs.Is(err, errs.ErrInvalidInput) {
		t.Errorf("DomainLabels() error = %v, want ErrInvalidInput", err)
	}
}
