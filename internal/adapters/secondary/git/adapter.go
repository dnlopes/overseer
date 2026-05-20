// Package git provides git adapter implementations of the domain.GitAdapter port.
package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/dnlopes/overseer/internal/core/domain"
)

var _ domain.GitAdapter = (*Adapter)(nil)

// Adapter drives a local git installation by invoking the git binary directly.
type Adapter struct {
	gitBin string
	logger *slog.Logger
}

// New constructs an Adapter using the `git` binary discovered on PATH.
// Returns an error if git is not installed on the system.
func New(logger *slog.Logger) (*Adapter, error) {
	path, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git: not found on PATH: %w", err)
	}
	return &Adapter{gitBin: path, logger: logger}, nil
}

// CreateWorktree runs `git -C <repoPath> worktree add -b <featureBranch>
// <worktreePath> <baseBranch>`, creating featureBranch from baseBranch at
// worktreePath.
func (a *Adapter) CreateWorktree(_ context.Context, repoPath, baseBranch, featureBranch, worktreePath string) error {
	if _, err := a.runIn(repoPath, "worktree", "add", "-b", featureBranch, worktreePath, baseBranch); err != nil {
		return fmt.Errorf("git: create worktree %q: %w", worktreePath, err)
	}
	a.logger.Debug("git worktree created",
		"repo", repoPath,
		"base_branch", baseBranch,
		"feature_branch", featureBranch,
		"worktree", worktreePath,
	)
	return nil
}

// RemoveWorktree runs `git -C <repoPath> worktree remove --force
// <worktreePath>`. The --force flag is used so worktrees with uncommitted
// changes are not stranded.
func (a *Adapter) RemoveWorktree(_ context.Context, repoPath, worktreePath string) error {
	if _, err := a.runIn(repoPath, "worktree", "remove", "--force", worktreePath); err != nil {
		return fmt.Errorf("git: remove worktree %q: %w", worktreePath, err)
	}
	a.logger.Debug("git worktree removed",
		"repo", repoPath,
		"worktree", worktreePath,
	)
	return nil
}

// IsGitRepo verifies path is the root of a working tree by calling
// `git -C <path> rev-parse --is-inside-work-tree`. Returns
// domain.ErrProjectNotGitRepo if path is not a git working tree.
func (a *Adapter) IsGitRepo(_ context.Context, path string) error {
	stdout, err := a.runIn(path, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		if errors.Is(err, errGitNotARepo) {
			return fmt.Errorf("%w: %s", domain.ErrProjectNotGitRepo, path)
		}
		return fmt.Errorf("git: is-git-repo %q: %w", path, err)
	}
	if strings.TrimSpace(stdout) != "true" {
		return fmt.Errorf("%w: %s", domain.ErrProjectNotGitRepo, path)
	}
	return nil
}

// errGitNotARepo is returned by run when git reports the target path is not a
// repository (or any directory above it).
var errGitNotARepo = errors.New("git: not a repository")

// runIn executes the git binary against repoPath via `git -C <repoPath>`,
// returning stdout. "not a git repository" stderr is mapped to errGitNotARepo
// so callers can map it to domain.ErrProjectNotGitRepo.
func (a *Adapter) runIn(repoPath string, args ...string) (string, error) {
	full := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command(a.gitBin, full...)
	stdout, err := cmd.Output()
	if err == nil {
		return string(stdout), nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr := strings.TrimSpace(string(exitErr.Stderr))
		if strings.Contains(stderr, "not a git repository") || strings.Contains(stderr, "cannot change to") {
			return "", errGitNotARepo
		}
		return "", fmt.Errorf("%w: %s", err, stderr)
	}
	return "", err
}
