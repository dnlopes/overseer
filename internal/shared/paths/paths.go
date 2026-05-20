package paths

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
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

// WorktreeRoot is the directory under DataDir that holds per-session git
// worktrees. Each session's worktree lives at WorktreeRoot()/<session-id>.
func WorktreeRoot() string {
	return filepath.Join(DataDir(), "worktrees")
}

// SessionWorktreePath returns the absolute worktree path for a session,
// keyed by its UUID. Using the UUID (not the name) keeps the path stable
// across renames.
func SessionWorktreePath(sessionID uuid.UUID) string {
	return filepath.Join(WorktreeRoot(), sessionID.String())
}

// SessionFeatureBranch returns the convention-based git branch name for a
// session's worktree: "overseer/<session-id>". The session UUID guarantees
// the branch name is unique within the repository.
func SessionFeatureBranch(sessionID uuid.UUID) string {
	return "overseer/" + sessionID.String()
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
