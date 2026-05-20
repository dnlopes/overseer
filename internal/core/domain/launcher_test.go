package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewLauncher_CreatesLauncherWithProvidedFields(t *testing.T) {
	l, err := NewLauncher("OpenCode", "opencode")

	if err != nil {
		t.Fatalf("NewLauncher() error = %v", err)
	}
	if l.DisplayName != "OpenCode" {
		t.Fatalf("NewLauncher() DisplayName = %q, want %q", l.DisplayName, "OpenCode")
	}
	if l.Command != "opencode" {
		t.Fatalf("NewLauncher() Command = %q, want %q", l.Command, "opencode")
	}
}

func TestNewLauncher_TrimsFields(t *testing.T) {
	l, err := NewLauncher("  Claude Code  ", "  claude --debug  ")
	if err != nil {
		t.Fatalf("NewLauncher() error = %v", err)
	}
	if l.DisplayName != "Claude Code" {
		t.Fatalf("NewLauncher() DisplayName = %q, want trimmed", l.DisplayName)
	}
	if l.Command != "claude --debug" {
		t.Fatalf("NewLauncher() Command = %q, want trimmed", l.Command)
	}
}

func TestNewLauncher_PreservesInternalWhitespaceInCommand(t *testing.T) {
	l, err := NewLauncher("Claude", "claude  --verbose  --json")
	if err != nil {
		t.Fatalf("NewLauncher() error = %v", err)
	}
	if l.Command != "claude  --verbose  --json" {
		t.Fatalf("NewLauncher() Command = %q, want internal whitespace preserved", l.Command)
	}
}

func TestNewLauncher_Validation(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		command     string
		wantErr     error
	}{
		{name: "empty display name", displayName: "", command: "x", wantErr: ErrLauncherEmptyDisplayName},
		{name: "blank display name", displayName: "   ", command: "x", wantErr: ErrLauncherEmptyDisplayName},
		{name: "empty command", displayName: "X", command: "", wantErr: ErrLauncherEmptyCommand},
		{name: "blank command", displayName: "X", command: "   ", wantErr: ErrLauncherEmptyCommand},
		{name: "display name too long", displayName: strings.Repeat("a", 101), command: "x", wantErr: ErrLauncherDisplayNameTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLauncher(tt.displayName, tt.command)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewLauncher(%q, %q) error = %v, want %v", tt.displayName, tt.command, err, tt.wantErr)
			}
		})
	}
}

func TestLauncher_IsZero_ReportsEmptyLauncher(t *testing.T) {
	var zero Launcher
	if !zero.IsZero() {
		t.Fatal("Launcher{}.IsZero() = false, want true for zero value")
	}

	l, err := NewLauncher("X", "x")
	if err != nil {
		t.Fatalf("NewLauncher() error = %v", err)
	}
	if l.IsZero() {
		t.Fatal("populated Launcher.IsZero() = true, want false")
	}
}
