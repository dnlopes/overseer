package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "overseer")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "overseer")
}

func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "overseer")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "overseer")
}

func StateDir() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "overseer")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", "overseer")
}

func DataFile() string {
	return filepath.Join(DataDir(), "data.json")
}

func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func LogFile() string {
	return filepath.Join(StateDir(), "overseer.log")
}

func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func AtomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("atomic write: %w", err)
	}
	return os.Rename(tmp, path)
}
