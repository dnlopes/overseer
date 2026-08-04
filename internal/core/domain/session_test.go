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

func TestSession_HasWorktree_FalseUntilAssigned(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	if s.HasWorktree() {
		t.Fatalf("HasWorktree() = true on fresh session, want false")
	}
	if err := s.AssignWorktree("/abs/wt", "main"); err != nil {
		t.Fatalf("AssignWorktree() error = %v", err)
	}
	if !s.HasWorktree() {
		t.Fatalf("HasWorktree() = false after AssignWorktree, want true")
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

func TestAssignWorktree_PopulatesPathAndBranch(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	originalUpdated := s.UpdatedAt
	time.Sleep(time.Millisecond)

	if err := s.AssignWorktree("/abs/worktree", "overseer/alpha"); err != nil {
		t.Fatalf("AssignWorktree() error = %v", err)
	}
	if s.WorktreePath != "/abs/worktree" {
		t.Fatalf("WorktreePath = %q, want %q", s.WorktreePath, "/abs/worktree")
	}
	if s.Branch != "overseer/alpha" {
		t.Fatalf("Branch = %q, want %q", s.Branch, "overseer/alpha")
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
	if err := s.AssignWorktree("  /abs/worktree  ", "  overseer/alpha  "); err != nil {
		t.Fatalf("AssignWorktree() error = %v", err)
	}
	if s.WorktreePath != "/abs/worktree" || s.Branch != "overseer/alpha" {
		t.Fatalf("AssignWorktree did not trim fields: %+v", s)
	}
}

func TestAssignWorktree_Validation(t *testing.T) {
	tests := []struct {
		name         string
		worktreePath string
		branch       string
		wantErr      error
	}{
		{name: "both empty", worktreePath: "", branch: "", wantErr: ErrSessionWorktreeFieldsMismatch},
		{name: "path only", worktreePath: "/abs/worktree", branch: "", wantErr: ErrSessionWorktreeFieldsMismatch},
		{name: "branch only", worktreePath: "", branch: "overseer/alpha", wantErr: ErrSessionWorktreeFieldsMismatch},
		{name: "relative path", worktreePath: "relative/worktree", branch: "overseer/alpha", wantErr: ErrSessionWorktreePathNotAbsolute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewSession("alpha", uuid.New())
			err := s.AssignWorktree(tt.worktreePath, tt.branch)
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

	if err := s.AssignLabel("  testing  "); err != nil {
		t.Fatalf("AssignLabel() error = %v", err)
	}
	if s.Label != "testing" {
		t.Fatalf("Label = %q, want trimmed %q", s.Label, "testing")
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

func TestSession_AssignAgentType_Validates(t *testing.T) {
	t.Run("rejects empty agent type", func(t *testing.T) {
		s, _ := NewSession("alpha", uuid.New())
		err := s.AssignAgentType("")
		if !errors.Is(err, ErrAgentTypeRequired) {
			t.Fatalf("AssignAgentType(\"\") error = %v, want %v", err, ErrAgentTypeRequired)
		}
		if s.AgentType != "" {
			t.Fatalf("AgentType = %q, want empty after rejected assignment", s.AgentType)
		}
	})

	t.Run("valid type stores and bumps UpdatedAt", func(t *testing.T) {
		s, _ := NewSession("alpha", uuid.New())
		originalUpdated := s.UpdatedAt
		time.Sleep(time.Millisecond)

		if err := s.AssignAgentType(AgentTypeClaudeCode); err != nil {
			t.Fatalf("AssignAgentType() error = %v", err)
		}
		if s.AgentType != AgentTypeClaudeCode {
			t.Fatalf("AgentType = %q, want %q", s.AgentType, AgentTypeClaudeCode)
		}
		if !s.UpdatedAt.After(originalUpdated) {
			t.Fatalf("UpdatedAt = %v, want after %v", s.UpdatedAt, originalUpdated)
		}
	})
}

func TestTmuxSessionName_FollowsRepositoryNameGuidConvention(t *testing.T) {
	id := uuid.MustParse("12345678-9abc-def0-1234-56789abcdef0")

	got := TmuxSessionName("overseer", "alpha", id)

	if got != "overseer-alpha-12345678" {
		t.Fatalf("TmuxSessionName() = %q, want %q", got, "overseer-alpha-12345678")
	}
}

func TestTmuxSessionName_KebabCasesComponents(t *testing.T) {
	id := uuid.MustParse("12345678-9abc-def0-1234-56789abcdef0")
	tests := []struct {
		name       string
		repository string
		session    string
		want       string
	}{
		{name: "uppercase is lowercased", repository: "OverSeer", session: "Alpha", want: "overseer-alpha-12345678"},
		{name: "dots become dashes", repository: "Overseer.TUI", session: "alpha", want: "overseer-tui-alpha-12345678"},
		{name: "double colons become dashes", repository: "overseer", session: "fix::bugs", want: "overseer-fix-bugs-12345678"},
		{name: "semicolons become dashes", repository: "overseer", session: "a;b", want: "overseer-a-b-12345678"},
		{name: "spaces become dashes", repository: "overseer", session: "my session", want: "overseer-my-session-12345678"},
		{name: "runs of weird chars collapse to one dash", repository: "over--seer", session: "a :: .. b", want: "over-seer-a-b-12345678"},
		{name: "leading and trailing separators are trimmed", repository: ".overseer.", session: " alpha ", want: "overseer-alpha-12345678"},
		{name: "underscores become dashes", repository: "my_repo", session: "my_session", want: "my-repo-my-session-12345678"},
		{name: "emoji and non-ascii are stripped", repository: "over💥seer", session: "café", want: "over-seer-caf-12345678"},
		{name: "empty repository falls back", repository: "", session: "alpha", want: "repository-alpha-12345678"},
		{name: "weird-only repository falls back", repository: ":::", session: "alpha", want: "repository-alpha-12345678"},
		{name: "weird-only session name falls back", repository: "overseer", session: "💥💥", want: "overseer-session-12345678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TmuxSessionName(tt.repository, tt.session, id)
			if got != tt.want {
				t.Fatalf("TmuxSessionName(%q, %q) = %q, want %q", tt.repository, tt.session, got, tt.want)
			}
		})
	}
}

func TestSession_AssignTmuxName_DerivesAndPersists(t *testing.T) {
	s, _ := NewSession("Fix: login bug", uuid.New())
	originalUpdated := s.UpdatedAt
	time.Sleep(time.Millisecond)

	s.AssignTmuxName("Overseer.TUI")

	want := TmuxSessionName("Overseer.TUI", "Fix: login bug", s.ID)
	if s.TmuxName != want {
		t.Fatalf("TmuxName = %q, want %q", s.TmuxName, want)
	}
	if !s.UpdatedAt.After(originalUpdated) {
		t.Fatalf("UpdatedAt = %v, want after %v", s.UpdatedAt, originalUpdated)
	}
}

func TestSession_AgentTmuxName_AppendsAgentSuffix(t *testing.T) {
	s, _ := NewSession("alpha", uuid.New())
	s.AssignTmuxName("overseer")

	if got := s.AgentTmuxName(); got != s.TmuxName+"-agent" {
		t.Fatalf("AgentTmuxName() = %q, want %q", got, s.TmuxName+"-agent")
	}
}
