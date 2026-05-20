package paths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestDataDirUsesXDGOverride(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")
	t.Setenv("HOME", "/tmp/home")

	got := DataDir()
	want := filepath.Join("/tmp/xdg-data", "overseer")
	if got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestConfigDirFallsBackToHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/home")

	got := ConfigDir()
	want := filepath.Join("/tmp/home", ".config", "overseer")
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestStateDirFallsBackToHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/tmp/home")

	got := StateDir()
	want := filepath.Join("/tmp/home", ".local", "state", "overseer")
	if got != want {
		t.Fatalf("StateDir() = %q, want %q", got, want)
	}
}

func TestFileHelpersUseExpectedNames(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-config")
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")

	if got, want := DataFile(), filepath.Join("/tmp/xdg-data", "overseer", "data.json"); got != want {
		t.Fatalf("DataFile() = %q, want %q", got, want)
	}
	if got, want := ConfigFile(), filepath.Join("/tmp/xdg-config", "overseer", "config.yaml"); got != want {
		t.Fatalf("ConfigFile() = %q, want %q", got, want)
	}
	if got, want := LogFile(), filepath.Join("/tmp/xdg-state", "overseer", "overseer.log"); got != want {
		t.Fatalf("LogFile() = %q, want %q", got, want)
	}
}

func TestEnsureDirCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("EnsureDir() created non-directory %v", info.Mode())
	}
}

func TestSessionWorktreePathUsesUUIDUnderWorktreeRoot(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")

	sessionID := uuid.New()
	got := SessionWorktreePath(sessionID)
	want := filepath.Join("/tmp/xdg-data", "overseer", "worktrees", sessionID.String())
	if got != want {
		t.Fatalf("SessionWorktreePath() = %q, want %q", got, want)
	}
}

func TestSessionFeatureBranchUsesOverseerPrefix(t *testing.T) {
	sessionID := uuid.New()
	got := SessionFeatureBranch(sessionID)
	want := "overseer/" + sessionID.String()
	if got != want {
		t.Fatalf("SessionFeatureBranch() = %q, want %q", got, want)
	}
}

func TestAtomicWriteWritesFileAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	data := []byte("hello, overseer")

	if err := AtomicWrite(path, data); err != nil {
		t.Fatalf("AtomicWrite() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("AtomicWrite() wrote %q, want %q", got, data)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("tmp file still exists or unexpected error: %v", err)
	}
}
