package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewLauncher_CreatesLauncherWithProvidedFields(t *testing.T) {
	l, err := NewLauncher("OpenCode", "opencode", AgentTypeOpenCode)

	if err != nil {
		t.Fatalf("NewLauncher() error = %v", err)
	}
	if l.DisplayName != "OpenCode" {
		t.Fatalf("NewLauncher() DisplayName = %q, want %q", l.DisplayName, "OpenCode")
	}
	if l.Command != "opencode" {
		t.Fatalf("NewLauncher() Command = %q, want %q", l.Command, "opencode")
	}
	if l.AgentType != AgentTypeOpenCode {
		t.Fatalf("NewLauncher() AgentType = %q, want %q", l.AgentType, AgentTypeOpenCode)
	}
}

func TestNewLauncher_TrimsFields(t *testing.T) {
	l, err := NewLauncher("  Claude Code  ", "  claude --debug  ", AgentTypeClaudeCode)
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
	l, err := NewLauncher("Claude", "claude  --verbose  --json", AgentTypeClaudeCode)
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
		agentType   AgentType
		wantErr     error
	}{
		{name: "empty display name", displayName: "", command: "x", agentType: AgentTypeOpenCode, wantErr: ErrLauncherEmptyDisplayName},
		{name: "blank display name", displayName: "   ", command: "x", agentType: AgentTypeOpenCode, wantErr: ErrLauncherEmptyDisplayName},
		{name: "empty command", displayName: "X", command: "", agentType: AgentTypeOpenCode, wantErr: ErrLauncherEmptyCommand},
		{name: "blank command", displayName: "X", command: "   ", agentType: AgentTypeOpenCode, wantErr: ErrLauncherEmptyCommand},
		{name: "display name too long", displayName: strings.Repeat("a", 101), command: "x", agentType: AgentTypeOpenCode, wantErr: ErrLauncherDisplayNameTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLauncher(tt.displayName, tt.command, tt.agentType)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewLauncher(%q, %q, %q) error = %v, want %v", tt.displayName, tt.command, tt.agentType, err, tt.wantErr)
			}
		})
	}
}

func TestNewLauncher_RequiresAgentType(t *testing.T) {
	_, err := NewLauncher("Claude", "claude", "")
	if !errors.Is(err, ErrAgentTypeRequired) {
		t.Fatalf("NewLauncher(\"Claude\", \"claude\", \"\") error = %v, want %v", err, ErrAgentTypeRequired)
	}
}

func TestLauncher_IsZero_ReportsEmptyLauncher(t *testing.T) {
	var zero Launcher
	if !zero.IsZero() {
		t.Fatal("Launcher{}.IsZero() = false, want true for zero value")
	}

	l, err := NewLauncher("X", "x", AgentTypeOpenCode)
	if err != nil {
		t.Fatalf("NewLauncher() error = %v", err)
	}
	if l.IsZero() {
		t.Fatal("populated Launcher.IsZero() = true, want false")
	}
}
