package paths

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type Resolver struct {
	dataDir  string
	stateDir string
}

func NewResolver(dataDirOverride string) Resolver {
	return Resolver{
		dataDir:  pickDir(dataDirOverride, defaultDataDir),
		stateDir: defaultStateDir(),
	}
}

func (r Resolver) DataFile() string {
	return filepath.Join(r.dataDir, "data.json")
}

func (r Resolver) LogFile() string {
	return filepath.Join(r.stateDir, "overseer.log")
}

func (r Resolver) WorktreeRoot() string {
	return filepath.Join(r.dataDir, "worktrees")
}

func (r Resolver) SessionWorktreePath(sessionID uuid.UUID) string {
	return filepath.Join(r.WorktreeRoot(), sessionID.String())
}

// ConfigFile is a package-level bootstrap helper used to locate the config
// file BEFORE a Resolver can be constructed (chicken-and-egg: the config
// itself defines what the Resolver should override).
//
// Resolution precedence: OVERSEER_CONFIG_FILE env var (verbatim, full path)
// → XDG_CONFIG_HOME/overseer/config.yaml → $HOME/.config/overseer/config.yaml.
func ConfigFile() string {
	if override := os.Getenv("OVERSEER_CONFIG_FILE"); override != "" {
		return override
	}
	return filepath.Join(defaultConfigDir(), "config.yaml")
}

// SessionFeatureBranch is the convention-based git branch name for a feature
// session's worktree: "overseer/<session-id>". The session UUID guarantees
// the branch name is unique within the repository.
func SessionFeatureBranch(sessionID uuid.UUID) string {
	return "overseer/" + sessionID.String()
}

// SessionTrackingBranch is the convention-based git branch name for a
// checkout session's worktree: "overseer/check/<session-id>". It lives
// in a sibling namespace to SessionFeatureBranch so a reader of `git
// branch` can tell at a glance which sessions are feature work and which
// are check / draft views.
func SessionTrackingBranch(sessionID uuid.UUID) string {
	return "overseer/check/" + sessionID.String()
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

func pickDir(override string, fallback func() string) string {
	if override != "" {
		return override
	}
	return fallback()
}

func defaultDataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "overseer")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "overseer")
}

func defaultConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "overseer")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "overseer")
}

func defaultStateDir() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "overseer")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", "overseer")
}
