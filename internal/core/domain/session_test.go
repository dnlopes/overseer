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

func TestNewSession_RejectsZeroProjectID(t *testing.T) {
	_, err := NewSession("orphan", uuid.Nil)
	if !errors.Is(err, ErrSessionEmptyProjectID) {
		t.Fatalf("NewSession() error = %v, want %v", err, ErrSessionEmptyProjectID)
	}
}

func TestNewSession_DefaultsToFeatureKind(t *testing.T) {
	s, err := NewSession("alpha", uuid.New())
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if s.Kind != SessionKindFeature {
		t.Fatalf("NewSession() Kind = %q, want %q", s.Kind, SessionKindFeature)
	}
	if s.IsCheckout() {
		t.Fatal("NewSession() IsCheckout() = true, want false")
	}
}

func TestNewCheckoutSession_SetsCheckoutKind(t *testing.T) {
	s, err := NewCheckoutSession("main", uuid.New())
	if err != nil {
		t.Fatalf("NewCheckoutSession() error = %v", err)
	}
	if s.Kind != SessionKindCheckout {
		t.Fatalf("NewCheckoutSession() Kind = %q, want %q", s.Kind, SessionKindCheckout)
	}
	if !s.IsCheckout() {
		t.Fatal("NewCheckoutSession() IsCheckout() = false, want true")
	}
}

func TestNewCheckoutSession_AppliesSameValidationAsNewSession(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		projectID uuid.UUID
		wantErr   error
	}{
		{name: "empty name", input: "", projectID: uuid.New(), wantErr: ErrSessionEmptyName},
		{name: "blank name", input: "   ", projectID: uuid.New(), wantErr: ErrSessionEmptyName},
		{name: "name too long", input: strings.Repeat("a", 101), projectID: uuid.New(), wantErr: ErrSessionNameTooLong},
		{name: "nil project id", input: "main", projectID: uuid.Nil, wantErr: ErrSessionEmptyProjectID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCheckoutSession(tt.input, tt.projectID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewCheckoutSession() error = %v, want %v", err, tt.wantErr)
			}
		})
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

func TestAssignAgentCommand_StoresAndUpdatesTimestamp(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	originalUpdated := s.UpdatedAt
	time.Sleep(time.Millisecond)

	if err := s.AssignAgentCommand("opencode"); err != nil {
		t.Fatalf("AssignAgentCommand() error = %v", err)
	}
	if s.AgentCommand != "opencode" {
		t.Fatalf("AgentCommand = %q, want %q", s.AgentCommand, "opencode")
	}
	if !s.UpdatedAt.After(originalUpdated) {
		t.Fatalf("UpdatedAt = %v, want after %v", s.UpdatedAt, originalUpdated)
	}
}

func TestAssignAgentCommand_TrimsCommand(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	if err := s.AssignAgentCommand("  opencode --config foo  "); err != nil {
		t.Fatalf("AssignAgentCommand() error = %v", err)
	}
	if s.AgentCommand != "opencode --config foo" {
		t.Fatalf("AgentCommand = %q, want %q", s.AgentCommand, "opencode --config foo")
	}
}

func TestAssignAgentCommand_RejectsEmpty(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{name: "empty", cmd: ""},
		{name: "blank", cmd: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewSession("alpha", uuid.New())
			err := s.AssignAgentCommand(tt.cmd)
			if !errors.Is(err, ErrSessionEmptyAgentCommand) {
				t.Fatalf("AssignAgentCommand(%q) error = %v, want %v", tt.cmd, err, ErrSessionEmptyAgentCommand)
			}
			if s.AgentCommand != "" {
				t.Fatalf("AgentCommand = %q, want empty after rejected assignment", s.AgentCommand)
			}
		})
	}
}

func TestAssignEditorCommand_StoresAndUpdatesTimestamp(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	originalUpdated := s.UpdatedAt
	time.Sleep(time.Millisecond)

	if err := s.AssignEditorCommand("code"); err != nil {
		t.Fatalf("AssignEditorCommand() error = %v", err)
	}
	if s.EditorCommand != "code" {
		t.Fatalf("EditorCommand = %q, want %q", s.EditorCommand, "code")
	}
	if !s.UpdatedAt.After(originalUpdated) {
		t.Fatalf("UpdatedAt = %v, want after %v", s.UpdatedAt, originalUpdated)
	}
}

func TestAssignEditorCommand_TrimsCommand(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	if err := s.AssignEditorCommand("  code --wait  "); err != nil {
		t.Fatalf("AssignEditorCommand() error = %v", err)
	}
	if s.EditorCommand != "code --wait" {
		t.Fatalf("EditorCommand = %q, want %q", s.EditorCommand, "code --wait")
	}
}

func TestAssignEditorCommand_RejectsEmpty(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{name: "empty", cmd: ""},
		{name: "blank", cmd: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewSession("alpha", uuid.New())
			err := s.AssignEditorCommand(tt.cmd)
			if !errors.Is(err, ErrSessionEmptyEditorCommand) {
				t.Fatalf("AssignEditorCommand(%q) error = %v, want %v", tt.cmd, err, ErrSessionEmptyEditorCommand)
			}
			if s.EditorCommand != "" {
				t.Fatalf("EditorCommand = %q, want empty after rejected assignment", s.EditorCommand)
			}
		})
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

func TestWorktreeIsInsideRoot(t *testing.T) {
	tests := []struct {
		name         string
		worktreePath string
		root         string
		want         bool
	}{
		{name: "no worktree is trivially inside", worktreePath: "", root: "/data/worktrees", want: true},
		{name: "direct child of root", worktreePath: "/data/worktrees/abc-123", root: "/data/worktrees", want: true},
		{name: "deeper descendant of root", worktreePath: "/data/worktrees/abc/nested/file", root: "/data/worktrees", want: true},
		{name: "root with trailing slash still matches child", worktreePath: "/data/worktrees/abc", root: "/data/worktrees/", want: true},
		{name: "exact root path is rejected", worktreePath: "/data/worktrees", root: "/data/worktrees", want: false},
		{name: "sibling sharing textual prefix is rejected", worktreePath: "/data/worktrees-evil/abc", root: "/data/worktrees", want: false},
		{name: "parent of root is rejected", worktreePath: "/data", root: "/data/worktrees", want: false},
		{name: "unrelated absolute path is rejected", worktreePath: "/etc/passwd", root: "/data/worktrees", want: false},
		{name: "home directory is rejected", worktreePath: "/home/user", root: "/data/worktrees", want: false},
		{name: "empty root rejects any non-empty path", worktreePath: "/data/worktrees/abc", root: "", want: false},
		{name: "relative root rejects any non-empty path", worktreePath: "/data/worktrees/abc", root: "data/worktrees", want: false},
		{name: "root with surrounding whitespace still validates", worktreePath: "/data/worktrees/abc", root: "  /data/worktrees  ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewSession("alpha", uuid.New())
			if tt.worktreePath != "" {
				s.WorktreePath = tt.worktreePath
			}
			if got := s.WorktreeIsInsideRoot(tt.root); got != tt.want {
				t.Fatalf("WorktreeIsInsideRoot(%q) with WorktreePath=%q = %v, want %v",
					tt.root, tt.worktreePath, got, tt.want)
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

func TestAssignLabel_StoresAndUpdatesTimestamp(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	originalUpdated := s.UpdatedAt
	time.Sleep(time.Millisecond)

	if err := s.AssignLabel("WIP"); err != nil {
		t.Fatalf("AssignLabel() error = %v", err)
	}
	if s.Label != "WIP" {
		t.Fatalf("Label = %q, want %q", s.Label, "WIP")
	}
	if !s.UpdatedAt.After(originalUpdated) {
		t.Fatalf("UpdatedAt = %v, want after %v", s.UpdatedAt, originalUpdated)
	}
}

func TestAssignLabel_TrimsCode(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())

	if err := s.AssignLabel("  reviewing  "); err != nil {
		t.Fatalf("AssignLabel() error = %v", err)
	}
	if s.Label != "reviewing" {
		t.Fatalf("Label = %q, want trimmed %q", s.Label, "reviewing")
	}
}

func TestAssignLabel_EmptyCodeClears(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	_ = s.AssignLabel("done")
	if s.Label != "done" {
		t.Fatalf("precondition: Label = %q, want %q", s.Label, "done")
	}

	if err := s.AssignLabel(""); err != nil {
		t.Fatalf("AssignLabel(\"\") error = %v, want nil (clear)", err)
	}
	if s.Label != "" {
		t.Fatalf("Label = %q, want empty after clear", s.Label)
	}
}

func TestAssignLabel_BlankCodeClears(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	_ = s.AssignLabel("done")

	if err := s.AssignLabel("   "); err != nil {
		t.Fatalf("AssignLabel(\"   \") error = %v, want nil (clear)", err)
	}
	if s.Label != "" {
		t.Fatalf("Label = %q, want empty (blank trims to empty)", s.Label)
	}
}

func TestAssignLabel_RejectsTooLong(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	tooLong := strings.Repeat("a", 51)

	err := s.AssignLabel(tooLong)

	if !errors.Is(err, ErrSessionLabelTooLong) {
		t.Fatalf("AssignLabel(too-long) error = %v, want %v", err, ErrSessionLabelTooLong)
	}
}

func TestAssignLabel_DoesNotBumpTimestampOnError(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	originalUpdated := s.UpdatedAt
	time.Sleep(time.Millisecond)

	_ = s.AssignLabel(strings.Repeat("a", 51))

	if !s.UpdatedAt.Equal(originalUpdated) {
		t.Fatalf("UpdatedAt = %v, want unchanged %v after rejected assignment", s.UpdatedAt, originalUpdated)
	}
}
