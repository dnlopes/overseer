package session

import (
	"context"
	"errors"
	"testing"

	domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
	"github.com/dnlopes/overseer/internal/shared/errs"
	"github.com/dnlopes/overseer/internal/testutil/fixtures"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func assertSessionOrder(t *testing.T, sessions []domainsession.Session, name string, wantOrder int) {
	t.Helper()
	for _, s := range sessions {
		if s.Name == name {
			if s.Order != wantOrder {
				t.Fatalf("%q: Order = %d, want %d", name, s.Order, wantOrder)
			}
			return
		}
	}
	t.Fatalf("session %q not found in response", name)
}

func TestReorderUseCase_MoveDown(t *testing.T) {
	a := fixtures.MakeSession("A", "proj")
	a.Order = 1
	b := fixtures.MakeSession("B", "proj")
	b.Order = 2
	c := fixtures.MakeSession("C", "proj")
	c.Order = 3

	repo := &mocks.MockSessionRepository{
		GetResult:  b,
		ListResult: []domainsession.Session{a, b, c},
	}
	uc := NewReorderUseCase(repo, testLogger())

	resp, err := uc.Execute(context.Background(), ReorderRequest{ID: b.ID, Direction: 1})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repo.SaveCalls != 2 {
		t.Fatalf("SaveCalls = %d, want 2", repo.SaveCalls)
	}
	if len(resp.Sessions) != 3 {
		t.Fatalf("len(resp.Sessions) = %d, want 3", len(resp.Sessions))
	}
	assertSessionOrder(t, resp.Sessions, "A", 1)
	assertSessionOrder(t, resp.Sessions, "C", 2)
	assertSessionOrder(t, resp.Sessions, "B", 3)
}

func TestReorderUseCase_MoveUp(t *testing.T) {
	a := fixtures.MakeSession("A", "proj")
	a.Order = 1
	b := fixtures.MakeSession("B", "proj")
	b.Order = 2
	c := fixtures.MakeSession("C", "proj")
	c.Order = 3

	repo := &mocks.MockSessionRepository{
		GetResult:  b,
		ListResult: []domainsession.Session{a, b, c},
	}
	uc := NewReorderUseCase(repo, testLogger())

	resp, err := uc.Execute(context.Background(), ReorderRequest{ID: b.ID, Direction: -1})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repo.SaveCalls != 2 {
		t.Fatalf("SaveCalls = %d, want 2", repo.SaveCalls)
	}
	if len(resp.Sessions) != 3 {
		t.Fatalf("len(resp.Sessions) = %d, want 3", len(resp.Sessions))
	}
	assertSessionOrder(t, resp.Sessions, "B", 1)
	assertSessionOrder(t, resp.Sessions, "A", 2)
	assertSessionOrder(t, resp.Sessions, "C", 3)
}

func TestReorderUseCase_BoundaryFirst_Up(t *testing.T) {
	a := fixtures.MakeSession("A", "proj")
	a.Order = 1
	b := fixtures.MakeSession("B", "proj")
	b.Order = 2
	c := fixtures.MakeSession("C", "proj")
	c.Order = 3

	repo := &mocks.MockSessionRepository{
		GetResult:  a,
		ListResult: []domainsession.Session{a, b, c},
	}
	uc := NewReorderUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), ReorderRequest{ID: a.ID, Direction: -1})

	if !errors.Is(err, errs.ErrNoOp) {
		t.Fatalf("Execute() error = %v, want %v", err, errs.ErrNoOp)
	}
	if repo.SaveCalls != 0 {
		t.Fatalf("SaveCalls = %d, want 0 (Save must not be called on boundary)", repo.SaveCalls)
	}
}

func TestReorderUseCase_BoundaryLast_Down(t *testing.T) {
	a := fixtures.MakeSession("A", "proj")
	a.Order = 1
	b := fixtures.MakeSession("B", "proj")
	b.Order = 2
	c := fixtures.MakeSession("C", "proj")
	c.Order = 3

	repo := &mocks.MockSessionRepository{
		GetResult:  c,
		ListResult: []domainsession.Session{a, b, c},
	}
	uc := NewReorderUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), ReorderRequest{ID: c.ID, Direction: 1})

	if !errors.Is(err, errs.ErrNoOp) {
		t.Fatalf("Execute() error = %v, want %v", err, errs.ErrNoOp)
	}
	if repo.SaveCalls != 0 {
		t.Fatalf("SaveCalls = %d, want 0 (Save must not be called on boundary)", repo.SaveCalls)
	}
}

func TestReorderUseCase_SingleSession(t *testing.T) {
	a := fixtures.MakeSession("A", "proj")
	a.Order = 1

	repo := &mocks.MockSessionRepository{
		GetResult:  a,
		ListResult: []domainsession.Session{a},
	}
	uc := NewReorderUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), ReorderRequest{ID: a.ID, Direction: 1})

	if !errors.Is(err, errs.ErrNoOp) {
		t.Fatalf("Execute() error = %v, want %v", err, errs.ErrNoOp)
	}
	if repo.SaveCalls != 0 {
		t.Fatalf("SaveCalls = %d, want 0 (Save must not be called for single-session group)", repo.SaveCalls)
	}
}

func TestReorderUseCase_NotFound(t *testing.T) {
	repo := &mocks.MockSessionRepository{
		GetErr: domainsession.ErrNotFound,
	}
	uc := NewReorderUseCase(repo, testLogger())

	_, err := uc.Execute(context.Background(), ReorderRequest{ID: fixtures.MakeSession("X", "proj").ID, Direction: 1})

	if !errors.Is(err, domainsession.ErrNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domainsession.ErrNotFound)
	}
	if repo.SaveCalls != 0 {
		t.Fatalf("SaveCalls = %d, want 0", repo.SaveCalls)
	}
}
