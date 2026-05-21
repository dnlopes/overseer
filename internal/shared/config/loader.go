// Package config loads and validates the overseer YAML config file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	yamlv3 "gopkg.in/yaml.v3"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/shared/errs"
)

type DashboardConfig struct {
	MinWidth               int    `yaml:"minWidth"`
	MinHeight              int    `yaml:"minHeight"`
	PreviewRefreshInterval string `yaml:"previewRefreshInterval"`
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

type Config struct {
	Theme     string           `yaml:"theme"`
	Dashboard DashboardConfig  `yaml:"dashboard"`
	Logging   LoggingConfig    `yaml:"logging"`
	Storage   StorageConfig    `yaml:"storage"`
	Launchers []LauncherConfig `yaml:"launchers"`
	Editors   []EditorConfig   `yaml:"editors"`
}

func Default() Config {
	return Config{
		Theme: "dark",
		Dashboard: DashboardConfig{
			MinWidth:               60,
			MinHeight:              15,
			PreviewRefreshInterval: "500ms",
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
	}
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
	if _, err := c.PreviewRefreshDuration(); err != nil {
		return err
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

// PreviewRefreshDuration parses Dashboard.PreviewRefreshInterval into a
// time.Duration and ensures it is strictly positive. Wraps failures in
// errs.ErrInvalidInput so callers can use errors.Is (same contract as
// Validate).
func (c Config) PreviewRefreshDuration() (time.Duration, error) {
	raw := c.Dashboard.PreviewRefreshInterval
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: dashboard.previewRefreshInterval %q: %v", raw, err))
	}
	if d <= 0 {
		return 0, errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: dashboard.previewRefreshInterval %q must be > 0", raw))
	}
	return d, nil
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
