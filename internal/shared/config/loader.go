// Package config loads and validates the overseer YAML config file.
package config

import (
	"errors"
	"fmt"
	"os"

	yamlv3 "gopkg.in/yaml.v3"

	"github.com/dnlopes/overseer/internal/shared/errs"
)

type DashboardConfig struct {
	MinWidth     int    `yaml:"minWidth"`
	MinHeight    int    `yaml:"minHeight"`
	FocusOnStart string `yaml:"focusOnStart"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type StorageConfig struct {
	Path string `yaml:"path"`
}

type Config struct {
	Dashboard DashboardConfig `yaml:"dashboard"`
	Logging   LoggingConfig   `yaml:"logging"`
	Storage   StorageConfig   `yaml:"storage"`
}

func Default() Config {
	return Config{
		Dashboard: DashboardConfig{
			MinWidth:     60,
			MinHeight:    15,
			FocusOnStart: "sessions",
		},
		Logging: LoggingConfig{Level: "info"},
		Storage: StorageConfig{Path: ""},
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

	if err := cfg.Validate(); err != nil {
		return Default(), err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	validFocus := map[string]bool{
		"sessions": true,
		"status":   true,
		"preview":  true,
	}

	if !validFocus[c.Dashboard.FocusOnStart] {
		return errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: focusOnStart %q must be one of: sessions, status, preview", c.Dashboard.FocusOnStart))
	}
	if c.Dashboard.MinWidth <= 0 {
		return errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: minWidth must be > 0, got %d", c.Dashboard.MinWidth))
	}
	if c.Dashboard.MinHeight <= 0 {
		return errs.Wrap(errs.ErrInvalidInput, fmt.Sprintf("config: minHeight must be > 0, got %d", c.Dashboard.MinHeight))
	}

	return nil
}
