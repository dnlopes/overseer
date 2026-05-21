package paths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestNewResolver_NoOverride_UsesXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")
	t.Setenv("HOME", "/tmp/home")

	r := NewResolver("")
	got := r.DataFile()
	want := filepath.Join("/tmp/xdg-data", "overseer", "data.json")
	if got != want {
		t.Fatalf("Resolver.DataFile() = %q, want %q", got, want)
	}
}

func TestNewResolver_NoOverride_FallsBackToHomeWhenXDGUnset(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/tmp/home")

	r := NewResolver("")
	got := r.DataFile()
	want := filepath.Join("/tmp/home", ".local", "share", "overseer", "data.json")
	if got != want {
		t.Fatalf("Resolver.DataFile() = %q, want %q", got, want)
	}
}

func TestNewResolver_DataDirOverride_UsedVerbatim(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")

	r := NewResolver("/custom/data/dir")
	want := filepath.Join("/custom/data/dir", "data.json")
	if got := r.DataFile(); got != want {
		t.Fatalf("Resolver.DataFile() = %q, want override-derived %q", got, want)
	}
}

func TestResolver_DataFile_JoinedFromDataDir(t *testing.T) {
	r := NewResolver("/custom/data")
	got := r.DataFile()
	want := filepath.Join("/custom/data", "data.json")
	if got != want {
		t.Fatalf("Resolver.DataFile() = %q, want %q", got, want)
	}
}

func TestResolver_WorktreeRoot_JoinedFromDataDir(t *testing.T) {
	r := NewResolver("/custom/data")
	got := r.WorktreeRoot()
	want := filepath.Join("/custom/data", "worktrees")
	if got != want {
		t.Fatalf("Resolver.WorktreeRoot() = %q, want %q", got, want)
	}
}

func TestResolver_SessionWorktreePath_UsesUUIDUnderWorktreeRoot(t *testing.T) {
	r := NewResolver("/custom/data")
	id := uuid.New()
	got := r.SessionWorktreePath(id)
	want := filepath.Join("/custom/data", "worktrees", id.String()[:8])
	if got != want {
		t.Fatalf("Resolver.SessionWorktreePath() = %q, want %q", got, want)
	}
}

func TestResolver_LogFile_JoinedFromStateDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")

	r := NewResolver("")
	got := r.LogFile()
	want := filepath.Join("/tmp/xdg-state", "overseer", "overseer.log")
	if got != want {
		t.Fatalf("Resolver.LogFile() = %q, want %q", got, want)
	}
}

func TestResolver_LogFile_NotAffectedByDataDirOverride(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")

	r := NewResolver("/custom/data")
	want := filepath.Join("/tmp/xdg-state", "overseer", "overseer.log")
	if got := r.LogFile(); got != want {
		t.Fatalf("Resolver.LogFile() = %q, want %q (DataDir override must not leak into state paths)", got, want)
	}
}

func TestConfigFile_UsesXDGConfigHome(t *testing.T) {
	t.Setenv("OVERSEER_CONFIG_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-config")

	got := ConfigFile()
	want := filepath.Join("/tmp/xdg-config", "overseer", "config.yaml")
	if got != want {
		t.Fatalf("ConfigFile() = %q, want %q", got, want)
	}
}

func TestConfigFile_FallsBackToHome(t *testing.T) {
	t.Setenv("OVERSEER_CONFIG_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/home")

	got := ConfigFile()
	want := filepath.Join("/tmp/home", ".config", "overseer", "config.yaml")
	if got != want {
		t.Fatalf("ConfigFile() = %q, want %q", got, want)
	}
}

func TestConfigFile_EnvOverride_TakesPrecedenceOverXDG(t *testing.T) {
	t.Setenv("OVERSEER_CONFIG_FILE", "/etc/overseer/custom.yaml")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-config")

	got := ConfigFile()
	if got != "/etc/overseer/custom.yaml" {
		t.Fatalf("ConfigFile() = %q, want env override %q", got, "/etc/overseer/custom.yaml")
	}
}

func TestConfigFile_EmptyEnvTreatedAsUnset(t *testing.T) {
	t.Setenv("OVERSEER_CONFIG_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-config")

	got := ConfigFile()
	want := filepath.Join("/tmp/xdg-config", "overseer", "config.yaml")
	if got != want {
		t.Fatalf("empty env: ConfigFile() = %q, want XDG path %q", got, want)
	}
}

func TestConfigFile_EnvOverride_PreservedVerbatim_NotTreatedAsDir(t *testing.T) {
	t.Setenv("OVERSEER_CONFIG_FILE", "/some/dir/myname.yml")

	got := ConfigFile()
	if got != "/some/dir/myname.yml" {
		t.Fatalf("ConfigFile() = %q, want verbatim env value (no config.yaml suffix appended)", got)
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

func TestSessionFeatureBranchUsesOverseerPrefix(t *testing.T) {
	sessionID := uuid.New()
	got := SessionFeatureBranch(sessionID)
	want := "overseer/" + sessionID.String()[:8]
	if got != want {
		t.Fatalf("SessionFeatureBranch() = %q, want %q", got, want)
	}
}
