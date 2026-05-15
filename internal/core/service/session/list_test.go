package session

import (
	"context"
	"testing"

	domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
	"github.com/dnlopes/overseer/internal/testutil/fixtures"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func TestListUseCase_Empty(t *testing.T) {
	repo := &mocks.MockSessionRepository{}
	uc := NewListUseCase(repo)

	resp, err := uc.Execute(context.Background(), ListRequest{})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Groups) != 0 {
		t.Fatalf("Execute() len(Groups) = %d, want 0", len(resp.Groups))
	}
	if repo.ListCalls != 1 {
		t.Fatalf("Repository.List calls = %d, want 1", repo.ListCalls)
	}
}

func TestListUseCase_SingleProject(t *testing.T) {
	s1 := fixtures.MakeSession("alpha", "overseer")
	s1.Order = 2
	s2 := fixtures.MakeSession("beta", "overseer")
	s2.Order = 1
	s3 := fixtures.MakeSession("gamma", "overseer")
	s3.Order = 3
	repo := &mocks.MockSessionRepository{ListResult: []domainsession.Session{s1, s2, s3}}
	uc := NewListUseCase(repo)

	resp, err := uc.Execute(context.Background(), ListRequest{})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Groups) != 1 {
		t.Fatalf("Execute() len(Groups) = %d, want 1", len(resp.Groups))
	}
	if resp.Groups[0].ProjectName != "overseer" {
		t.Fatalf("Groups[0].ProjectName = %q, want %q", resp.Groups[0].ProjectName, "overseer")
	}
	if len(resp.Groups[0].Sessions) != 3 {
		t.Fatalf("Groups[0] len(Sessions) = %d, want 3", len(resp.Groups[0].Sessions))
	}
	if resp.Groups[0].Sessions[0].Name != "beta" {
		t.Fatalf("Sessions[0].Name = %q, want %q", resp.Groups[0].Sessions[0].Name, "beta")
	}
	if resp.Groups[0].Sessions[1].Name != "alpha" {
		t.Fatalf("Sessions[1].Name = %q, want %q", resp.Groups[0].Sessions[1].Name, "alpha")
	}
	if resp.Groups[0].Sessions[2].Name != "gamma" {
		t.Fatalf("Sessions[2].Name = %q, want %q", resp.Groups[0].Sessions[2].Name, "gamma")
	}
}

func TestListUseCase_MultipleProjects(t *testing.T) {
	s1 := fixtures.MakeSession("alpha", "bravo")
	s1.Order = 1
	s2 := fixtures.MakeSession("beta", "alpha")
	s2.Order = 1
	repo := &mocks.MockSessionRepository{ListResult: []domainsession.Session{s1, s2}}
	uc := NewListUseCase(repo)

	resp, err := uc.Execute(context.Background(), ListRequest{})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Groups) != 2 {
		t.Fatalf("Execute() len(Groups) = %d, want 2", len(resp.Groups))
	}
	if resp.Groups[0].ProjectName != "alpha" {
		t.Fatalf("Groups[0].ProjectName = %q, want %q", resp.Groups[0].ProjectName, "alpha")
	}
	if resp.Groups[1].ProjectName != "bravo" {
		t.Fatalf("Groups[1].ProjectName = %q, want %q", resp.Groups[1].ProjectName, "bravo")
	}
}

func TestListUseCase_OrderWithinGroup(t *testing.T) {
	s1 := fixtures.MakeSession("first", "overseer")
	s1.Order = 10
	s2 := fixtures.MakeSession("second", "overseer")
	s2.Order = 3
	s3 := fixtures.MakeSession("third", "overseer")
	s3.Order = 7
	repo := &mocks.MockSessionRepository{ListResult: []domainsession.Session{s1, s2, s3}}
	uc := NewListUseCase(repo)

	resp, err := uc.Execute(context.Background(), ListRequest{})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Groups) != 1 {
		t.Fatalf("Execute() len(Groups) = %d, want 1", len(resp.Groups))
	}
	sessions := resp.Groups[0].Sessions
	if sessions[0].Order != 3 || sessions[1].Order != 7 || sessions[2].Order != 10 {
		t.Fatalf("Sessions not sorted by Order ASC: got %d,%d,%d, want 3,7,10",
			sessions[0].Order, sessions[1].Order, sessions[2].Order)
	}
}
