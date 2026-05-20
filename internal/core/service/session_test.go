package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/shared/errs"
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

func newSessionMocks(t *testing.T) (*mocks.MockSessionRepository, *mocks.MockTmuxAdapter, *mocks.MockGitAdapter) {
	t.Helper()
	return mocks.NewMockSessionRepository(t),
		mocks.NewMockTmuxAdapter(t),
		mocks.NewMockGitAdapter(t)
}

// --- Create ---

func TestSessionService_Create_HappyPath(t *testing.T) {
	overseerID := uuid.New()
	repo, tmux, git := newSessionMocks(t)

	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, "alpha").Return("tmux-alpha", nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, "main", "alpha").Return(nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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
	if savedSession.Name != "alpha" || savedSession.ProjectID != overseerID || savedSession.Order != 1 {
		t.Fatalf("SessionRepository.Save session = %#v, want alpha/<overseerID>/order 1", savedSession)
	}
}

func TestSessionService_Create_WithoutProjectAllowed(t *testing.T) {
	repo, tmux, git := newSessionMocks(t)

	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, "orphan").Return("tmux-orphan", nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, "main", "orphan").Return(nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
	resp, err := svc.Create(context.Background(), CreateSessionRequest{Name: "orphan", ProjectID: uuid.Nil})

	if err != nil {
		t.Fatalf("Create() error = %v, want nil for project-less session", err)
	}
	if resp.Session.ProjectID != uuid.Nil {
		t.Fatalf("Create() Session.ProjectID = %v, want uuid.Nil", resp.Session.ProjectID)
	}
}

func TestSessionService_Create_EmptyName(t *testing.T) {
	repo, tmux, git := newSessionMocks(t)
	svc := NewSessionService(repo, tmux, git, testLogger())

	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "", ProjectID: uuid.New()})

	if !errors.Is(err, domain.ErrSessionEmptyName) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrSessionEmptyName)
	}
}

func TestSessionService_Create_DuplicateNameWithinSameProject(t *testing.T) {
	overseerID := uuid.New()
	existing := testutil.MakeSession("alpha", overseerID)
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{existing}, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: overseerID})

	if !errors.Is(err, domain.ErrSessionAlreadyExists) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrSessionAlreadyExists)
	}
}

func TestSessionService_Create_DuplicateNameAcrossProjectsAllowed(t *testing.T) {
	overseerID := uuid.New()
	otherID := uuid.New()
	existing := testutil.MakeSession("alpha", otherID)
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{existing}, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, "alpha").Return("tmux-alpha", nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, "main", "alpha").Return(nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: overseerID})

	if err != nil {
		t.Fatalf("Create() error = %v, want nil (same name in different project is allowed)", err)
	}
}

func TestSessionService_Create_DuplicateNameAmongUnassignedSessions(t *testing.T) {
	existing := testutil.MakeSession("solo", uuid.Nil)
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{existing}, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).
		Return([]domain.Session{first, second, otherProject}, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, "gamma").Return("tmux-gamma", nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, "main", "gamma").Return(nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, "alpha").Return("", tmuxErr).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
	_, err := svc.Create(context.Background(), CreateSessionRequest{Name: "alpha", ProjectID: uuid.New()})

	if !errors.Is(err, tmuxErr) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, tmuxErr)
	}
}

func TestSessionService_Create_FirstSessionOrder(t *testing.T) {
	overseerID := uuid.New()
	otherID := uuid.New()
	otherProject := testutil.MakeSession("alpha", otherID)
	otherProject.Order = 4
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{otherProject}, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, "alpha").Return("tmux-alpha", nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, "main", "alpha").Return(nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, original.ID).Return(original, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{original}, nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, original.ID).Return(original, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{original}, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
	_, err := svc.Rename(context.Background(), RenameSessionRequest{ID: original.ID, NewName: ""})

	if !errors.Is(err, domain.ErrSessionEmptyName) {
		t.Fatalf("Rename() error = %v, want %v", err, domain.ErrSessionEmptyName)
	}
}

func TestSessionService_Rename_NotFound(t *testing.T) {
	repo, tmux, git := newSessionMocks(t)
	missingID := uuid.New()
	repo.EXPECT().Get(mock.Anything, missingID).
		Return(domain.Session{}, domain.ErrSessionNotFound).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
	_, err := svc.Rename(context.Background(), RenameSessionRequest{ID: missingID, NewName: "beta"})

	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("Rename() error = %v, want %v", err, domain.ErrSessionNotFound)
	}
}

func TestSessionService_Rename_DuplicateNameInSameProject(t *testing.T) {
	overseerID := uuid.New()
	original := testutil.MakeSession("alpha", overseerID)
	conflicting := testutil.MakeSession("beta", overseerID)
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, original.ID).Return(original, nil).Once()
	repo.EXPECT().List(mock.Anything).
		Return([]domain.Session{original, conflicting}, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
	_, err := svc.Rename(context.Background(), RenameSessionRequest{ID: original.ID, NewName: "beta"})

	if !errors.Is(err, domain.ErrSessionAlreadyExists) {
		t.Fatalf("Rename() error = %v, want %v", err, domain.ErrSessionAlreadyExists)
	}
}

func TestSessionService_Rename_UpdatedAtChanges(t *testing.T) {
	original := testutil.MakeSession("alpha", uuid.New())
	original.UpdatedAt = time.Now().Add(-time.Minute)
	beforeRename := original.UpdatedAt

	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, original.ID).Return(original, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{original}, nil).Once()

	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{s1, s2, s3}, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).
		Return([]domain.Session{highSession, lowSession}, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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
	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{s1, s2, s3}, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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

	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, b.ID).Return(b, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a, b, c}, nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Twice()

	svc := NewSessionService(repo, tmux, git, testLogger())
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

	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, b.ID).Return(b, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a, b, c}, nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Twice()

	svc := NewSessionService(repo, tmux, git, testLogger())
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

	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, a.ID).Return(a, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a, b, c}, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
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

	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, c.ID).Return(c, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a, b, c}, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
	_, err := svc.Reorder(context.Background(), ReorderSessionRequest{ID: c.ID, Direction: 1})

	if !errors.Is(err, errs.ErrNoOp) {
		t.Fatalf("Reorder() error = %v, want %v", err, errs.ErrNoOp)
	}
}

func TestSessionService_Reorder_SingleSession(t *testing.T) {
	a := testutil.MakeSession("A", uuid.New())
	a.Order = 1

	repo, tmux, git := newSessionMocks(t)
	repo.EXPECT().Get(mock.Anything, a.ID).Return(a, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{a}, nil).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
	_, err := svc.Reorder(context.Background(), ReorderSessionRequest{ID: a.ID, Direction: 1})

	if !errors.Is(err, errs.ErrNoOp) {
		t.Fatalf("Reorder() error = %v, want %v", err, errs.ErrNoOp)
	}
}

func TestSessionService_Reorder_NotFound(t *testing.T) {
	repo, tmux, git := newSessionMocks(t)
	missingID := uuid.New()
	repo.EXPECT().Get(mock.Anything, missingID).
		Return(domain.Session{}, domain.ErrSessionNotFound).Once()

	svc := NewSessionService(repo, tmux, git, testLogger())
	_, err := svc.Reorder(context.Background(), ReorderSessionRequest{ID: missingID, Direction: 1})

	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("Reorder() error = %v, want %v", err, domain.ErrSessionNotFound)
	}
}
