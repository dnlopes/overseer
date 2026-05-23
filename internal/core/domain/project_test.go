package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewProject_CreatesProjectWithProvidedName(t *testing.T) {
	before := time.Now()

	p, err := NewProject("/home/user/repos/overseer", "overseer")

	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if p.ID == uuid.Nil {
		t.Fatal("NewProject() ID is nil")
	}
	if p.Name != "overseer" {
		t.Fatalf("NewProject() Name = %q, want %q", p.Name, "overseer")
	}
	if p.Path != "/home/user/repos/overseer" {
		t.Fatalf("NewProject() Path = %q, want %q", p.Path, "/home/user/repos/overseer")
	}
	if p.CreatedAt.Before(before) {
		t.Fatalf("NewProject() CreatedAt = %v, before creation start %v", p.CreatedAt, before)
	}
	if !p.CreatedAt.Equal(p.UpdatedAt) {
		t.Fatalf("NewProject() CreatedAt = %v, UpdatedAt = %v, want equal", p.CreatedAt, p.UpdatedAt)
	}
}

func TestNewProject_DerivesNameFromPathBasenameWhenNameEmpty(t *testing.T) {
	p, err := NewProject("/home/user/repos/overseer", "")
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if p.Name != "overseer" {
		t.Fatalf("NewProject() Name = %q, want %q (basename)", p.Name, "overseer")
	}
}

func TestNewProject_DerivesNameWhenNameIsOnlyWhitespace(t *testing.T) {
	p, err := NewProject("/srv/code/widgets", "   ")
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if p.Name != "widgets" {
		t.Fatalf("NewProject() Name = %q, want %q (basename)", p.Name, "widgets")
	}
}

func TestNewProject_TrimsPathAndName(t *testing.T) {
	p, err := NewProject("  /home/user/repos/overseer  ", "  Overseer  ")
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if p.Path != "/home/user/repos/overseer" {
		t.Fatalf("NewProject() Path = %q, want trimmed", p.Path)
	}
	if p.Name != "Overseer" {
		t.Fatalf("NewProject() Name = %q, want trimmed", p.Name)
	}
}

func TestNewProject_Validation(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		project string
		wantErr error
	}{
		{name: "empty path", path: "", project: "x", wantErr: ErrProjectEmptyPath},
		{name: "blank path", path: "   ", project: "x", wantErr: ErrProjectEmptyPath},
		{name: "relative path", path: "repos/overseer", project: "x", wantErr: ErrProjectPathNotAbsolute},
		{name: "dot-relative path", path: "./overseer", project: "x", wantErr: ErrProjectPathNotAbsolute},
		{name: "name too long", path: "/abs/path", project: strings.Repeat("a", 101), wantErr: ErrProjectNameTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProject(tt.path, tt.project)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewProject(%q, %q) error = %v, want %v", tt.path, tt.project, err, tt.wantErr)
			}
		})
	}
}

func TestNewProject_AcceptsExactlyOneHundredCharacterName(t *testing.T) {
	exactly100 := strings.Repeat("a", 100)
	p, err := NewProject("/abs/path", exactly100)
	if err != nil {
		t.Fatalf("NewProject() error = %v, want nil for 100-char name", err)
	}
	if p.Name != exactly100 {
		t.Fatalf("NewProject() Name length = %d, want 100", len(p.Name))
	}
}

func TestProject_Rename_UpdatesNameAndUpdatedAt(t *testing.T) {
	p, err := NewProject("/repo/overseer", "old")
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	p.UpdatedAt = time.Now().Add(-time.Minute)
	beforeRename := p.UpdatedAt

	if err := p.Rename("new"); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if p.Name != "new" {
		t.Fatalf("Rename() Name = %q, want %q", p.Name, "new")
	}
	if !p.UpdatedAt.After(beforeRename) {
		t.Fatalf("Rename() UpdatedAt = %v, want after %v", p.UpdatedAt, beforeRename)
	}
}

func TestProject_Rename_TrimsName(t *testing.T) {
	p, err := NewProject("/repo/overseer", "old")
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}

	if err := p.Rename("  new  "); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if p.Name != "new" {
		t.Fatalf("Rename() Name = %q, want trimmed %q", p.Name, "new")
	}
}

func TestProject_Rename_RejectsEmptyName(t *testing.T) {
	p, err := NewProject("/repo/overseer", "old")
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}

	if err := p.Rename("   "); !errors.Is(err, ErrProjectEmptyName) {
		t.Fatalf("Rename() error = %v, want %v", err, ErrProjectEmptyName)
	}
	if p.Name != "old" {
		t.Fatalf("Rename() left Name = %q, want %q (unchanged on validation failure)", p.Name, "old")
	}
}

func TestProject_Rename_RejectsNameTooLong(t *testing.T) {
	p, err := NewProject("/repo/overseer", "old")
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}

	if err := p.Rename(strings.Repeat("a", 101)); !errors.Is(err, ErrProjectNameTooLong) {
		t.Fatalf("Rename() error = %v, want %v", err, ErrProjectNameTooLong)
	}
}

func TestProject_Rename_AcceptsExactlyOneHundredCharacterName(t *testing.T) {
	p, err := NewProject("/repo/overseer", "old")
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	exactly100 := strings.Repeat("a", 100)

	if err := p.Rename(exactly100); err != nil {
		t.Fatalf("Rename() error = %v, want nil for 100-char name", err)
	}
	if p.Name != exactly100 {
		t.Fatalf("Rename() Name length = %d, want 100", len(p.Name))
	}
}
