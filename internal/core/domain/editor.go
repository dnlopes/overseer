package domain

import (
	"errors"
	"strings"
)

const editorDisplayNameMaxLen = 100

type Editor struct {
	DisplayName string
	Command     string
}

func NewEditor(displayName, command string) (Editor, error) {
	displayName = strings.TrimSpace(displayName)
	command = strings.TrimSpace(command)

	if displayName == "" {
		return Editor{}, ErrEditorEmptyDisplayName
	}
	if len(displayName) > editorDisplayNameMaxLen {
		return Editor{}, ErrEditorDisplayNameTooLong
	}
	if command == "" {
		return Editor{}, ErrEditorEmptyCommand
	}

	return Editor{
		DisplayName: displayName,
		Command:     command,
	}, nil
}

func (e Editor) IsZero() bool {
	return e.DisplayName == "" && e.Command == ""
}

var (
	ErrEditorEmptyDisplayName   = errors.New("editor display name cannot be empty")
	ErrEditorDisplayNameTooLong = errors.New("editor display name exceeds 100 characters")
	ErrEditorEmptyCommand       = errors.New("editor command cannot be empty")
)
