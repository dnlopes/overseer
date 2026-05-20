package domain

import (
	"errors"
	"strings"
)

const launcherDisplayNameMaxLen = 100

type Launcher struct {
	DisplayName string
	Command     string
}

func NewLauncher(displayName, command string) (Launcher, error) {
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

	return Launcher{
		DisplayName: displayName,
		Command:     command,
	}, nil
}

func (l Launcher) IsZero() bool {
	return l.DisplayName == "" && l.Command == ""
}

var (
	ErrLauncherEmptyDisplayName   = errors.New("launcher display name cannot be empty")
	ErrLauncherDisplayNameTooLong = errors.New("launcher display name exceeds 100 characters")
	ErrLauncherEmptyCommand       = errors.New("launcher command cannot be empty")
)
