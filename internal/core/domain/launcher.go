package domain

import (
	"errors"
	"strings"
)

const launcherDisplayNameMaxLen = 100

type Launcher struct {
	DisplayName string
	Command     string
	AgentType   AgentType
}

func NewLauncher(displayName, command string, agentType AgentType) (Launcher, error) {
	displayName = strings.TrimSpace(displayName)
	command = strings.TrimSpace(command)

	if displayName == "" {
		return Launcher{}, ErrLauncherEmptyDisplayName
	}
	if len(displayName) > launcherDisplayNameMaxLen {
		return Launcher{}, ErrLauncherDisplayNameTooLong
	}
	if command == "" {
		return Launcher{}, ErrLauncherEmptyCommand
	}
	if agentType == "" {
		return Launcher{}, ErrAgentTypeRequired
	}

	return Launcher{
		DisplayName: displayName,
		Command:     command,
		AgentType:   agentType,
	}, nil
}

func (l Launcher) IsZero() bool {
	return l.DisplayName == "" && l.Command == "" && l.AgentType == ""
}

var (
	ErrLauncherEmptyDisplayName   = errors.New("launcher display name cannot be empty")
	ErrLauncherDisplayNameTooLong = errors.New("launcher display name exceeds 100 characters")
	ErrLauncherEmptyCommand       = errors.New("launcher command cannot be empty")
)
