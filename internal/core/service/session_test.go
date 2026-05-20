package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/shared/errs"
	"github.com/dnlopes/overseer/internal/shared/paths"
	"github.com/dnlopes/overseer/internal/testutil"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func assertSessionOrder(t *testing.T, sessions []domain.Session, name string, wantOrder int) {
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

func newSessionMocks(t *testing.T) (*mocks.MockSessionRepository, *mocks.MockProjectRepository, *mocks.MockTmuxAdapter, *mocks.MockGitAdapter) {
	t.Helper()
	return mocks.NewMockSessionRepository(t),
		mocks.NewMockProjectRepository(t),
		mocks.NewMockTmuxAdapter(t),
		mocks.NewMockGitAdapter(t)
}

func newTestSessionService(
	repo domain.SessionRepository,
	projects domain.ProjectRepository,
	tmux domain.TmuxAdapter,
	git domain.GitAdapter,
	logger *slog.Logger,
) *SessionService {
	launcher, _ := domain.NewLauncher("OpenCode", "opencode")
	return NewSessionService(repo, projects, tmux, git, paths.NewResolver(""), launcher, logger)
}

// --- Create ---

// expectProjectLookup wires the project repository mock to return a project
// whose Path is "/repo/<name>" for the given project ID. The Path is the
// repoPath the service will hand to git.CreateWorktree.
func expectProjectLookup(t *testing.T, projects *mocks.MockProjectRepository, projectID uuid.UUID, name string) string {
	t.Helper()
	repoPath := "/repo/" + name
	project := testutil.MakeProject(repoPath, name)
	project.ID = projectID
	projects.EXPECT().Get(mock.Anything, projectID).Return(project, nil).Once()
	return repoPath
}

func TestSessionService_Create_HappyPath(t *testing.T) {
	overseerID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)

	repoPath := expectProjectLookup(t, projects, overseerID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: overseerID})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.Session.Name != "alpha" {
		t.Fatalf("Create() Session.Name = %q, want %q", resp.Session.Name, "alpha")
	}
	if resp.Session.ProjectID != overseerID {
		t.Fatalf("Create() Session.ProjectID = %v, want %v", resp.Session.ProjectID, overseerID)
	}
	if resp.Session.Order != 1 {
		t.Fatalf("Create() Session.Order = %d, want 1", resp.Session.Order)
	}
	if !resp.Session.HasWorktree() {
		t.Fatalf("Create() Session.HasWorktree() = false, want true for project-backed session")
	}
	wantBranch := paths.SessionFeatureBranch(resp.Session.ID)
	if resp.Session.FeatureBranch != wantBranch {
		t.Fatalf("Create() Session.FeatureBranch = %q, want %q", resp.Session.FeatureBranch, wantBranch)
	}
	if resp.Session.BaseBranch != "main" {
		t.Fatalf("Create() Session.BaseBranch = %q, want %q", resp.Session.BaseBranch, "main")
	}
	wantPath := paths.NewResolver("").SessionWorktreePath(resp.Session.ID)
	if resp.Session.WorktreePath != wantPath {
		t.Fatalf("Create() Session.WorktreePath = %q, want %q", resp.Session.WorktreePath, wantPath)
	}
	if savedSession.WorktreePath != wantPath || savedSession.FeatureBranch != wantBranch {
		t.Fatalf("SessionRepository.Save session = %#v, want worktree+branch populated", savedSession)
	}
}

func TestSessionService_Create_WithoutProjectShellsIntoHome(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	repo, projects, tmux, git := newSessionMocks(t)

	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), "/tmp/overseer-home", "").Return("tmux-orphan", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{Name: "orphan", ProjectID: uuid.Nil})

	if err != nil {
		t.Fatalf("Create() error = %v, want nil for project-less session", err)
	}
	if resp.Session.ProjectID != uuid.Nil {
		t.Fatalf("Create() Session.ProjectID = %v, want uuid.Nil", resp.Session.ProjectID)
	}
	if resp.Session.HasWorktree() {
		t.Fatalf("Create() project-less Session.HasWorktree() = true, want false")
	}
	if savedSession.WorktreePath != "" || savedSession.BaseBranch != "" || savedSession.FeatureBranch != "" {
		t.Fatalf("project-less session persisted worktree fields: %#v", savedSession)
	}
}

func TestSessionService_Create_EmptyName(t *testing.T) {
	repo, projects, tmux, git := newSessionMocks(t)
	svc := newTestSessionService(repo, projects, tmux, git, testLogger())

	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "", ProjectID: uuid.New()})

	if !errors.Is(err, domain.ErrSessionEmptyName) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrSessionEmptyName)
	}
}

func TestSessionService_Create_DuplicateNameWithinSameProject(t *testing.T) {
	overseerID := uuid.New()
	existing := testutil.MakeSession("alpha", overseerID)
	repo, projects, tmux, git := newSessionMocks(t)
	expectProjectLookup(t, projects, overseerID, "overseer")
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{existing}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: overseerID})

	if !errors.Is(err, domain.ErrSessionAlreadyExists) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrSessionAlreadyExists)
	}
}

func TestSessionService_Create_DuplicateNameAcrossProjectsAllowed(t *testing.T) {
	overseerID := uuid.New()
	otherID := uuid.New()
	existing := testutil.MakeSession("alpha", otherID)
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, overseerID, "overseer")
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{existing}, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: overseerID})

	if err != nil {
		t.Fatalf("Create() error = %v, want nil (same name in different project is allowed)", err)
	}
}

func TestSessionService_Create_DuplicateNameAmongUnassignedSessions(t *testing.T) {
	existing := testutil.MakeSession("solo", uuid.Nil)
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{existing}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "solo", ProjectID: uuid.Nil})

	if !errors.Is(err, domain.ErrSessionAlreadyExists) {
		t.Fatalf("Create() error = %v, want %v (duplicate in unassigned bucket)", err, domain.ErrSessionAlreadyExists)
	}
}

func TestSessionService_Create_OrderIncrement(t *testing.T) {
	overseerID := uuid.New()
	otherID := uuid.New()
	first := testutil.MakeSession("alpha", overseerID)
	first.Order = 1
	second := testutil.MakeSession("beta", overseerID)
	second.Order = 2
	otherProject := testutil.MakeSession("gamma", otherID)
	otherProject.Order = 9
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, overseerID, "overseer")
	repo.EXPECT().List(mock.Anything).
		Return([]domain.Session{first, second, otherProject}, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-gamma", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{Name: "gamma", ProjectID: overseerID})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.Session.Order != 3 {
		t.Fatalf("Create() Session.Order = %d, want 3", resp.Session.Order)
	}
	if savedSession.Order != 3 {
		t.Fatalf("SessionRepository.Save Order = %d, want 3", savedSession.Order)
	}
}

func TestSessionService_Create_TmuxError(t *testing.T) {
	tmuxErr := errors.New("tmux unavailable")
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, projID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("", tmuxErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: projID})

	if !errors.Is(err, tmuxErr) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, tmuxErr)
	}
}

func TestSessionService_Create_GitError(t *testing.T) {
	gitErr := errors.New("git refused")
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, projID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(gitErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: projID})

	if !errors.Is(err, gitErr) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, gitErr)
	}
}

func TestSessionService_Create_ProjectLookupErrorBubblesUp(t *testing.T) {
	lookupErr := errors.New("project lookup failed")
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	projects.EXPECT().Get(mock.Anything, projID).Return(domain.Project{}, lookupErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: projID})

	if !errors.Is(err, lookupErr) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, lookupErr)
	}
}

func TestSessionService_Create_FirstSessionOrder(t *testing.T) {
	overseerID := uuid.New()
	otherID := uuid.New()
	otherProject := testutil.MakeSession("alpha", otherID)
	otherProject.Order = 4
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, overseerID, "overseer")
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{otherProject}, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: overseerID})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.Session.Order != 1 {
		t.Fatalf("Create() Session.Order = %d, want 1", resp.Session.Order)
	}
}

// --- Rename ---

func TestSessionService_Rename_HappyPath(t *testing.T) {
	original := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, original.ID).Return(original, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{original}, nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Rename(context.Background(), RenameSessionRequest{ID: original.ID, NewName: "beta"})

	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if resp.Session.Name != "beta" {
		t.Fatalf("Rename() Session.Name = %q, want %q", resp.Session.Name, "beta")
	}
	if savedSession.Name != "beta" {
		t.Fatalf("SessionRepository.Save Session.Name = %q, want %q", savedSession.Name, "beta")
	}
}

func TestSessionService_Rename_EmptyName(t *testing.T) {
	original := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, original.ID).Return(original, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{original}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Rename(context.Background(), RenameSessionRequest{ID: original.ID, NewName: ""})

	if !errors.Is(err, domain.ErrSessionEmptyName) {
		t.Fatalf("Rename() error = %v, want %v", err, domain.ErrSessionEmptyName)
	}
}

func TestSessionService_Rename_NotFound(t *testing.T) {
	repo, projects, tmux, git := newSessionMocks(t)
	missingID := uuid.New()
	repo.EXPECT().Get(mock.Anything, missingID).
		Return(domain.Session{}, domain.ErrSessionNotFound).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Rename(context.Background(), RenameSessionRequest{ID: missingID, NewName: "beta"})

	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("Rename() error = %v, want %v", err, domain.ErrSessionNotFound)
	}
}

func TestSessionService_Rename_DuplicateNameInSameProject(t *testing.T) {
	overseerID := uuid.New()
	original := testutil.MakeSession("alpha", overseerID)
	conflicting := testutil.MakeSession("beta", overseerID)
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, original.ID).Return(original, nil).Once()
	repo.EXPECT().List(mock.Anything).
		Return([]domain.Session{original, conflicting}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Rename(context.Background(), RenameSessionRequest{ID: original.ID, NewName: "beta"})

	if !errors.Is(err, domain.ErrSessionAlreadyExists) {
		t.Fatalf("Rename() error = %v, want %v", err, domain.ErrSessionAlreadyExists)
	}
}

func TestSessionService_Rename_UpdatedAtChanges(t *testing.T) {
	original := testutil.MakeSession("alpha", uuid.New())
	original.UpdatedAt = time.Now().Add(-time.Minute)
	beforeRename := original.UpdatedAt

	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, original.ID).Return(original, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{original}, nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Rename(context.Background(), RenameSessionRequest{ID: original.ID, NewName: "beta"})

	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if !savedSession.UpdatedAt.After(beforeRename) {
		t.Fatalf("SavedSession.UpdatedAt = %v, want after %v", savedSession.UpdatedAt, beforeRename)
	}
}

// --- List ---

func TestSessionService_List_Empty(t *testing.T) {
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.List(context.Background(), ListSessionsRequest{})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(resp.Sessions) != 0 {
		t.Fatalf("List() len(Sessions) = %d, want 0", len(resp.Sessions))
	}
}

func TestSessionService_List_SortsByOrderWithinSameProject(t *testing.T) {
	projectID := uuid.New()
	s1 := testutil.MakeSession("alpha", projectID)
	s1.Order = 2
	s2 := testutil.MakeSession("beta", projectID)
	s2.Order = 1
	s3 := testutil.MakeSession("gamma", projectID)
	s3.Order = 3
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{s1, s2, s3}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.List(context.Background(), ListSessionsRequest{})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(resp.Sessions) != 3 {
		t.Fatalf("List() len(Sessions) = %d, want 3", len(resp.Sessions))
	}
	if resp.Sessions[0].Name != "beta" {
		t.Fatalf("Sessions[0].Name = %q, want %q", resp.Sessions[0].Name, "beta")
	}
	if resp.Sessions[1].Name != "alpha" {
		t.Fatalf("Sessions[1].Name = %q, want %q", resp.Sessions[1].Name, "alpha")
	}
	if resp.Sessions[2].Name != "gamma" {
		t.Fatalf("Sessions[2].Name = %q, want %q", resp.Sessions[2].Name, "gamma")
	}
}

func TestSessionService_List_GroupsByProjectID(t *testing.T) {
	lowID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	highID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	highSession := testutil.MakeSession("alpha", highID)
	highSession.Order = 1
	lowSession := testutil.MakeSession("beta", lowID)
	lowSession.Order = 1
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).
		Return([]domain.Session{highSession, lowSession}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.List(context.Background(), ListSessionsRequest{})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(resp.Sessions) != 2 {
		t.Fatalf("List() len(Sessions) = %d, want 2", len(resp.Sessions))
	}
	if resp.Sessions[0].ProjectID != lowID {
		t.Fatalf("Sessions[0].ProjectID = %v, want lowID", resp.Sessions[0].ProjectID)
	}
	if resp.Sessions[1].ProjectID != highID {
		t.Fatalf("Sessions[1].ProjectID = %v, want highID", resp.Sessions[1].ProjectID)
	}
}

func TestSessionService_List_OrderWithinGroup(t *testing.T) {
	projectID := uuid.New()
	s1 := testutil.MakeSession("first", projectID)
	s1.Order = 10
	s2 := testutil.MakeSession("second", projectID)
	s2.Order = 3
	s3 := testutil.MakeSession("third", projectID)
	s3.Order = 7
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{s1, s2, s3}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.List(context.Background(), ListSessionsRequest{})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	sessions := resp.Sessions
	if sessions[0].Order != 3 || sessions[1].Order != 7 || sessions[2].Order != 10 {
		t.Fatalf("Sessions not sorted by Order ASC: got %d,%d,%d, want 3,7,10",
			sessions[0].Order, sessions[1].Order, sessions[2].Order)
	}
}

// --- Reorder ---

func TestSessionService_Reorder_MoveDown(t *testing.T) {
	projectID := uuid.New()
	a := testutil.MakeSession("A", projectID)
	a.Order = 1
	b := testutil.MakeSession("B", projectID)
	b.Order = 2
	c := testutil.MakeSession("C", projectID)
	c.Order = 3

	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, b.ID).Return(b, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a, b, c}, nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Twice()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Reorder(context.Background(), ReorderSessionRequest{ID: b.ID, Direction: 1})

	if err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}
	if len(resp.Sessions) != 3 {
		t.Fatalf("len(resp.Sessions) = %d, want 3", len(resp.Sessions))
	}
	assertSessionOrder(t, resp.Sessions, "A", 1)
	assertSessionOrder(t, resp.Sessions, "C", 2)
	assertSessionOrder(t, resp.Sessions, "B", 3)
}

func TestSessionService_Reorder_MoveUp(t *testing.T) {
	projectID := uuid.New()
	a := testutil.MakeSession("A", projectID)
	a.Order = 1
	b := testutil.MakeSession("B", projectID)
	b.Order = 2
	c := testutil.MakeSession("C", projectID)
	c.Order = 3

	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, b.ID).Return(b, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a, b, c}, nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Twice()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Reorder(context.Background(), ReorderSessionRequest{ID: b.ID, Direction: -1})

	if err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}
	if len(resp.Sessions) != 3 {
		t.Fatalf("len(resp.Sessions) = %d, want 3", len(resp.Sessions))
	}
	assertSessionOrder(t, resp.Sessions, "B", 1)
	assertSessionOrder(t, resp.Sessions, "A", 2)
	assertSessionOrder(t, resp.Sessions, "C", 3)
}

func TestSessionService_Reorder_BoundaryFirst_Up(t *testing.T) {
	projectID := uuid.New()
	a := testutil.MakeSession("A", projectID)
	a.Order = 1
	b := testutil.MakeSession("B", projectID)
	b.Order = 2
	c := testutil.MakeSession("C", projectID)
	c.Order = 3

	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, a.ID).Return(a, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a, b, c}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Reorder(context.Background(), ReorderSessionRequest{ID: a.ID, Direction: -1})

	if !errors.Is(err, errs.ErrNoOp) {
		t.Fatalf("Reorder() error = %v, want %v", err, errs.ErrNoOp)
	}
}

func TestSessionService_Reorder_BoundaryLast_Down(t *testing.T) {
	projectID := uuid.New()
	a := testutil.MakeSession("A", projectID)
	a.Order = 1
	b := testutil.MakeSession("B", projectID)
	b.Order = 2
	c := testutil.MakeSession("C", projectID)
	c.Order = 3

	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, c.ID).Return(c, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a, b, c}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Reorder(context.Background(), ReorderSessionRequest{ID: c.ID, Direction: 1})

	if !errors.Is(err, errs.ErrNoOp) {
		t.Fatalf("Reorder() error = %v, want %v", err, errs.ErrNoOp)
	}
}

func TestSessionService_Reorder_SingleSession(t *testing.T) {
	a := testutil.MakeSession("A", uuid.New())
	a.Order = 1

	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, a.ID).Return(a, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Reorder(context.Background(), ReorderSessionRequest{ID: a.ID, Direction: 1})

	if !errors.Is(err, errs.ErrNoOp) {
		t.Fatalf("Reorder() error = %v, want %v", err, errs.ErrNoOp)
	}
}

func TestSessionService_Reorder_NotFound(t *testing.T) {
	repo, projects, tmux, git := newSessionMocks(t)
	missingID := uuid.New()
	repo.EXPECT().Get(mock.Anything, missingID).
		Return(domain.Session{}, domain.ErrSessionNotFound).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Reorder(context.Background(), ReorderSessionRequest{ID: missingID, Direction: 1})

	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("Reorder() error = %v, want %v", err, domain.ErrSessionNotFound)
	}
}

// --- AttachShell ---

func TestSessionService_AttachShell_HappyPath(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{ID: sess.ID.String()}, nil).Once()
	wantCmd := exec.Command("tmux", "attach-session", "-t", sess.ID.String())
	tmux.EXPECT().AttachCommand(mock.Anything, sess.ID.String()).Return(wantCmd, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.AttachShell(context.Background(), AttachShellRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("AttachShell() error = %v", err)
	}
	if resp.Command != wantCmd {
		t.Fatalf("AttachShell() Command = %v, want %v", resp.Command, wantCmd)
	}
}

func TestSessionService_AttachShell_SessionNotFound(t *testing.T) {
	missingID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, missingID).
		Return(domain.Session{}, domain.ErrSessionNotFound).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.AttachShell(context.Background(), AttachShellRequest{ID: missingID})

	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("AttachShell() error = %v, want %v", err, domain.ErrSessionNotFound)
	}
}

func TestSessionService_AttachShell_TmuxAdapterErrorWrapped(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{ID: sess.ID.String()}, nil).Once()
	adapterErr := errors.New("tmux binary missing")
	tmux.EXPECT().AttachCommand(mock.Anything, sess.ID.String()).
		Return(nil, adapterErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.AttachShell(context.Background(), AttachShellRequest{ID: sess.ID})

	if err == nil || !errors.Is(err, adapterErr) {
		t.Fatalf("AttachShell() error = %v, want wrapped %v", err, adapterErr)
	}
}

func TestSessionService_AttachShell_TmuxSessionMissing_RecreatesThenAttaches(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{}, domain.ErrTmuxSessionNotFound).Once()
	tmux.EXPECT().CreateSession(mock.Anything, sess.ID.String(), "/tmp/overseer-home", "").
		Return(sess.ID.String(), nil).Once()
	wantCmd := exec.Command("tmux", "attach-session", "-t", sess.ID.String())
	tmux.EXPECT().AttachCommand(mock.Anything, sess.ID.String()).Return(wantCmd, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.AttachShell(context.Background(), AttachShellRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("AttachShell() error = %v, want nil after recreate", err)
	}
	if resp.Command != wantCmd {
		t.Fatalf("AttachShell() Command = %v, want %v", resp.Command, wantCmd)
	}
}

func TestSessionService_AttachShell_ProjectBackedRecreatesAtWorktreePath(t *testing.T) {
	worktreePath := "/abs/worktree/alpha"
	sess := testutil.MakeSessionWithWorktree("alpha", uuid.New(), worktreePath, "main", "overseer/alpha")
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{}, domain.ErrTmuxSessionNotFound).Once()
	tmux.EXPECT().CreateSession(mock.Anything, sess.ID.String(), worktreePath, "").
		Return(sess.ID.String(), nil).Once()
	wantCmd := exec.Command("tmux", "attach-session", "-t", sess.ID.String())
	tmux.EXPECT().AttachCommand(mock.Anything, sess.ID.String()).Return(wantCmd, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.AttachShell(context.Background(), AttachShellRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("AttachShell() error = %v, want nil", err)
	}
	if resp.Command != wantCmd {
		t.Fatalf("AttachShell() Command = %v, want %v", resp.Command, wantCmd)
	}
}

func TestSessionService_AttachShell_GetSessionUnexpectedErrorBubblesUp(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	inspectErr := errors.New("tmux server unreachable")
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{}, inspectErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.AttachShell(context.Background(), AttachShellRequest{ID: sess.ID})

	if err == nil || !errors.Is(err, inspectErr) {
		t.Fatalf("AttachShell() error = %v, want wrapped %v", err, inspectErr)
	}
}

func TestSessionService_AttachShell_RecreateErrorBubblesUp(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{}, domain.ErrTmuxSessionNotFound).Once()
	createErr := errors.New("tmux create failed")
	tmux.EXPECT().CreateSession(mock.Anything, sess.ID.String(), "/tmp/overseer-home", "").
		Return("", createErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.AttachShell(context.Background(), AttachShellRequest{ID: sess.ID})

	if err == nil || !errors.Is(err, createErr) {
		t.Fatalf("AttachShell() error = %v, want wrapped %v", err, createErr)
	}
}

func TestSessionService_Create_WithAgentCommand(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), "/tmp/overseer-home", "").
		Return("tmux-alpha", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{
		Name:         "alpha",
		ProjectID:    uuid.Nil,
		AgentCommand: "claude",
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.Session.AgentCommand != "claude" {
		t.Fatalf("Create() Session.AgentCommand = %q, want %q", resp.Session.AgentCommand, "claude")
	}
	if savedSession.AgentCommand != "claude" {
		t.Fatalf("SessionRepository.Save session.AgentCommand = %q, want %q", savedSession.AgentCommand, "claude")
	}
}

func TestSessionService_Create_RejectsInvalidAgentCommand(t *testing.T) {
	repo, projects, tmux, git := newSessionMocks(t)

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{
		Name:         "alpha",
		ProjectID:    uuid.Nil,
		AgentCommand: "   ",
	})

	if !errors.Is(err, domain.ErrSessionEmptyAgentCommand) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrSessionEmptyAgentCommand)
	}
}

// --- AttachAgent ---

func TestSessionService_AttachAgent_HappyPath(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	if err := sess.AssignAgentCommand("opencode"); err != nil {
		t.Fatalf("seed AssignAgentCommand: %v", err)
	}
	agentTmuxID := sess.ID.String() + "-agent"
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, agentTmuxID).
		Return(domain.TmuxSession{ID: agentTmuxID}, nil).Once()
	wantCmd := exec.Command("tmux", "attach-session", "-t", agentTmuxID)
	tmux.EXPECT().AttachCommand(mock.Anything, agentTmuxID).Return(wantCmd, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.AttachAgent(context.Background(), AttachAgentRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("AttachAgent() error = %v", err)
	}
	if resp.Command != wantCmd {
		t.Fatalf("AttachAgent() Command = %v, want %v", resp.Command, wantCmd)
	}
}

func TestSessionService_AttachAgent_AgentTmuxMissing_RecreatesWithCommand(t *testing.T) {
	worktreePath := "/abs/worktree/alpha"
	sess := testutil.MakeSessionWithWorktree("alpha", uuid.New(), worktreePath, "main", "overseer/alpha")
	if err := sess.AssignAgentCommand("opencode --config /tmp/cfg"); err != nil {
		t.Fatalf("seed AssignAgentCommand: %v", err)
	}
	agentTmuxID := sess.ID.String() + "-agent"
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, agentTmuxID).
		Return(domain.TmuxSession{}, domain.ErrTmuxSessionNotFound).Once()
	tmux.EXPECT().CreateSession(mock.Anything, agentTmuxID, worktreePath, "opencode --config /tmp/cfg").
		Return(agentTmuxID, nil).Once()
	wantCmd := exec.Command("tmux", "attach-session", "-t", agentTmuxID)
	tmux.EXPECT().AttachCommand(mock.Anything, agentTmuxID).Return(wantCmd, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.AttachAgent(context.Background(), AttachAgentRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("AttachAgent() error = %v, want nil after recreate", err)
	}
	if resp.Command != wantCmd {
		t.Fatalf("AttachAgent() Command = %v, want %v", resp.Command, wantCmd)
	}
}

func TestSessionService_AttachAgent_ProjectLessRecreatesAtHome(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	sess := testutil.MakeSession("alpha", uuid.Nil)
	if err := sess.AssignAgentCommand("claude"); err != nil {
		t.Fatalf("seed AssignAgentCommand: %v", err)
	}
	agentTmuxID := sess.ID.String() + "-agent"
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, agentTmuxID).
		Return(domain.TmuxSession{}, domain.ErrTmuxSessionNotFound).Once()
	tmux.EXPECT().CreateSession(mock.Anything, agentTmuxID, "/tmp/overseer-home", "claude").
		Return(agentTmuxID, nil).Once()
	wantCmd := exec.Command("tmux", "attach-session", "-t", agentTmuxID)
	tmux.EXPECT().AttachCommand(mock.Anything, agentTmuxID).Return(wantCmd, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.AttachAgent(context.Background(), AttachAgentRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("AttachAgent() error = %v", err)
	}
	if resp.Command != wantCmd {
		t.Fatalf("AttachAgent() Command = %v, want %v", resp.Command, wantCmd)
	}
}

func TestSessionService_AttachAgent_EmptyAgentCommand_FallsBackToOpencode(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	sess := testutil.MakeSession("alpha", uuid.Nil)
	if sess.AgentCommand != "" {
		t.Fatalf("test precondition: AgentCommand must start empty, got %q", sess.AgentCommand)
	}
	agentTmuxID := sess.ID.String() + "-agent"
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, agentTmuxID).
		Return(domain.TmuxSession{}, domain.ErrTmuxSessionNotFound).Once()
	tmux.EXPECT().CreateSession(mock.Anything, agentTmuxID, "/tmp/overseer-home", "opencode").
		Return(agentTmuxID, nil).Once()
	wantCmd := exec.Command("tmux", "attach-session", "-t", agentTmuxID)
	tmux.EXPECT().AttachCommand(mock.Anything, agentTmuxID).Return(wantCmd, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.AttachAgent(context.Background(), AttachAgentRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("AttachAgent() error = %v, want nil with opencode fallback", err)
	}
	if resp.Command != wantCmd {
		t.Fatalf("AttachAgent() Command = %v, want %v", resp.Command, wantCmd)
	}
}

func TestSessionService_AttachAgent_NoSessionCommandAndNoDefaultLauncher_ReturnsSentinel(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	sess := testutil.MakeSession("alpha", uuid.Nil)
	if sess.AgentCommand != "" {
		t.Fatalf("test precondition: AgentCommand must start empty, got %q", sess.AgentCommand)
	}
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()

	svc := NewSessionService(repo, projects, tmux, git, paths.NewResolver(""), domain.Launcher{}, testLogger())
	_, err := svc.AttachAgent(context.Background(), AttachAgentRequest{ID: sess.ID})

	if !errors.Is(err, domain.ErrSessionNoAgentCommandAvailable) {
		t.Fatalf("AttachAgent() error = %v, want ErrSessionNoAgentCommandAvailable", err)
	}
}

func TestSessionService_AttachAgent_SessionNotFound(t *testing.T) {
	missingID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, missingID).
		Return(domain.Session{}, domain.ErrSessionNotFound).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.AttachAgent(context.Background(), AttachAgentRequest{ID: missingID})

	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("AttachAgent() error = %v, want %v", err, domain.ErrSessionNotFound)
	}
}

func TestSessionService_AttachAgent_GetTmuxSessionUnexpectedErrorBubblesUp(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	if err := sess.AssignAgentCommand("opencode"); err != nil {
		t.Fatalf("seed AssignAgentCommand: %v", err)
	}
	agentTmuxID := sess.ID.String() + "-agent"
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	inspectErr := errors.New("tmux server unreachable")
	tmux.EXPECT().GetSession(mock.Anything, agentTmuxID).
		Return(domain.TmuxSession{}, inspectErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.AttachAgent(context.Background(), AttachAgentRequest{ID: sess.ID})

	if err == nil || !errors.Is(err, inspectErr) {
		t.Fatalf("AttachAgent() error = %v, want wrapped %v", err, inspectErr)
	}
}

func TestSessionService_AttachAgent_RecreateErrorBubblesUp(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	sess := testutil.MakeSession("alpha", uuid.New())
	if err := sess.AssignAgentCommand("opencode"); err != nil {
		t.Fatalf("seed AssignAgentCommand: %v", err)
	}
	agentTmuxID := sess.ID.String() + "-agent"
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, agentTmuxID).
		Return(domain.TmuxSession{}, domain.ErrTmuxSessionNotFound).Once()
	createErr := errors.New("tmux create failed")
	tmux.EXPECT().CreateSession(mock.Anything, agentTmuxID, "/tmp/overseer-home", "opencode").
		Return("", createErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.AttachAgent(context.Background(), AttachAgentRequest{ID: sess.ID})

	if err == nil || !errors.Is(err, createErr) {
		t.Fatalf("AttachAgent() error = %v, want wrapped %v", err, createErr)
	}
}

func TestSessionService_AttachAgent_TmuxAdapterErrorWrapped(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	if err := sess.AssignAgentCommand("opencode"); err != nil {
		t.Fatalf("seed AssignAgentCommand: %v", err)
	}
	agentTmuxID := sess.ID.String() + "-agent"
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, agentTmuxID).
		Return(domain.TmuxSession{ID: agentTmuxID}, nil).Once()
	adapterErr := errors.New("tmux binary missing")
	tmux.EXPECT().AttachCommand(mock.Anything, agentTmuxID).
		Return(nil, adapterErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.AttachAgent(context.Background(), AttachAgentRequest{ID: sess.ID})

	if err == nil || !errors.Is(err, adapterErr) {
		t.Fatalf("AttachAgent() error = %v, want wrapped %v", err, adapterErr)
	}
}

// --- Delete ---

// pinWorktreeRoot isolates worktree paths to a temp directory for the test
// by overriding XDG_DATA_HOME, then returns the resolved worktree root.
func pinWorktreeRoot(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	return paths.NewResolver("").WorktreeRoot()
}

func TestSessionService_Delete_HappyPath_ProjectBackedSession(t *testing.T) {
	pinWorktreeRoot(t)
	overseerID := uuid.New()
	sess := testutil.MakeSessionWithWorktree(
		"alpha",
		overseerID,
		paths.NewResolver("").SessionWorktreePath(uuid.New()),
		"main",
		"overseer/alpha",
	)
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	project := testutil.MakeProject("/repo/overseer", "overseer")
	project.ID = overseerID
	projects.EXPECT().Get(mock.Anything, overseerID).Return(project, nil).Once()
	git.EXPECT().RemoveWorktree(mock.Anything, "/repo/overseer", sess.WorktreePath).Return(nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{ID: sess.ID.String()}, nil).Once()
	tmux.EXPECT().KillSession(mock.Anything, sess.ID.String()).Return(nil).Once()
	repo.EXPECT().Delete(mock.Anything, sess.ID).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
}

func TestSessionService_Delete_HappyPath_ProjectlessSessionSkipsFilesystem(t *testing.T) {
	pinWorktreeRoot(t)
	sess := testutil.MakeSession("orphan", uuid.Nil)
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{ID: sess.ID.String()}, nil).Once()
	tmux.EXPECT().KillSession(mock.Anything, sess.ID.String()).Return(nil).Once()
	repo.EXPECT().Delete(mock.Anything, sess.ID).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
}

func TestSessionService_Delete_NotFound(t *testing.T) {
	missingID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, missingID).
		Return(domain.Session{}, domain.ErrSessionNotFound).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: missingID})

	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("Delete() error = %v, want %v", err, domain.ErrSessionNotFound)
	}
}

func TestSessionService_Delete_WorktreePathOutsideRoot_Refused(t *testing.T) {
	pinWorktreeRoot(t)
	sess := testutil.MakeSessionWithWorktree(
		"malicious",
		uuid.New(),
		"/etc/passwd",
		"main",
		"overseer/malicious",
	)
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if !errors.Is(err, domain.ErrSessionWorktreePathOutsideRoot) {
		t.Fatalf("Delete() error = %v, want %v", err, domain.ErrSessionWorktreePathOutsideRoot)
	}
}

func TestSessionService_Delete_WorktreePathSiblingOfRoot_Refused(t *testing.T) {
	root := pinWorktreeRoot(t)
	sess := testutil.MakeSessionWithWorktree(
		"sneaky",
		uuid.New(),
		root+"-evil/abc",
		"main",
		"overseer/sneaky",
	)
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if !errors.Is(err, domain.ErrSessionWorktreePathOutsideRoot) {
		t.Fatalf("Delete() error = %v, want %v (root=%q)", err, domain.ErrSessionWorktreePathOutsideRoot, root)
	}
}

func TestSessionService_Delete_ProjectMissing_StillDeletesSessionAndTmux(t *testing.T) {
	pinWorktreeRoot(t)
	overseerID := uuid.New()
	sess := testutil.MakeSessionWithWorktree(
		"alpha",
		overseerID,
		paths.NewResolver("").SessionWorktreePath(uuid.New()),
		"main",
		"overseer/alpha",
	)
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	projects.EXPECT().Get(mock.Anything, overseerID).
		Return(domain.Project{}, domain.ErrProjectNotFound).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{ID: sess.ID.String()}, nil).Once()
	tmux.EXPECT().KillSession(mock.Anything, sess.ID.String()).Return(nil).Once()
	repo.EXPECT().Delete(mock.Anything, sess.ID).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("Delete() error = %v, want nil when project missing", err)
	}
}

func TestSessionService_Delete_ProjectLookupErrorBubblesUp(t *testing.T) {
	pinWorktreeRoot(t)
	overseerID := uuid.New()
	sess := testutil.MakeSessionWithWorktree(
		"alpha",
		overseerID,
		paths.NewResolver("").SessionWorktreePath(uuid.New()),
		"main",
		"overseer/alpha",
	)
	lookupErr := errors.New("project store unavailable")
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	projects.EXPECT().Get(mock.Anything, overseerID).Return(domain.Project{}, lookupErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if !errors.Is(err, lookupErr) {
		t.Fatalf("Delete() error = %v, want wrapped %v", err, lookupErr)
	}
}

func TestSessionService_Delete_GitRemoveWorktreeErrorAbortsBeforeTmuxAndDB(t *testing.T) {
	pinWorktreeRoot(t)
	overseerID := uuid.New()
	sess := testutil.MakeSessionWithWorktree(
		"alpha",
		overseerID,
		paths.NewResolver("").SessionWorktreePath(uuid.New()),
		"main",
		"overseer/alpha",
	)
	project := testutil.MakeProject("/repo/overseer", "overseer")
	project.ID = overseerID
	gitErr := errors.New("git worktree busy")
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	projects.EXPECT().Get(mock.Anything, overseerID).Return(project, nil).Once()
	git.EXPECT().RemoveWorktree(mock.Anything, project.Path, sess.WorktreePath).Return(gitErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if !errors.Is(err, gitErr) {
		t.Fatalf("Delete() error = %v, want wrapped %v", err, gitErr)
	}
}

func TestSessionService_Delete_TmuxSessionAlreadyGone_StillDeletes(t *testing.T) {
	pinWorktreeRoot(t)
	sess := testutil.MakeSession("orphan", uuid.Nil)
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{}, domain.ErrTmuxSessionNotFound).Once()
	repo.EXPECT().Delete(mock.Anything, sess.ID).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("Delete() error = %v, want nil when tmux session is already gone", err)
	}
}

func TestSessionService_Delete_TmuxInspectErrorBubblesUp(t *testing.T) {
	pinWorktreeRoot(t)
	sess := testutil.MakeSession("orphan", uuid.Nil)
	inspectErr := errors.New("tmux server unreachable")
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{}, inspectErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if !errors.Is(err, inspectErr) {
		t.Fatalf("Delete() error = %v, want wrapped %v", err, inspectErr)
	}
}

func TestSessionService_Delete_TmuxKillErrorAbortsBeforeDB(t *testing.T) {
	pinWorktreeRoot(t)
	sess := testutil.MakeSession("orphan", uuid.Nil)
	killErr := errors.New("tmux refused kill")
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{ID: sess.ID.String()}, nil).Once()
	tmux.EXPECT().KillSession(mock.Anything, sess.ID.String()).Return(killErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if !errors.Is(err, killErr) {
		t.Fatalf("Delete() error = %v, want wrapped %v", err, killErr)
	}
}

func TestSessionService_Delete_RepoDeleteErrorBubblesUp(t *testing.T) {
	pinWorktreeRoot(t)
	sess := testutil.MakeSession("orphan", uuid.Nil)
	deleteErr := errors.New("storage write failed")
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{ID: sess.ID.String()}, nil).Once()
	tmux.EXPECT().KillSession(mock.Anything, sess.ID.String()).Return(nil).Once()
	repo.EXPECT().Delete(mock.Anything, sess.ID).Return(deleteErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if !errors.Is(err, deleteErr) {
		t.Fatalf("Delete() error = %v, want wrapped %v", err, deleteErr)
	}
}
