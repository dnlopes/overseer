package session

import (
	"context"
	"errors"
	"testing"
	"time"

	domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
	"github.com/dnlopes/overseer/internal/testutil/fixtures"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func TestRenameUseCase_HappyPath(t *testing.T) {
	original := fixtures.MakeSession("alpha", "overseer")
	repo := &mocks.MockSessionRepository{
		GetResult:  original,
		ListResult: []domainsession.Session{original},
	}
	uc := NewRenameUseCase(repo, testLogger())

	resp, err := uc.Execute(context.Background(), RenameRequest{ID: original.ID, NewName: "beta"})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Session.Name != "beta" {
		t.Fatalf("Execute() Session.Name = %q, want %q", resp.Session.Name, "beta")
	}
	if repo.GetCalls != 1 {
		t.Fatalf("Repository.Get calls = %d, want 1", repo.GetCalls)
	}
	if repo.ListCalls != 1 {
		t.Fatalf("Repository.List calls = %d, want 1", repo.ListCalls)
	}
	if repo.SaveCalls != 1 {
		t.Fatalf("Repository.Save calls = %d, want 1", repo.SaveCalls)
	}
	if repo.SavedSession.Name != "beta" {
		t.Fatalf("Repository.Save Session.Name = %q, want %q", repo.SavedSession.Name, "beta")
	}
}

func TestRenameUseCase_EmptyName(t *testing.T) {
	original := fixtures.MakeSession("alpha", "overseer")
	repo := &mocks.MockSessionRepository{GetResult: original}
	uc := NewRenameUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), RenameRequest{ID: original.ID, NewName: ""})

	if !errors.Is(err, domainsession.ErrEmptyName) {
		t.Fatalf("Execute() error = %v, want %v", err, domainsession.ErrEmptyName)
	}
	if repo.SaveCalls != 0 {
		t.Fatalf("Repository.Save calls = %d, want 0 on validation failure", repo.SaveCalls)
	}
}

func TestRenameUseCase_NotFound(t *testing.T) {
	repo := &mocks.MockSessionRepository{GetErr: domainsession.ErrNotFound}
	uc := NewRenameUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), RenameRequest{ID: fixtures.MakeSession("x", "p").ID, NewName: "beta"})

	if !errors.Is(err, domainsession.ErrNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domainsession.ErrNotFound)
	}
	if repo.SaveCalls != 0 {
		t.Fatalf("Repository.Save calls = %d, want 0 when not found", repo.SaveCalls)
	}
}

func TestRenameUseCase_DuplicateName(t *testing.T) {
	original := fixtures.MakeSession("alpha", "overseer")
	conflicting := fixtures.MakeSession("beta", "overseer")
	repo := &mocks.MockSessionRepository{
		GetResult:  original,
		ListResult: []domainsession.Session{original, conflicting},
	}
	uc := NewRenameUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), RenameRequest{ID: original.ID, NewName: "beta"})

	if !errors.Is(err, domainsession.ErrAlreadyExists) {
		t.Fatalf("Execute() error = %v, want %v", err, domainsession.ErrAlreadyExists)
	}
	if repo.SaveCalls != 0 {
		t.Fatalf("Repository.Save calls = %d, want 0 on duplicate", repo.SaveCalls)
	}
}

func TestRenameUseCase_UpdatedAtChanges(t *testing.T) {
	original := fixtures.MakeSession("alpha", "overseer")
	original.UpdatedAt = time.Now().Add(-time.Minute)
	beforeRename := original.UpdatedAt

	repo := &mocks.MockSessionRepository{
		GetResult:  original,
		ListResult: []domainsession.Session{original},
	}
	uc := NewRenameUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), RenameRequest{ID: original.ID, NewName: "beta"})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !repo.SavedSession.UpdatedAt.After(beforeRename) {
		t.Fatalf("SavedSession.UpdatedAt = %v, want after %v", repo.SavedSession.UpdatedAt, beforeRename)
	}
}
