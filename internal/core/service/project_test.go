package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/testutil"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func newProjectMocks(t *testing.T) (*mocks.MockProjectRepository, *mocks.MockGitAdapter) {
	t.Helper()
	return mocks.NewMockProjectRepository(t), mocks.NewMockGitAdapter(t)
}

// --- Register ---

func TestProjectService_Register_HappyPath(t *testing.T) {
	repo, git := newProjectMocks(t)
	repo.EXPECT().GetByPath(mock.Anything, "/repo/overseer").
		Return(domain.Project{}, domain.ErrProjectNotFound).Once()
	git.EXPECT().IsGitRepo(mock.Anything, "/repo/overseer").Return(nil).Once()

	var savedProject domain.Project
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, p domain.Project) { savedProject = p }).
		Return(nil).Once()

	svc := NewProjectService(repo, git, testLogger())
	resp, err := svc.Register(context.Background(), RegisterProjectRequest{Path: "/repo/overseer", Name: "Overseer"})

	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if resp.Project.Name != "Overseer" {
		t.Fatalf("Register() Project.Name = %q, want %q", resp.Project.Name, "Overseer")
	}
	if resp.Project.Path != "/repo/overseer" {
		t.Fatalf("Register() Project.Path = %q, want %q", resp.Project.Path, "/repo/overseer")
	}
	if savedProject.Name != "Overseer" {
		t.Fatalf("Saved Project.Name = %q, want %q", savedProject.Name, "Overseer")
	}
}

func TestProjectService_Register_DerivesNameFromPath(t *testing.T) {
	repo, git := newProjectMocks(t)
	repo.EXPECT().GetByPath(mock.Anything, "/repo/widgets").
		Return(domain.Project{}, domain.ErrProjectNotFound).Once()
	git.EXPECT().IsGitRepo(mock.Anything, "/repo/widgets").Return(nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := NewProjectService(repo, git, testLogger())
	resp, err := svc.Register(context.Background(), RegisterProjectRequest{Path: "/repo/widgets", Name: ""})

	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if resp.Project.Name != "widgets" {
		t.Fatalf("Register() Project.Name = %q, want %q (derived)", resp.Project.Name, "widgets")
	}
}

func TestProjectService_Register_RejectsEmptyPath(t *testing.T) {
	repo, git := newProjectMocks(t)
	svc := NewProjectService(repo, git, testLogger())

	_, err := svc.Register(context.Background(), RegisterProjectRequest{Path: "", Name: ""})

	if !errors.Is(err, domain.ErrProjectEmptyPath) {
		t.Fatalf("Register() error = %v, want %v", err, domain.ErrProjectEmptyPath)
	}
}

func TestProjectService_Register_RejectsRelativePath(t *testing.T) {
	repo, git := newProjectMocks(t)
	svc := NewProjectService(repo, git, testLogger())

	_, err := svc.Register(context.Background(), RegisterProjectRequest{Path: "repos/x", Name: ""})

	if !errors.Is(err, domain.ErrProjectPathNotAbsolute) {
		t.Fatalf("Register() error = %v, want %v", err, domain.ErrProjectPathNotAbsolute)
	}
}

func TestProjectService_Register_RejectsNonGitRepo(t *testing.T) {
	repo, git := newProjectMocks(t)
	repo.EXPECT().GetByPath(mock.Anything, "/not/a/repo").
		Return(domain.Project{}, domain.ErrProjectNotFound).Once()
	git.EXPECT().IsGitRepo(mock.Anything, "/not/a/repo").
		Return(domain.ErrProjectNotGitRepo).Once()

	svc := NewProjectService(repo, git, testLogger())
	_, err := svc.Register(context.Background(), RegisterProjectRequest{Path: "/not/a/repo", Name: ""})

	if !errors.Is(err, domain.ErrProjectNotGitRepo) {
		t.Fatalf("Register() error = %v, want %v", err, domain.ErrProjectNotGitRepo)
	}
}

func TestProjectService_Register_RejectsDuplicatePath(t *testing.T) {
	existing := testutil.MakeProject("/repo/overseer", "OldName")
	repo, git := newProjectMocks(t)
	repo.EXPECT().GetByPath(mock.Anything, "/repo/overseer").
		Return(existing, nil).Once()

	svc := NewProjectService(repo, git, testLogger())
	_, err := svc.Register(context.Background(), RegisterProjectRequest{Path: "/repo/overseer", Name: "NewName"})

	if !errors.Is(err, domain.ErrProjectAlreadyExists) {
		t.Fatalf("Register() error = %v, want %v", err, domain.ErrProjectAlreadyExists)
	}
}

func TestProjectService_Register_GetByPathInfrastructureError(t *testing.T) {
	infraErr := errors.New("disk failed")
	repo, git := newProjectMocks(t)
	repo.EXPECT().GetByPath(mock.Anything, "/repo/overseer").
		Return(domain.Project{}, infraErr).Once()

	svc := NewProjectService(repo, git, testLogger())
	_, err := svc.Register(context.Background(), RegisterProjectRequest{Path: "/repo/overseer", Name: ""})

	if !errors.Is(err, infraErr) {
		t.Fatalf("Register() error = %v, want wrapped %v", err, infraErr)
	}
}

// --- List ---

func TestProjectService_List_Empty(t *testing.T) {
	repo, git := newProjectMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()

	svc := NewProjectService(repo, git, testLogger())
	resp, err := svc.List(context.Background(), ListProjectsRequest{})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(resp.Projects) != 0 {
		t.Fatalf("List() len = %d, want 0", len(resp.Projects))
	}
}

func TestProjectService_List_SortsByName(t *testing.T) {
	a := testutil.MakeProject("/r/zeta", "Zeta")
	b := testutil.MakeProject("/r/alpha", "Alpha")
	c := testutil.MakeProject("/r/mike", "Mike")
	repo, git := newProjectMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Project{a, b, c}, nil).Once()

	svc := NewProjectService(repo, git, testLogger())
	resp, err := svc.List(context.Background(), ListProjectsRequest{})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(resp.Projects) != 3 {
		t.Fatalf("List() len = %d, want 3", len(resp.Projects))
	}
	wantOrder := []string{"Alpha", "Mike", "Zeta"}
	for i, want := range wantOrder {
		if resp.Projects[i].Name != want {
			t.Fatalf("Projects[%d].Name = %q, want %q", i, resp.Projects[i].Name, want)
		}
	}
}
