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
	launcher, _ := domain.NewLauncher("OpenCode", "opencode", domain.AgentTypeOpenCode)
	editor, _ := domain.NewEditor("True", "true")
	return NewSessionService(repo, projects, tmux, git, paths.NewResolver(""), launcher, editor, logger)
}

func expectProjectLookup(t *testing.T, projects *mocks.MockProjectRepository, projectID uuid.UUID, name string) string {
	t.Helper()
	repoPath := "/repo/" + name
	project := testutil.MakeProject(repoPath, name)
	project.ID = projectID
	projects.EXPECT().Get(mock.Anything, projectID).Return(project, nil).Once()
	return repoPath
}

func worktreeCreateReq(name string, projectID uuid.UUID, baseBranch string) CreateSessionRequest {
	return CreateSessionRequest{
		Name:           name,
		ProjectID:      projectID,
		CreateWorktree: true,
		BaseBranch:     baseBranch,
	}
}

func TestSessionService_Create_WorktreeMode_HappyPath(t *testing.T) {
	overseerID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)

	repoPath := expectProjectLookup(t, projects, overseerID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").Return("tmux-alpha-agent", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), worktreeCreateReq("alpha", overseerID, "main"))

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
		t.Fatalf("Create() Session.HasWorktree() = false, want true for worktree-mode session")
	}
	wantBranch := paths.SessionFeatureBranch(resp.Session.ID)
	if resp.Session.Branch != wantBranch {
		t.Fatalf("Create() Session.Branch = %q, want %q", resp.Session.Branch, wantBranch)
	}
	wantPath := paths.NewResolver("").SessionWorktreePath(resp.Session.ID)
	if resp.Session.WorktreePath != wantPath {
		t.Fatalf("Create() Session.WorktreePath = %q, want %q", resp.Session.WorktreePath, wantPath)
	}
	if savedSession.WorktreePath != wantPath || savedSession.Branch != wantBranch {
		t.Fatalf("SessionRepository.Save session = %#v, want worktree+branch populated", savedSession)
	}
}

func TestSessionService_Create_ProjectMode_SkipsGit_PersistsWithoutWorktree(t *testing.T) {
	overseerID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)

	repoPath := expectProjectLookup(t, projects, overseerID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), repoPath, "").Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), repoPath, "opencode").Return("tmux-alpha-agent", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{
		Name:           "alpha",
		ProjectID:      overseerID,
		CreateWorktree: false,
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.Session.HasWorktree() {
		t.Fatalf("Create() HasWorktree() = true, want false for project-mode session")
	}
	if resp.Session.Branch != "" {
		t.Fatalf("Create() Session.Branch = %q, want empty for project-mode", resp.Session.Branch)
	}
	if resp.Session.WorktreePath != "" {
		t.Fatalf("Create() Session.WorktreePath = %q, want empty for project-mode", resp.Session.WorktreePath)
	}
	if savedSession.HasWorktree() {
		t.Fatalf("saved session HasWorktree() = true, want false")
	}
}

func TestSessionService_Create_EmptyName(t *testing.T) {
	repo, projects, tmux, git := newSessionMocks(t)
	svc := newTestSessionService(repo, projects, tmux, git, testLogger())

	_, err := svc.Create(context.Background(), worktreeCreateReq("", uuid.New(), "main"))

	if !errors.Is(err, domain.ErrSessionEmptyName) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrSessionEmptyName)
	}
}

func TestSessionService_Create_WorktreeMode_EmptyBaseBranch_ResolvesProjectDefault(t *testing.T) {
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, projID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().GetDefaultBranch(mock.Anything, repoPath).Return("trunk", nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "trunk", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").Return("tmux-alpha-agent", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: projID, CreateWorktree: true})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestSessionService_Create_WorktreeMode_DefaultBranchResolveError(t *testing.T) {
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, projID, "overseer")
	gitErr := errors.New("no remote configured")
	git.EXPECT().GetDefaultBranch(mock.Anything, repoPath).Return("", gitErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: projID, CreateWorktree: true})

	if !errors.Is(err, gitErr) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, gitErr)
	}
}

func TestSessionService_Create_EmptyProjectID(t *testing.T) {
	repo, projects, tmux, git := newSessionMocks(t)
	svc := newTestSessionService(repo, projects, tmux, git, testLogger())

	_, err := svc.Create(context.Background(), worktreeCreateReq("alpha", uuid.Nil, "main"))

	if !errors.Is(err, domain.ErrSessionEmptyProjectID) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrSessionEmptyProjectID)
	}
}

func TestSessionService_Create_WorktreeMode_UsesUserProvidedBranch(t *testing.T) {
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, projID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", "my-feature", mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").Return("tmux-alpha-agent", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{
		Name:           "alpha",
		ProjectID:      projID,
		CreateWorktree: true,
		BaseBranch:     "main",
		Branch:         "my-feature",
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.Session.Branch != "my-feature" {
		t.Fatalf("Create() Session.Branch = %q, want %q", resp.Session.Branch, "my-feature")
	}
	if savedSession.Branch != "my-feature" {
		t.Fatalf("SessionRepository.Save Branch = %q, want %q", savedSession.Branch, "my-feature")
	}
}

func TestSessionService_Create_WorktreeMode_BlankBranchGeneratesDefault(t *testing.T) {
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, projID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").Return("tmux-alpha-agent", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{
		Name:           "alpha",
		ProjectID:      projID,
		CreateWorktree: true,
		BaseBranch:     "main",
		Branch:         "   ",
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	want := paths.SessionFeatureBranch(resp.Session.ID)
	if resp.Session.Branch != want {
		t.Fatalf("Create() Session.Branch = %q, want generated default %q", resp.Session.Branch, want)
	}
}

func TestSessionService_Create_DuplicateNameWithinSameProject(t *testing.T) {
	overseerID := uuid.New()
	existing := testutil.MakeSession("alpha", overseerID)
	repo, projects, tmux, git := newSessionMocks(t)
	expectProjectLookup(t, projects, overseerID, "overseer")
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{existing}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), worktreeCreateReq("alpha", overseerID, "main"))

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
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").Return("tmux-alpha-agent", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), worktreeCreateReq("alpha", overseerID, "main"))

	if err != nil {
		t.Fatalf("Create() error = %v, want nil (same name in different project is allowed)", err)
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
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").Return("tmux-gamma-agent", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), worktreeCreateReq("gamma", overseerID, "main"))

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

func TestSessionService_Create_AgentTmuxErrorKillsShellAndPropagates(t *testing.T) {
	tmuxErr := errors.New("tmux out of capacity")
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, projID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").
		Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").
		Return("", tmuxErr).Once()
	tmux.EXPECT().KillSession(mock.Anything, testutil.UUIDString()).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), worktreeCreateReq("alpha", projID, "main"))

	if err == nil || !errors.Is(err, tmuxErr) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, tmuxErr)
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
	_, err := svc.Create(context.Background(), worktreeCreateReq("alpha", projID, "main"))

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
	_, err := svc.Create(context.Background(), worktreeCreateReq("alpha", projID, "main"))

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
	_, err := svc.Create(context.Background(), worktreeCreateReq("alpha", projID, "main"))

	if !errors.Is(err, lookupErr) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, lookupErr)
	}
}

func TestSessionService_Create_WithAgentCommand(t *testing.T) {
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, projID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").
		Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "claude").
		Return("tmux-alpha-agent", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{
		Name:           "alpha",
		ProjectID:      projID,
		CreateWorktree: true,
		BaseBranch:     "main",
		AgentCommand:   "claude",
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
		Name:           "alpha",
		ProjectID:      uuid.New(),
		CreateWorktree: true,
		BaseBranch:     "main",
		AgentCommand:   "   ",
	})

	if !errors.Is(err, domain.ErrSessionEmptyAgentCommand) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrSessionEmptyAgentCommand)
	}
}

func TestSessionService_Create_WithEditorCommand(t *testing.T) {
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, projID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").
		Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").
		Return("tmux-alpha-agent", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{
		Name:           "alpha",
		ProjectID:      projID,
		CreateWorktree: true,
		BaseBranch:     "main",
		AgentCommand:   "opencode",
		EditorCommand:  "cursor",
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.Session.EditorCommand != "cursor" {
		t.Fatalf("Create() Session.EditorCommand = %q, want %q", resp.Session.EditorCommand, "cursor")
	}
	if savedSession.EditorCommand != "cursor" {
		t.Fatalf("SessionRepository.Save session.EditorCommand = %q, want %q", savedSession.EditorCommand, "cursor")
	}
}

func TestCreateSession_CapturesAgentTypeFromLauncher(t *testing.T) {
	projID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, projID, "overseer")
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, repoPath, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").
		Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "claude").
		Return("tmux-alpha-agent", nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{
		Name:           "alpha",
		ProjectID:      projID,
		CreateWorktree: true,
		BaseBranch:     "main",
		AgentCommand:   "claude",
		AgentType:      domain.AgentTypeClaudeCode,
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.Session.AgentType != domain.AgentTypeClaudeCode {
		t.Fatalf("Create() Session.AgentType = %q, want %q", resp.Session.AgentType, domain.AgentTypeClaudeCode)
	}
	if savedSession.AgentType != domain.AgentTypeClaudeCode {
		t.Fatalf("SessionRepository.Save session.AgentType = %q, want %q", savedSession.AgentType, domain.AgentTypeClaudeCode)
	}
}

func TestSessionService_Create_RejectsInvalidEditorCommand(t *testing.T) {
	repo, projects, tmux, git := newSessionMocks(t)
	svc := newTestSessionService(repo, projects, tmux, git, testLogger())

	_, err := svc.Create(context.Background(), CreateSessionRequest{
		Name:           "alpha",
		ProjectID:      uuid.New(),
		CreateWorktree: true,
		BaseBranch:     "main",
		EditorCommand:  "   ",
	})

	if !errors.Is(err, domain.ErrSessionEmptyEditorCommand) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrSessionEmptyEditorCommand)
	}
}

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
}

func TestSessionService_ListBranches_HappyPath(t *testing.T) {
	overseerID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, overseerID, "overseer")

	when := time.Now()
	branches := []domain.BranchInfo{
		{Name: "main", Scope: domain.BranchScopeLocal, CommitterDate: when},
		{Name: "origin/feat/x", Scope: domain.BranchScopeRemote, CommitterDate: when.Add(-time.Hour)},
	}
	git.EXPECT().ListBranches(mock.Anything, repoPath).Return(branches, nil).Once()
	git.EXPECT().GetDefaultBranch(mock.Anything, repoPath).Return("main", nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.ListBranches(context.Background(), ListBranchesRequest{ProjectID: overseerID})

	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}
	if len(resp.Branches) != 2 {
		t.Fatalf("ListBranches() len = %d, want 2", len(resp.Branches))
	}
	if resp.Branches[0].Name != "main" {
		t.Fatalf("ListBranches()[0].Name = %q, want %q", resp.Branches[0].Name, "main")
	}
	if resp.DefaultBranch != "main" {
		t.Fatalf("ListBranches() DefaultBranch = %q, want %q", resp.DefaultBranch, "main")
	}
}

func TestSessionService_ListBranches_DefaultBranchResolveError_NotFatal(t *testing.T) {
	overseerID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, overseerID, "overseer")
	git.EXPECT().ListBranches(mock.Anything, repoPath).Return(nil, nil).Once()
	git.EXPECT().GetDefaultBranch(mock.Anything, repoPath).Return("", errors.New("no default")).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.ListBranches(context.Background(), ListBranchesRequest{ProjectID: overseerID})

	if err != nil {
		t.Fatalf("ListBranches() error = %v, want nil (default-branch error is non-fatal)", err)
	}
	if resp.DefaultBranch != "" {
		t.Fatalf("ListBranches() DefaultBranch = %q, want empty on resolve error", resp.DefaultBranch)
	}
}

func TestSessionService_ListBranches_ProjectLookupErrorBubblesUp(t *testing.T) {
	overseerID := uuid.New()
	lookupErr := errors.New("project store down")
	repo, projects, tmux, git := newSessionMocks(t)
	projects.EXPECT().Get(mock.Anything, overseerID).Return(domain.Project{}, lookupErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.ListBranches(context.Background(), ListBranchesRequest{ProjectID: overseerID})

	if !errors.Is(err, lookupErr) {
		t.Fatalf("ListBranches() error = %v, want wrapped %v", err, lookupErr)
	}
}

func TestSessionService_ListBranches_GitErrorBubblesUp(t *testing.T) {
	overseerID := uuid.New()
	gitErr := errors.New("git refused")
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, overseerID, "overseer")
	git.EXPECT().ListBranches(mock.Anything, repoPath).Return(nil, gitErr).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.ListBranches(context.Background(), ListBranchesRequest{ProjectID: overseerID})

	if !errors.Is(err, gitErr) {
		t.Fatalf("ListBranches() error = %v, want wrapped %v", err, gitErr)
	}
}

func TestSessionService_ProjectCurrentBranch_HappyPath(t *testing.T) {
	overseerID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repoPath := expectProjectLookup(t, projects, overseerID, "overseer")
	git.EXPECT().CurrentBranch(mock.Anything, repoPath).Return("feat/foo", nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.ProjectCurrentBranch(context.Background(), ProjectCurrentBranchRequest{ProjectID: overseerID})

	if err != nil {
		t.Fatalf("ProjectCurrentBranch() error = %v", err)
	}
	if resp.Branch != "feat/foo" {
		t.Fatalf("ProjectCurrentBranch() Branch = %q, want %q", resp.Branch, "feat/foo")
	}
}

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
	assertSessionOrder(t, resp.Sessions, "A", 1)
	assertSessionOrder(t, resp.Sessions, "C", 2)
	assertSessionOrder(t, resp.Sessions, "B", 3)
}

func TestSessionService_Reorder_BoundaryFirst_Up(t *testing.T) {
	projectID := uuid.New()
	a := testutil.MakeSession("A", projectID)
	a.Order = 1
	b := testutil.MakeSession("B", projectID)
	b.Order = 2

	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, a.ID).Return(a, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a, b}, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Reorder(context.Background(), ReorderSessionRequest{ID: a.ID, Direction: -1})

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

func TestSessionService_AttachShell_TmuxSessionMissing_RecreatesAtWorktreePath(t *testing.T) {
	worktreePath := "/abs/worktree/alpha"
	sess := testutil.MakeSessionWithWorktree("alpha", uuid.New(), worktreePath, "overseer/alpha")
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

func TestSessionService_AttachShell_ProjectMode_TmuxMissing_RecreatesAtProjectPath(t *testing.T) {
	overseerID := uuid.New()
	sess := testutil.MakeSession("alpha", overseerID)
	project := testutil.MakeProject("/repo/overseer", "overseer")
	project.ID = overseerID

	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{}, domain.ErrTmuxSessionNotFound).Once()
	projects.EXPECT().Get(mock.Anything, overseerID).Return(project, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, sess.ID.String(), project.Path, "").
		Return(sess.ID.String(), nil).Once()
	wantCmd := exec.Command("tmux", "attach-session", "-t", sess.ID.String())
	tmux.EXPECT().AttachCommand(mock.Anything, sess.ID.String()).Return(wantCmd, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.AttachShell(context.Background(), AttachShellRequest{ID: sess.ID})
	if err != nil {
		t.Fatalf("AttachShell() error = %v, want nil for project-mode recreate", err)
	}
}

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
	sess := testutil.MakeSessionWithWorktree("alpha", uuid.New(), worktreePath, "overseer/alpha")
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

func TestSessionService_AttachAgent_NoSessionCommandAndNoDefaultLauncher_ReturnsSentinel(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()

	editor, _ := domain.NewEditor("VSCode", "code")
	svc := NewSessionService(repo, projects, tmux, git, paths.NewResolver(""), domain.Launcher{}, editor, testLogger())
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

func TestSessionService_PreviewSession_Shell_ReturnsContent(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().CapturePane(mock.Anything, sess.ID.String()).Return("shell content", nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.PreviewSession(context.Background(), PreviewSessionRequest{ID: sess.ID, Kind: PreviewKindShell})

	if err != nil {
		t.Fatalf("PreviewSession() error = %v", err)
	}
	if !resp.SessionReady {
		t.Errorf("PreviewSession() SessionReady = false, want true")
	}
	if resp.Content != "shell content" {
		t.Errorf("PreviewSession() Content = %q, want %q", resp.Content, "shell content")
	}
}

func TestSessionService_PreviewSession_TmuxSessionMissing_ReturnsNotReady(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().CapturePane(mock.Anything, sess.ID.String()+"-agent").
		Return("", domain.ErrTmuxSessionNotFound).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.PreviewSession(context.Background(), PreviewSessionRequest{ID: sess.ID, Kind: PreviewKindAgent})

	if err != nil {
		t.Fatalf("PreviewSession() error = %v, want nil for not-found", err)
	}
	if resp.SessionReady {
		t.Errorf("PreviewSession() SessionReady = true, want false")
	}
}

func pinWorktreeRoot(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	return paths.NewResolver("").WorktreeRoot()
}

func TestSessionService_Delete_HappyPath_WorktreeSession(t *testing.T) {
	pinWorktreeRoot(t)
	overseerID := uuid.New()
	sess := testutil.MakeSessionWithWorktree(
		"alpha",
		overseerID,
		paths.NewResolver("").SessionWorktreePath(uuid.New()),
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

func TestSessionService_Delete_HappyPath_ProjectMode_NoGitCall(t *testing.T) {
	pinWorktreeRoot(t)
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(domain.TmuxSession{ID: sess.ID.String()}, nil).Once()
	tmux.EXPECT().KillSession(mock.Anything, sess.ID.String()).Return(nil).Once()
	repo.EXPECT().Delete(mock.Anything, sess.ID).Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.Delete(context.Background(), DeleteSessionRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("Delete() error = %v, want nil for project-mode session", err)
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

func TestSessionService_Delete_TmuxSessionAlreadyGone_StillDeletes(t *testing.T) {
	pinWorktreeRoot(t)
	sess := testutil.MakeSession("orphan", uuid.New())
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

func TestSessionService_OpenEditor_WorktreeSession_LaunchesAtWorktree(t *testing.T) {
	worktreePath := t.TempDir()
	sess := testutil.MakeSessionWithWorktree("alpha", uuid.New(), worktreePath, "overseer/alpha")
	if err := sess.AssignEditorCommand("true"); err != nil {
		t.Fatalf("seed AssignEditorCommand: %v", err)
	}
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.OpenEditor(context.Background(), OpenEditorRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("OpenEditor() error = %v", err)
	}
	if resp.Command.Dir != worktreePath {
		t.Fatalf("OpenEditor() Command.Dir = %q, want %q", resp.Command.Dir, worktreePath)
	}
}

func TestSessionService_OpenEditor_ProjectMode_LaunchesAtProjectPath(t *testing.T) {
	projectPath := t.TempDir()
	overseerID := uuid.New()
	sess := testutil.MakeSession("alpha", overseerID)
	if err := sess.AssignEditorCommand("true"); err != nil {
		t.Fatalf("seed AssignEditorCommand: %v", err)
	}
	project := testutil.MakeProject(projectPath, "overseer")
	project.ID = overseerID
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	projects.EXPECT().Get(mock.Anything, overseerID).Return(project, nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.OpenEditor(context.Background(), OpenEditorRequest{ID: sess.ID})

	if err != nil {
		t.Fatalf("OpenEditor() error = %v", err)
	}
	if resp.Command.Dir != projectPath {
		t.Fatalf("OpenEditor() Command.Dir = %q, want %q", resp.Command.Dir, projectPath)
	}
}

func TestSessionService_OpenEditor_NoSessionCommandAndNoDefaultEditor_ReturnsSentinel(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()

	launcher, _ := domain.NewLauncher("OpenCode", "opencode", domain.AgentTypeOpenCode)
	svc := NewSessionService(repo, projects, tmux, git, paths.NewResolver(""), launcher, domain.Editor{}, testLogger())
	_, err := svc.OpenEditor(context.Background(), OpenEditorRequest{ID: sess.ID})

	if !errors.Is(err, domain.ErrSessionNoEditorCommandAvailable) {
		t.Fatalf("OpenEditor() error = %v, want ErrSessionNoEditorCommandAvailable", err)
	}
}

func TestSessionService_OpenEditor_SessionNotFound(t *testing.T) {
	sessID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sessID).Return(domain.Session{}, domain.ErrSessionNotFound).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.OpenEditor(context.Background(), OpenEditorRequest{ID: sessID})

	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("OpenEditor() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionService_CycleLabel_FromEmptyAssignsFirst(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()

	var saved domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { saved = s }).
		Return(nil).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	resp, err := svc.CycleLabel(context.Background(), CycleSessionLabelRequest{
		ID:     sess.ID,
		Labels: domain.DefaultLabels,
	})

	if err != nil {
		t.Fatalf("CycleLabel() error = %v", err)
	}
	if resp.Session.Label != "WIP" {
		t.Fatalf("response Label = %q, want %q", resp.Session.Label, "WIP")
	}
	if saved.Label != "WIP" {
		t.Fatalf("saved Label = %q, want %q", saved.Label, "WIP")
	}
}

func TestSessionService_CycleLabel_NotFound(t *testing.T) {
	missingID := uuid.New()
	repo, projects, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, missingID).
		Return(domain.Session{}, domain.ErrSessionNotFound).Once()

	svc := newTestSessionService(repo, projects, tmux, git, testLogger())
	_, err := svc.CycleLabel(context.Background(), CycleSessionLabelRequest{
		ID:     missingID,
		Labels: domain.DefaultLabels,
	})

	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("CycleLabel() error = %v, want %v", err, domain.ErrSessionNotFound)
	}
}
