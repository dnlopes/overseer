// Package config loads and validates the overseer YAML config file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	yamlv3 "gopkg.in/yaml.v3"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/shared/errs"
)

type DashboardConfig struct {
	MinWidth  int `yaml:"minWidth"`
	MinHeight int `yaml:"minHeight"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type StorageConfig struct {
	DataDir string `yaml:"dataDir"`
}

type LauncherConfig struct {
	DisplayName string `yaml:"displayName"`
	Command     string `yaml:"command"`
}

type EditorConfig struct {
	DisplayName string `yaml:"displayName"`
	Command     string `yaml:"command"`
}

type LabelConfig struct {
	Code  string `yaml:"code"`
	Color string `yaml:"color"`
	Glyph string `yaml:"glyph"`
}

type Config struct {
	Theme     string           `yaml:"theme"`
	Dashboard DashboardConfig  `yaml:"dashboard"`
	Logging   LoggingConfig    `yaml:"logging"`
	Storage   StorageConfig    `yaml:"storage"`
	Launchers []LauncherConfig `yaml:"launchers"`
	Editors   []EditorConfig   `yaml:"editors"`
	Labels    []LabelConfig    `yaml:"labels"`
}

func Default() Config {
	return Config{
		Theme: "dark",
		Dashboard: DashboardConfig{
			MinWidth:  60,
			MinHeight: 15,
		},
		Logging: LoggingConfig{Level: "info"},
		Storage: StorageConfig{DataDir: ""},
		Launchers: []LauncherConfig{
			{DisplayName: "OpenCode", Command: "opencode"},
			{DisplayName: "Claude Code", Command: "claude"},
		},
		Editors: []EditorConfig{
			{DisplayName: "VSCode", Command: "code"},
		},
		Labels: defaultLabelConfigs(),
	}
}

func defaultLabelConfigs() []LabelConfig {
	out := make([]LabelConfig, len(domain.DefaultLabels))
	for i, l := range domain.DefaultLabels {
		out[i] = LabelConfig{Code: l.Code, Color: l.Color, Glyph: l.Glyph}
	}
	return out
}

func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("config: read %s: %w", path, err)
	}

	if err := yamlv3.Unmarshal(data, &cfg); err != nil {
		return Default(), errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: parse %s: %v", path, err))
	}
	if hasTopLevelKey(data, "launchers") {
		cfg.Launchers = append(Default().Launchers, cfg.Launchers...)
	}
	if hasTopLevelKey(data, "editors") {
		cfg.Editors = append(Default().Editors, cfg.Editors...)
	}

	if err := cfg.Validate(); err != nil {
		return Default(), err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.Dashboard.MinWidth <= 0 {
		return errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: minWidth must be > 0, got %d", c.Dashboard.MinWidth))
	}
	if c.Dashboard.MinHeight <= 0 {
		return errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: minHeight must be > 0, got %d", c.Dashboard.MinHeight))
	}

	if c.Storage.DataDir != "" && !filepath.IsAbs(c.Storage.DataDir) {
		return errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: storage.dataDir %q must be absolute", c.Storage.DataDir))
	}

	for i, l := range c.Launchers {
		if _, err := domain.NewLauncher(l.DisplayName, l.Command); err != nil {
			return errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: launchers[%d]: %v", i, err))
		}
	}

	for i, e := range c.Editors {
		if _, err := domain.NewEditor(e.DisplayName, e.Command); err != nil {
			return errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: editors[%d]: %v", i, err))
		}
	}

	seenLabelCodes := make(map[string]struct{}, len(c.Labels))
	for i, l := range c.Labels {
		label, err := domain.NewLabel(l.Code, l.Color, l.Glyph)
		if err != nil {
			return errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: labels[%d]: %v", i, err))
		}
		if _, dup := seenLabelCodes[label.Code]; dup {
			return errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: labels[%d]: duplicate code %q", i, label.Code))
		}
		seenLabelCodes[label.Code] = struct{}{}
	}

	return nil
}

func hasTopLevelKey(data []byte, key string) bool {
	var root yamlv3.Node
	if err := yamlv3.Unmarshal(data, &root); err != nil {
		return false
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yamlv3.MappingNode {
		return false
	}

	for i := 0; i+1 < len(root.Content[0].Content); i += 2 {
		if root.Content[0].Content[i].Value == key {
			return true
		}
	}
	return false
}

// DomainLaunchers wraps each entry in errs.ErrInvalidInput on failure so
// callers can use errors.Is (same contract as Validate).
func (c Config) DomainLaunchers() ([]domain.Launcher, error) {
	out := make([]domain.Launcher, 0, len(c.Launchers))
	for i, l := range c.Launchers {
		launcher, err := domain.NewLauncher(l.DisplayName, l.Command)
		if err != nil {
			return nil, errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: launchers[%d]: %v", i, err))
		}
		out = append(out, launcher)
	}
	return out, nil
}

// DomainEditors wraps each entry in errs.ErrInvalidInput on failure so
// callers can use errors.Is (same contract as Validate).
func (c Config) DomainEditors() ([]domain.Editor, error) {
	out := make([]domain.Editor, 0, len(c.Editors))
	for i, e := range c.Editors {
		editor, err := domain.NewEditor(e.DisplayName, e.Command)
		if err != nil {
			return nil, errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: editors[%d]: %v", i, err))
		}
		out = append(out, editor)
	}
	return out, nil
}

// DomainLabels wraps each entry in errs.ErrInvalidInput on failure so
// callers can use errors.Is (same contract as Validate). Duplicate-code
// detection lives in Validate, not here — DomainLabels assumes Validate
// has already run.
func (c Config) DomainLabels() ([]domain.Label, error) {
	out := make([]domain.Label, 0, len(c.Labels))
	for i, l := range c.Labels {
		label, err := domain.NewLabel(l.Code, l.Color, l.Glyph)
		if err != nil {
			return nil, errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: labels[%d]: %v", i, err))
		}
		out = append(out, label)
	}
	return out, nil
}
