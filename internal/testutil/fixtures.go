package testutil

import (
	"strings"

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
	return s
}

// MakeSessionWithWorktree builds a project-backed Session populated with the
// supplied worktree path and branch names.
func MakeSessionWithWorktree(name string, projectID uuid.UUID, worktreePath, baseBranch, featureBranch string) domain.Session {
	s, err := domain.NewSession(name, projectID)
	if err != nil {
		panic(err)
	}
	if err := s.AssignWorktree(worktreePath, baseBranch, featureBranch); err != nil {
		panic(err)
	}
	return s
}

func MakeProject(path, name string) domain.Project {
	p, err := domain.NewProject(path, name)
	if err != nil {
		panic(err)
	}
	return p
}

// UUIDString matches any string that parses as a UUID — used to assert the service
// passes a Session.ID (rather than a user-typed name) as the tmux session name.
func UUIDString() interface{} {
	return mock.MatchedBy(func(s string) bool {
		_, err := uuid.Parse(s)
		return err == nil
	})
}

// AgentTmuxIDString matches strings of the form "<uuid>-agent", used by the
// service to name the tmux session that hosts the agent process.
func AgentTmuxIDString() interface{} {
	return mock.MatchedBy(func(s string) bool {
		if !strings.HasSuffix(s, "-agent") {
			return false
		}
		_, err := uuid.Parse(strings.TrimSuffix(s, "-agent"))
		return err == nil
	})
}
