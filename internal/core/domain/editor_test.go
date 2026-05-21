package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewEditor_CreatesEditorWithProvidedFields(t *testing.T) {
	e, err := NewEditor("VSCode", "code")

	if err != nil {
		t.Fatalf("NewEditor() error = %v", err)
	}
	if e.DisplayName != "VSCode" {
		t.Fatalf("NewEditor() DisplayName = %q, want %q", e.DisplayName, "VSCode")
	}
	if e.Command != "code" {
		t.Fatalf("NewEditor() Command = %q, want %q", e.Command, "code")
	}
}

func TestNewEditor_TrimsFields(t *testing.T) {
	e, err := NewEditor("  Cursor  ", "  cursor --wait  ")
	if err != nil {
		t.Fatalf("NewEditor() error = %v", err)
	}
	if e.DisplayName != "Cursor" {
		t.Fatalf("NewEditor() DisplayName = %q, want trimmed", e.DisplayName)
	}
	if e.Command != "cursor --wait" {
		t.Fatalf("NewEditor() Command = %q, want trimmed", e.Command)
	}
}

func TestNewEditor_PreservesInternalWhitespaceInCommand(t *testing.T) {
	e, err := NewEditor("Neovim", "nvim  --headless  -c  startinsert")
	if err != nil {
		t.Fatalf("NewEditor() error = %v", err)
	}
	if e.Command != "nvim  --headless  -c  startinsert" {
		t.Fatalf("NewEditor() Command = %q, want internal whitespace preserved", e.Command)
	}
}

func TestNewEditor_Validation(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		command     string
		wantErr     error
	}{
		{name: "empty display name", displayName: "", command: "code", wantErr: ErrEditorEmptyDisplayName},
		{name: "blank display name", displayName: "   ", command: "code", wantErr: ErrEditorEmptyDisplayName},
		{name: "empty command", displayName: "VSCode", command: "", wantErr: ErrEditorEmptyCommand},
		{name: "blank command", displayName: "VSCode", command: "   ", wantErr: ErrEditorEmptyCommand},
		{name: "display name too long", displayName: strings.Repeat("a", 101), command: "code", wantErr: ErrEditorDisplayNameTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEditor(tt.displayName, tt.command)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewEditor(%q, %q) error = %v, want %v", tt.displayName, tt.command, err, tt.wantErr)
			}
		})
	}
}

func TestEditor_IsZero_ReportsEmptyEditor(t *testing.T) {
	var zero Editor
	if !zero.IsZero() {
		t.Fatal("Editor{}.IsZero() = false, want true for zero value")
	}

	e, err := NewEditor("VSCode", "code")
	if err != nil {
		t.Fatalf("NewEditor() error = %v", err)
	}
	if e.IsZero() {
		t.Fatal("populated Editor.IsZero() = true, want false")
	}
}
