package session

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
	"github.com/dnlopes/overseer/internal/testutil/fixtures"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func TestCreateUseCase_HappyPath(t *testing.T) {
	repo := &mocks.MockSessionRepository{}
	tmux := &mocks.MockTmuxAdapter{CreateSessionResult: "tmux-alpha"}
	git := &mocks.MockGitAdapter{}
	uc := NewCreateUseCase(repo, tmux, git, testLogger())

	resp, err := uc.Execute(context.Background(), CreateRequest{Name: "alpha", ProjectName: "overseer"})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Session.Name != "alpha" {
		t.Fatalf("Execute() Session.Name = %q, want %q", resp.Session.Name, "alpha")
	}
	if resp.Session.ProjectName != "overseer" {
		t.Fatalf("Execute() Session.ProjectName = %q, want %q", resp.Session.ProjectName, "overseer")
	}
	if resp.Session.Order != 1 {
		t.Fatalf("Execute() Session.Order = %d, want 1", resp.Session.Order)
	}
	if repo.ListCalls != 1 {
		t.Fatalf("Repository.List calls = %d, want 1", repo.ListCalls)
	}
	if tmux.CreateSessionCalls != 1 {
		t.Fatalf("Tmux.CreateSession calls = %d, want 1", tmux.CreateSessionCalls)
	}
	if git.CreateWorktreeCalls != 1 {
		t.Fatalf("Git.CreateWorktree calls = %d, want 1", git.CreateWorktreeCalls)
	}
	if repo.SaveCalls != 1 {
		t.Fatalf("Repository.Save calls = %d, want 1", repo.SaveCalls)
	}
	if repo.SavedSession.Name != "alpha" || repo.SavedSession.ProjectName != "overseer" || repo.SavedSession.Order != 1 {
		t.Fatalf("Repository.Save session = %#v, want alpha/overseer order 1", repo.SavedSession)
	}
}

func TestCreateUseCase_EmptyName(t *testing.T) {
	repo := &mocks.MockSessionRepository{}
	tmux := &mocks.MockTmuxAdapter{}
	git := &mocks.MockGitAdapter{}
	uc := NewCreateUseCase(repo, tmux, git, testLogger())

	_, err := uc.Execute(context.Background(), CreateRequest{Name: "", ProjectName: "overseer"})

	if !errors.Is(err, domainsession.ErrEmptyName) {
		t.Fatalf("Execute() error = %v, want %v", err, domainsession.ErrEmptyName)
	}
	if repo.ListCalls != 0 || tmux.CreateSessionCalls != 0 || git.CreateWorktreeCalls != 0 || repo.SaveCalls != 0 {
		t.Fatalf("mocks called on validation failure: list=%d tmux=%d git=%d save=%d", repo.ListCalls, tmux.CreateSessionCalls, git.CreateWorktreeCalls, repo.SaveCalls)
	}
}

func TestCreateUseCase_DuplicateName(t *testing.T) {
	existing := fixtures.MakeSession("alpha", "overseer")
	repo := &mocks.MockSessionRepository{ListResult: []domainsession.Session{existing}}
	tmux := &mocks.MockTmuxAdapter{}
	git := &mocks.MockGitAdapter{}
	uc := NewCreateUseCase(repo, tmux, git, testLogger())

	_, err := uc.Execute(context.Background(), CreateRequest{Name: "alpha", ProjectName: "overseer"})

	if !errors.Is(err, domainsession.ErrAlreadyExists) {
		t.Fatalf("Execute() error = %v, want %v", err, domainsession.ErrAlreadyExists)
	}
	if repo.ListCalls != 1 {
		t.Fatalf("Repository.List calls = %d, want 1", repo.ListCalls)
	}
	if tmux.CreateSessionCalls != 0 || git.CreateWorktreeCalls != 0 || repo.SaveCalls != 0 {
		t.Fatalf("mocks called on duplicate: tmux=%d git=%d save=%d", tmux.CreateSessionCalls, git.CreateWorktreeCalls, repo.SaveCalls)
	}
}

func TestCreateUseCase_OrderIncrement(t *testing.T) {
	first := fixtures.MakeSession("alpha", "overseer")
	first.Order = 1
	second := fixtures.MakeSession("beta", "overseer")
	second.Order = 2
	otherProject := fixtures.MakeSession("gamma", "other")
	otherProject.Order = 9
	repo := &mocks.MockSessionRepository{ListResult: []domainsession.Session{first, second, otherProject}}
	tmux := &mocks.MockTmuxAdapter{}
	git := &mocks.MockGitAdapter{}
	uc := NewCreateUseCase(repo, tmux, git, testLogger())

	resp, err := uc.Execute(context.Background(), CreateRequest{Name: "gamma", ProjectName: "overseer"})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Session.Order != 3 {
		t.Fatalf("Execute() Session.Order = %d, want 3", resp.Session.Order)
	}
	if repo.SavedSession.Order != 3 {
		t.Fatalf("Repository.Save Order = %d, want 3", repo.SavedSession.Order)
	}
}

func TestCreateUseCase_TmuxError(t *testing.T) {
	tmuxErr := errors.New("tmux unavailable")
	repo := &mocks.MockSessionRepository{}
	tmux := &mocks.MockTmuxAdapter{CreateSessionErr: tmuxErr}
	git := &mocks.MockGitAdapter{}
	uc := NewCreateUseCase(repo, tmux, git, testLogger())

	_, err := uc.Execute(context.Background(), CreateRequest{Name: "alpha", ProjectName: "overseer"})

	if !errors.Is(err, tmuxErr) {
		t.Fatalf("Execute() error = %v, want wrapped %v", err, tmuxErr)
	}
	if repo.SaveCalls != 0 {
		t.Fatalf("Repository.Save calls = %d, want 0", repo.SaveCalls)
	}
	if git.CreateWorktreeCalls != 0 {
		t.Fatalf("Git.CreateWorktree calls = %d, want 0", git.CreateWorktreeCalls)
	}
}

func TestCreateUseCase_FirstSessionOrder(t *testing.T) {
	otherProject := fixtures.MakeSession("alpha", "other")
	otherProject.Order = 4
	repo := &mocks.MockSessionRepository{ListResult: []domainsession.Session{otherProject}}
	tmux := &mocks.MockTmuxAdapter{}
	git := &mocks.MockGitAdapter{}
	uc := NewCreateUseCase(repo, tmux, git, testLogger())

	resp, err := uc.Execute(context.Background(), CreateRequest{Name: "alpha", ProjectName: "overseer"})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Session.Order != 1 {
		t.Fatalf("Execute() Session.Order = %d, want 1", resp.Session.Order)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
