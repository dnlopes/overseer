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

type Config struct {
	Dashboard DashboardConfig  `yaml:"dashboard"`
	Logging   LoggingConfig    `yaml:"logging"`
	Storage   StorageConfig    `yaml:"storage"`
	Launchers []LauncherConfig `yaml:"launchers"`
}

func Default() Config {
	return Config{
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
