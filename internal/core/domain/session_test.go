package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSession_CreatesSession(t *testing.T) {
	before := time.Now()

	s, err := NewSession("alpha", "overseer")

	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if s.ID == uuid.Nil {
		t.Fatal("NewSession() ID is nil")
	}
	if s.Name != "alpha" {
		t.Fatalf("NewSession() Name = %q, want %q", s.Name, "alpha")
	}
	if s.ProjectName != "overseer" {
		t.Fatalf("NewSession() ProjectName = %q, want %q", s.ProjectName, "overseer")
	}
	if s.Order != 0 {
		t.Fatalf("NewSession() Order = %d, want 0", s.Order)
	}
	if s.CreatedAt.Before(before) {
		t.Fatalf("NewSession() CreatedAt = %v, before creation start %v", s.CreatedAt, before)
	}
	if s.UpdatedAt.Before(before) {
		t.Fatalf("NewSession() UpdatedAt = %v, before creation start %v", s.UpdatedAt, before)
	}
	if !s.CreatedAt.Equal(s.UpdatedAt) {
		t.Fatalf("NewSession() CreatedAt = %v, UpdatedAt = %v, want equal", s.CreatedAt, s.UpdatedAt)
	}
}

func TestNewSession_TrimsNameAndProject(t *testing.T) {
	s, err := NewSession("  alpha  ", "  overseer  ")

	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if s.Name != "alpha" {
		t.Fatalf("NewSession() Name = %q, want %q", s.Name, "alpha")
	}
	if s.ProjectName != "overseer" {
		t.Fatalf("NewSession() ProjectName = %q, want %q", s.ProjectName, "overseer")
	}
}

func TestNewSession_Validation(t *testing.T) {
	long := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tests := []struct {
		name    string
		session string
		project string
		wantErr error
	}{
		{name: "empty name", session: "", project: "overseer", wantErr: ErrSessionEmptyName},
		{name: "blank name", session: "   ", project: "overseer", wantErr: ErrSessionEmptyName},
		{name: "name too long", session: long, project: "overseer", wantErr: ErrSessionNameTooLong},
		{name: "empty project", session: "alpha", project: "", wantErr: ErrSessionEmptyProject},
		{name: "blank project", session: "alpha", project: "   ", wantErr: ErrSessionEmptyProject},
		{name: "project too long", session: "alpha", project: long, wantErr: ErrSessionProjectTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSession(tt.session, tt.project)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewSession() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewSession_AcceptsOneHundredCharacterFields(t *testing.T) {
	exactly100 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	s, err := NewSession(exactly100, exactly100)

	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if s.Name != exactly100 {
		t.Fatalf("NewSession() Name length = %d, want 100", len(s.Name))
	}
	if s.ProjectName != exactly100 {
		t.Fatalf("NewSession() ProjectName length = %d, want 100", len(s.ProjectName))
	}
}

func TestRename_UpdatesNameAndUpdatedAt(t *testing.T) {
	s, err := NewSession("alpha", "overseer")
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	originalCreatedAt := s.CreatedAt
	originalUpdatedAt := s.UpdatedAt
	time.Sleep(time.Nanosecond)

	err = s.Rename("beta")

	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if s.Name != "beta" {
		t.Fatalf("Rename() Name = %q, want %q", s.Name, "beta")
	}
	if !s.CreatedAt.Equal(originalCreatedAt) {
		t.Fatalf("Rename() CreatedAt = %v, want unchanged %v", s.CreatedAt, originalCreatedAt)
	}
	if !s.UpdatedAt.After(originalUpdatedAt) {
		t.Fatalf("Rename() UpdatedAt = %v, want after %v", s.UpdatedAt, originalUpdatedAt)
	}
}

func TestRename_TrimsName(t *testing.T) {
	s, err := NewSession("alpha", "overseer")
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	err = s.Rename("  beta  ")

	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if s.Name != "beta" {
		t.Fatalf("Rename() Name = %q, want %q", s.Name, "beta")
	}
}

func TestRename_Validation(t *testing.T) {
	long := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tests := []struct {
		name    string
		newName string
		wantErr error
	}{
		{name: "empty name", newName: "", wantErr: ErrSessionEmptyName},
		{name: "blank name", newName: "   ", wantErr: ErrSessionEmptyName},
		{name: "name too long", newName: long, wantErr: ErrSessionNameTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewSession("alpha", "overseer")
			if err != nil {
				t.Fatalf("NewSession() error = %v", err)
			}
			originalName := s.Name
			originalUpdatedAt := s.UpdatedAt

			err = s.Rename(tt.newName)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Rename() error = %v, want %v", err, tt.wantErr)
			}
			if s.Name != originalName {
				t.Fatalf("Rename() changed Name to %q, want unchanged %q", s.Name, originalName)
			}
			if !s.UpdatedAt.Equal(originalUpdatedAt) {
				t.Fatalf("Rename() changed UpdatedAt to %v, want unchanged %v", s.UpdatedAt, originalUpdatedAt)
			}
		})
	}
}

func TestSession_String(t *testing.T) {
	s, err := NewSession("alpha", "overseer")
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	got := s.String()

	if got != "[overseer] alpha" {
		t.Fatalf("String() = %q, want %q", got, "[overseer] alpha")
	}
	if got == s.ID.String() || got == "[overseer] alpha "+s.ID.String() {
		t.Fatal("String() includes UUID")
	}
}

func TestSessionSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "empty name", err: ErrSessionEmptyName, want: "session name cannot be empty"},
		{name: "name too long", err: ErrSessionNameTooLong, want: "session name exceeds 100 characters"},
		{name: "empty project", err: ErrSessionEmptyProject, want: "project name cannot be empty"},
		{name: "project too long", err: ErrSessionProjectTooLong, want: "project name exceeds 100 characters"},
		{name: "not found", err: ErrSessionNotFound, want: "session not found"},
		{name: "already exists", err: ErrSessionAlreadyExists, want: "session already exists"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("error is nil")
			}
			if tt.err.Error() != tt.want {
				t.Fatalf("error message = %q, want %q", tt.err.Error(), tt.want)
			}
		})
	}
}

func TestSessionPortInterfaces(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	s, err := NewSession("alpha", "overseer")
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	var repo SessionRepository = fakeSessionRepository{}
	if err := repo.Save(ctx, s); err != nil {
		t.Fatalf("SessionRepository.Save() error = %v", err)
	}
	if _, err := repo.Get(ctx, id); err != nil {
		t.Fatalf("SessionRepository.Get() error = %v", err)
	}
	if _, err := repo.List(ctx); err != nil {
		t.Fatalf("SessionRepository.List() error = %v", err)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("SessionRepository.Delete() error = %v", err)
	}

	var tmux TmuxAdapter = fakeTmuxAdapter{}
	if _, err := tmux.CreateSession(ctx, "alpha"); err != nil {
		t.Fatalf("TmuxAdapter.CreateSession() error = %v", err)
	}
	if err := tmux.KillSession(ctx, "tmux-alpha"); err != nil {
		t.Fatalf("TmuxAdapter.KillSession() error = %v", err)
	}

	var git GitAdapter = fakeGitAdapter{}
	if err := git.CreateWorktree(ctx, "main", "/tmp/alpha"); err != nil {
		t.Fatalf("GitAdapter.CreateWorktree() error = %v", err)
	}
	if err := git.RemoveWorktree(ctx, "/tmp/alpha"); err != nil {
		t.Fatalf("GitAdapter.RemoveWorktree() error = %v", err)
	}

	var launcher AgentLauncher = fakeAgentLauncher{}
	if _, err := launcher.Launch(ctx, "claude", "/tmp/alpha"); err != nil {
		t.Fatalf("AgentLauncher.Launch() error = %v", err)
	}
}

type fakeSessionRepository struct{}

func (fakeSessionRepository) Save(context.Context, Session) error { return nil }
func (fakeSessionRepository) Get(context.Context, uuid.UUID) (Session, error) {
	return NewSession("alpha", "overseer")
}
func (fakeSessionRepository) List(context.Context) ([]Session, error) { return []Session{}, nil }
func (fakeSessionRepository) Delete(context.Context, uuid.UUID) error { return nil }

type fakeTmuxAdapter struct{}

func (fakeTmuxAdapter) CreateSession(context.Context, string) (string, error) {
	return "tmux-alpha", nil
}
func (fakeTmuxAdapter) KillSession(context.Context, string) error { return nil }

type fakeGitAdapter struct{}

func (fakeGitAdapter) CreateWorktree(context.Context, string, string) error { return nil }
func (fakeGitAdapter) RemoveWorktree(context.Context, string) error         { return nil }

type fakeAgentLauncher struct{}

func (fakeAgentLauncher) Launch(context.Context, string, string) (int, error) { return 1, nil }
