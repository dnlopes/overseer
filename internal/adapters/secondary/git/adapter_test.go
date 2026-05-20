package git_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dnlopes/overseer/internal/adapters/secondary/git"
	"github.com/dnlopes/overseer/internal/core/domain"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// seedRepo initialises a fresh git repository inside dir with a single commit
// on the "main" branch. It returns the absolute repository path.
func seedRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitCommands := [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "test@overseer.local"},
		{"config", "user.name", "Overseer Test"},
		{"commit", "--allow-empty", "-q", "-m", "init"},
	}
	for _, args := range gitCommands {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestAdapter_NewReturnsAdapterWhenGitPresent(t *testing.T) {
	a, err := git.New(discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if a == nil {
		t.Fatal("New() returned nil adapter")
	}
}

func TestAdapter_CreateWorktreeCreatesBranchedWorktree(t *testing.T) {
	a, err := git.New(discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	repo := seedRepo(t)
	worktree := filepath.Join(t.TempDir(), "wt-alpha")

	if err := a.CreateWorktree(context.Background(), repo, "main", "overseer/alpha", worktree); err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}

	if info, err := os.Stat(worktree); err != nil {
		t.Fatalf("worktree path missing: %v", err)
	} else if !info.IsDir() {
		t.Fatalf("worktree path is not a directory: %v", info.Mode())
	}

	head, err := exec.Command("git", "-C", worktree, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD in worktree: %v", err)
	}
	if got, want := string(head), "overseer/alpha\n"; got != want {
		t.Fatalf("worktree HEAD = %q, want %q", got, want)
	}
}

func TestAdapter_CreateWorktreeFailsForMissingBaseBranch(t *testing.T) {
	a, err := git.New(discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	repo := seedRepo(t)
	worktree := filepath.Join(t.TempDir(), "wt-alpha")

	err = a.CreateWorktree(context.Background(), repo, "no-such-branch", "overseer/alpha", worktree)
	if err == nil {
		t.Fatalf("CreateWorktree() error = nil, want non-nil for missing base branch")
	}
}

func TestAdapter_RemoveWorktreeRemovesIt(t *testing.T) {
	a, err := git.New(discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	repo := seedRepo(t)
	worktree := filepath.Join(t.TempDir(), "wt-alpha")

	if err := a.CreateWorktree(context.Background(), repo, "main", "overseer/alpha", worktree); err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}
	if err := a.RemoveWorktree(context.Background(), repo, worktree); err != nil {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}

	if _, err := os.Stat(worktree); !os.IsNotExist(err) {
		t.Fatalf("worktree path still exists or unexpected error: %v", err)
	}
}

func TestAdapter_IsGitRepoReturnsNilForRealRepo(t *testing.T) {
	a, err := git.New(discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	repo := seedRepo(t)
	if err := a.IsGitRepo(context.Background(), repo); err != nil {
		t.Fatalf("IsGitRepo() error = %v, want nil", err)
	}
}

func TestAdapter_IsGitRepoReturnsSentinelForNonRepo(t *testing.T) {
	a, err := git.New(discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	dir := t.TempDir()
	err = a.IsGitRepo(context.Background(), dir)
	if !errors.Is(err, domain.ErrProjectNotGitRepo) {
		t.Fatalf("IsGitRepo() error = %v, want wrapped %v", err, domain.ErrProjectNotGitRepo)
	}
}

func TestAdapter_IsGitRepoReturnsSentinelForMissingPath(t *testing.T) {
	a, err := git.New(discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	err = a.IsGitRepo(context.Background(), filepath.Join(t.TempDir(), "does-not-exist"))
	if !errors.Is(err, domain.ErrProjectNotGitRepo) {
		t.Fatalf("IsGitRepo() error = %v, want wrapped %v", err, domain.ErrProjectNotGitRepo)
	}
}
