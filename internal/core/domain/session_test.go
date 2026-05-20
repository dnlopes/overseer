package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSession_CreatesSession(t *testing.T) {
	before := time.Now()
	projectID := uuid.New()

	s, err := NewSession("alpha", projectID)

	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if s.ID == uuid.Nil {
		t.Fatal("NewSession() ID is nil")
	}
	if s.Name != "alpha" {
		t.Fatalf("NewSession() Name = %q, want %q", s.Name, "alpha")
	}
	if s.ProjectID != projectID {
		t.Fatalf("NewSession() ProjectID = %v, want %v", s.ProjectID, projectID)
	}
	if s.Order != 0 {
		t.Fatalf("NewSession() Order = %d, want 0", s.Order)
	}
	if s.HasWorktree() {
		t.Fatalf("NewSession() HasWorktree() = true, want false (no worktree assigned)")
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

func TestNewSession_TrimsName(t *testing.T) {
	s, err := NewSession("  alpha  ", uuid.New())

	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if s.Name != "alpha" {
		t.Fatalf("NewSession() Name = %q, want %q", s.Name, "alpha")
	}
}

func TestNewSession_AcceptsZeroProjectIDAsUnassigned(t *testing.T) {
	s, err := NewSession("orphan", uuid.Nil)
	if err != nil {
		t.Fatalf("NewSession() error = %v, want nil for unassigned project", err)
	}
	if s.ProjectID != uuid.Nil {
		t.Fatalf("NewSession() ProjectID = %v, want uuid.Nil", s.ProjectID)
	}
}

func TestNewSession_Validation(t *testing.T) {
	long := strings.Repeat("a", 101)
	tests := []struct {
		name    string
		session string
		wantErr error
	}{
		{name: "empty name", session: "", wantErr: ErrSessionEmptyName},
		{name: "blank name", session: "   ", wantErr: ErrSessionEmptyName},
		{name: "name too long", session: long, wantErr: ErrSessionNameTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSession(tt.session, uuid.New())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewSession() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewSession_AcceptsExactlyOneHundredCharacterName(t *testing.T) {
	exactly100 := strings.Repeat("a", 100)
	s, err := NewSession(exactly100, uuid.New())
	if err != nil {
		t.Fatalf("NewSession() error = %v, want nil for 100-char name", err)
	}
	if s.Name != exactly100 {
		t.Fatalf("NewSession() Name length = %d, want 100", len(s.Name))
	}
}

func TestAssignWorktree_PopulatesEnsemble(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	originalUpdated := s.UpdatedAt
	time.Sleep(time.Millisecond)

	if err := s.AssignWorktree("/abs/worktree", "main", "overseer/alpha"); err != nil {
		t.Fatalf("AssignWorktree() error = %v", err)
	}
	if s.WorktreePath != "/abs/worktree" {
		t.Fatalf("WorktreePath = %q, want %q", s.WorktreePath, "/abs/worktree")
	}
	if s.BaseBranch != "main" {
		t.Fatalf("BaseBranch = %q, want %q", s.BaseBranch, "main")
	}
	if s.FeatureBranch != "overseer/alpha" {
		t.Fatalf("FeatureBranch = %q, want %q", s.FeatureBranch, "overseer/alpha")
	}
	if !s.HasWorktree() {
		t.Fatalf("HasWorktree() = false, want true")
	}
	if !s.UpdatedAt.After(originalUpdated) {
		t.Fatalf("UpdatedAt = %v, want after %v", s.UpdatedAt, originalUpdated)
	}
}

func TestAssignWorktree_TrimsFields(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	if err := s.AssignWorktree("  /abs/worktree  ", "  main  ", "  overseer/alpha  "); err != nil {
		t.Fatalf("AssignWorktree() error = %v", err)
	}
	if s.WorktreePath != "/abs/worktree" || s.BaseBranch != "main" || s.FeatureBranch != "overseer/alpha" {
		t.Fatalf("AssignWorktree did not trim fields: %+v", s)
	}
}

func TestAssignWorktree_Validation(t *testing.T) {
	tests := []struct {
		name          string
		worktreePath  string
		baseBranch    string
		featureBranch string
		wantErr       error
	}{
		{name: "all empty", worktreePath: "", baseBranch: "", featureBranch: "", wantErr: ErrSessionWorktreeFieldsMismatch},
		{name: "path only", worktreePath: "/abs/worktree", baseBranch: "", featureBranch: "", wantErr: ErrSessionWorktreeFieldsMismatch},
		{name: "base branch only", worktreePath: "", baseBranch: "main", featureBranch: "", wantErr: ErrSessionWorktreeFieldsMismatch},
		{name: "feature branch only", worktreePath: "", baseBranch: "", featureBranch: "overseer/alpha", wantErr: ErrSessionWorktreeFieldsMismatch},
		{name: "missing feature", worktreePath: "/abs/worktree", baseBranch: "main", featureBranch: "", wantErr: ErrSessionWorktreeFieldsMismatch},
		{name: "missing base", worktreePath: "/abs/worktree", baseBranch: "", featureBranch: "overseer/alpha", wantErr: ErrSessionWorktreeFieldsMismatch},
		{name: "relative path", worktreePath: "relative/worktree", baseBranch: "main", featureBranch: "overseer/alpha", wantErr: ErrSessionWorktreePathNotAbsolute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewSession("alpha", uuid.New())
			err := s.AssignWorktree(tt.worktreePath, tt.baseBranch, tt.featureBranch)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("AssignWorktree() error = %v, want %v", err, tt.wantErr)
			}
			if s.HasWorktree() {
				t.Fatalf("AssignWorktree failed but session still HasWorktree(): %+v", s)
			}
		})
	}
}

func TestRename_UpdatesNameAndUpdatedAt(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	originalUpdated := s.UpdatedAt
	time.Sleep(time.Millisecond)

	if err := s.Rename("beta"); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	if s.Name != "beta" {
		t.Fatalf("Rename() Name = %q, want %q", s.Name, "beta")
	}
	if !s.UpdatedAt.After(originalUpdated) {
		t.Fatalf("Rename() UpdatedAt = %v, want after %v", s.UpdatedAt, originalUpdated)
	}
}

func TestRename_TrimsAndValidates(t *testing.T) {
	long := strings.Repeat("a", 101)
	tests := []struct {
		name    string
		newName string
		wantErr error
	}{
		{name: "empty", newName: "", wantErr: ErrSessionEmptyName},
		{name: "blank", newName: "   ", wantErr: ErrSessionEmptyName},
		{name: "too long", newName: long, wantErr: ErrSessionNameTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewSession("alpha", uuid.New())
			err := s.Rename(tt.newName)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Rename(%q) error = %v, want %v", tt.newName, err, tt.wantErr)
			}
		})
	}

	t.Run("trims valid name", func(t *testing.T) {
		s, _ := NewSession("alpha", uuid.New())
		if err := s.Rename("  beta  "); err != nil {
			t.Fatalf("Rename() error = %v", err)
		}
		if s.Name != "beta" {
			t.Fatalf("Rename() Name = %q, want trimmed %q", s.Name, "beta")
		}
	})
}
