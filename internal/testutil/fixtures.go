package testutil

import (
	"regexp"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/dnlopes/overseer/internal/core/domain"
)

// MakeSession builds a project-less Session (no worktree). Tests that need a
// project-backed Session with worktree fields populated should use
// MakeSessionWithWorktree.
func MakeSession(name string, projectID uuid.UUID) domain.Session {
	s, err := domain.NewSession(name, projectID)
	if err != nil {
		panic(err)
	}
	s.AssignTmuxName("project")
	return s
}

// MakeSessionWithWorktree builds a project-backed Mode 1 Session populated
// with the supplied worktree path and branch.
func MakeSessionWithWorktree(name string, projectID uuid.UUID, worktreePath, branch string) domain.Session {
	s, err := domain.NewSession(name, projectID)
	if err != nil {
		panic(err)
	}
	if err := s.AssignWorktree(worktreePath, branch); err != nil {
		panic(err)
	}
	s.AssignTmuxName("project")
	return s
}

func MakeProject(path, name string) domain.Project {
	p, err := domain.NewProject(path, name)
	if err != nil {
		panic(err)
	}
	return p
}

// tmuxNamePattern matches a well-formed shell tmux session name of the form
// "<repository>-<session-name>-<guid8>" (see domain.TmuxSessionName).
var tmuxNamePattern = regexp.MustCompile(`^[a-z0-9-]+-[0-9a-f]{8}$`)

// agentTmuxNamePattern matches the agent variant of the tmux session name:
// tmuxNamePattern with the conventional "-agent" suffix.
var agentTmuxNamePattern = regexp.MustCompile(`^[a-z0-9-]+-[0-9a-f]{8}-agent$`)

// TmuxNameString matches any well-formed shell tmux session name — used to
// assert the service passes the sanitized Session.TmuxName (rather than a
// raw Session.ID) to the tmux adapter.
func TmuxNameString() interface{} {
	return mock.MatchedBy(func(s string) bool {
		return tmuxNamePattern.MatchString(s)
	})
}

// AgentTmuxNameString matches well-formed agent tmux session names, used by
// the service to name the tmux session that hosts the agent process.
func AgentTmuxNameString() interface{} {
	return mock.MatchedBy(func(s string) bool {
		return agentTmuxNamePattern.MatchString(s)
	})
}
