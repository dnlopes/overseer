package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sort"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/shared/errs"
	"github.com/dnlopes/overseer/internal/shared/paths"
)

// defaultBaseBranch is the fork point used when CreateSessionRequest does
// not specify one. Surfacing this as a request field is follow-up work.
const defaultBaseBranch = "main"

type SessionService struct {
	repo            domain.SessionRepository
	projects        domain.ProjectRepository
	tmux            domain.TmuxAdapter
	git             domain.GitAdapter
	pathsResolver   paths.Resolver
	defaultLauncher domain.Launcher
	logger          *slog.Logger
}

// NewSessionService wires the session use-cases. defaultLauncher is used by
// AttachAgent when a session's AgentCommand is empty (pre-launcher sessions).
func NewSessionService(
	repo domain.SessionRepository,
	projects domain.ProjectRepository,
	tmux domain.TmuxAdapter,
	git domain.GitAdapter,
	resolver paths.Resolver,
	defaultLauncher domain.Launcher,
	logger *slog.Logger,
) *SessionService {
	return &SessionService{
		repo:            repo,
		projects:        projects,
		tmux:            tmux,
		git:             git,
		pathsResolver:   resolver,
		defaultLauncher: defaultLauncher,
		logger:          logger,
	}
}

// --- Create ---

type CreateSessionRequest struct {
	Name         string
	ProjectID    uuid.UUID
	AgentCommand string
}

type CreateSessionResponse struct {
	Session domain.Session
}

func (s *SessionService) Create(ctx context.Context, req CreateSessionRequest) (CreateSessionResponse, error) {
	sess, err := domain.NewSession(req.Name, req.ProjectID)
	if err != nil {
		return CreateSessionResponse{}, err
	}

	if req.AgentCommand != "" {
		if err := sess.AssignAgentCommand(req.AgentCommand); err != nil {
			return CreateSessionResponse{}, err
		}
	}

	var repoPath string
	if req.ProjectID != uuid.Nil {
		project, err := s.projects.Get(ctx, req.ProjectID)
		if err != nil {
			return CreateSessionResponse{}, fmt.Errorf("lookup project: %w", err)
		}
		repoPath = project.Path
		if err := sess.AssignWorktree(
			s.pathsResolver.SessionWorktreePath(sess.ID),
			defaultBaseBranch,
			paths.SessionFeatureBranch(sess.ID),
		); err != nil {
			return CreateSessionResponse{}, fmt.Errorf("assign worktree: %w", err)
		}
	}

	existing, err := s.repo.List(ctx)
	if err != nil {
		return CreateSessionResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	nextOrder := 1
	for _, candidate := range existing {
		if candidate.ProjectID != sess.ProjectID {
			continue
		}
		if candidate.Name == sess.Name {
			return CreateSessionResponse{}, domain.ErrSessionAlreadyExists
		}
		if candidate.Order >= nextOrder {
			nextOrder = candidate.Order + 1
		}
	}
	sess.Order = nextOrder

	startDir, err := sessionStartDir(sess)
	if err != nil {
		return CreateSessionResponse{}, err
	}

	if sess.HasWorktree() {
		if err := s.git.CreateWorktree(ctx, repoPath, sess.BaseBranch, sess.FeatureBranch, sess.WorktreePath); err != nil {
			return CreateSessionResponse{}, fmt.Errorf("create git worktree: %w", err)
		}
	}
	if _, err := s.tmux.CreateSession(ctx, sess.ID.String(), startDir, ""); err != nil {
		return CreateSessionResponse{}, fmt.Errorf("create tmux session: %w", err)
	}
	if err := s.repo.Save(ctx, sess); err != nil {
		return CreateSessionResponse{}, fmt.Errorf("save session: %w", err)
	}

	return CreateSessionResponse{Session: sess}, nil
}

// --- Rename ---

type RenameSessionRequest struct {
	ID      uuid.UUID
	NewName string
}

type RenameSessionResponse struct {
	Session domain.Session
}

func (s *SessionService) Rename(ctx context.Context, req RenameSessionRequest) (RenameSessionResponse, error) {
	sess, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return RenameSessionResponse{}, err
	}

	existing, err := s.repo.List(ctx)
	if err != nil {
		return RenameSessionResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	for _, candidate := range existing {
		if candidate.ID == sess.ID {
			continue
		}
		if candidate.ProjectID == sess.ProjectID && candidate.Name == req.NewName {
			return RenameSessionResponse{}, domain.ErrSessionAlreadyExists
		}
	}

	if err := sess.Rename(req.NewName); err != nil {
		return RenameSessionResponse{}, err
	}

	if err := s.repo.Save(ctx, sess); err != nil {
		return RenameSessionResponse{}, fmt.Errorf("save session: %w", err)
	}

	return RenameSessionResponse{Session: sess}, nil
}

// --- List ---

type ListSessionsRequest struct{}

type ListSessionsResponse struct {
	Sessions []domain.Session
}

func (s *SessionService) List(ctx context.Context, _ ListSessionsRequest) (ListSessionsResponse, error) {
	sessions, err := s.repo.List(ctx)
	if err != nil {
		return ListSessionsResponse{}, err
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].ProjectID == sessions[j].ProjectID {
			return sessions[i].Order < sessions[j].Order
		}
		return sessions[i].ProjectID.String() < sessions[j].ProjectID.String()
	})

	return ListSessionsResponse{Sessions: sessions}, nil
}

// --- Reorder ---

type ReorderSessionRequest struct {
	ID        uuid.UUID
	Direction int // +1 = down (higher order), -1 = up (lower order)
}

type ReorderSessionResponse struct {
	Sessions []domain.Session
}

func (s *SessionService) Reorder(ctx context.Context, req ReorderSessionRequest) (ReorderSessionResponse, error) {
	target, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return ReorderSessionResponse{}, err
	}

	all, err := s.repo.List(ctx)
	if err != nil {
		return ReorderSessionResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	projectSessions := make([]domain.Session, 0, len(all))
	for _, sess := range all {
		if sess.ProjectID == target.ProjectID {
			projectSessions = append(projectSessions, sess)
		}
	}
	sort.Slice(projectSessions, func(i, j int) bool {
		return projectSessions[i].Order < projectSessions[j].Order
	})

	if len(projectSessions) <= 1 {
		return ReorderSessionResponse{}, errs.ErrNoOp
	}

	idx := -1
	for i, sess := range projectSessions {
		if sess.ID == target.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ReorderSessionResponse{}, fmt.Errorf("session %s not found in project list", target.ID)
	}

	if (idx == 0 && req.Direction == -1) || (idx == len(projectSessions)-1 && req.Direction == 1) {
		return ReorderSessionResponse{}, errs.ErrNoOp
	}

	neighbor := idx + req.Direction

	projectSessions[idx].Order, projectSessions[neighbor].Order = projectSessions[neighbor].Order, projectSessions[idx].Order
	projectSessions[idx].UpdatedAt = projectSessions[neighbor].UpdatedAt

	if err := s.repo.Save(ctx, projectSessions[idx]); err != nil {
		return ReorderSessionResponse{}, fmt.Errorf("save target session: %w", err)
	}
	if err := s.repo.Save(ctx, projectSessions[neighbor]); err != nil {
		return ReorderSessionResponse{}, fmt.Errorf("save neighbor session: %w", err)
	}

	sort.Slice(projectSessions, func(i, j int) bool {
		return projectSessions[i].Order < projectSessions[j].Order
	})

	s.logger.InfoContext(ctx, "session reordered",
		slog.String("id", target.ID.String()),
		slog.Int("direction", req.Direction),
	)

	return ReorderSessionResponse{Sessions: projectSessions}, nil
}

// --- AttachShell ---

type AttachShellRequest struct {
	ID uuid.UUID
}

type AttachShellResponse struct {
	Command *exec.Cmd
}

func (s *SessionService) AttachShell(ctx context.Context, req AttachShellRequest) (AttachShellResponse, error) {
	sess, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return AttachShellResponse{}, err
	}

	tmuxID := sess.ID.String()
	startDir, err := sessionStartDir(sess)
	if err != nil {
		return AttachShellResponse{}, err
	}
	if err := s.ensureTmuxSession(ctx, tmuxID, startDir, ""); err != nil {
		return AttachShellResponse{}, err
	}

	cmd, err := s.tmux.AttachCommand(ctx, tmuxID)
	if err != nil {
		return AttachShellResponse{}, fmt.Errorf("attach tmux session: %w", err)
	}

	s.logger.InfoContext(ctx, "shell attach prepared",
		slog.String("id", tmuxID),
	)

	return AttachShellResponse{Command: cmd}, nil
}

// --- AttachAgent ---

type AttachAgentRequest struct {
	ID uuid.UUID
}

type AttachAgentResponse struct {
	Command *exec.Cmd
}

func (s *SessionService) AttachAgent(ctx context.Context, req AttachAgentRequest) (AttachAgentResponse, error) {
	sess, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return AttachAgentResponse{}, err
	}

	agentTmuxID := sess.ID.String() + "-agent"
	startDir, err := sessionStartDir(sess)
	if err != nil {
		return AttachAgentResponse{}, err
	}
	agentCmd := sess.AgentCommand
	if agentCmd == "" {
		agentCmd = s.defaultLauncher.Command
	}
	if agentCmd == "" {
		return AttachAgentResponse{}, domain.ErrSessionNoAgentCommandAvailable
	}
	if err := s.ensureTmuxSession(ctx, agentTmuxID, startDir, agentCmd); err != nil {
		return AttachAgentResponse{}, err
	}

	cmd, err := s.tmux.AttachCommand(ctx, agentTmuxID)
	if err != nil {
		return AttachAgentResponse{}, fmt.Errorf("attach tmux session: %w", err)
	}

	s.logger.InfoContext(ctx, "agent attach prepared",
		slog.String("id", agentTmuxID),
		slog.String("agent_command", agentCmd),
	)

	return AttachAgentResponse{Command: cmd}, nil
}

// --- Delete ---

type DeleteSessionRequest struct {
	ID uuid.UUID
}

type DeleteSessionResponse struct{}

// Delete tears down a session in three steps, in this order:
//
//  1. If the session has a worktree, the path is verified to live inside
//     paths.WorktreeRoot() (defence in depth against a tampered DB row) and
//     the git worktree is removed. If the owning project no longer exists in
//     the repository, git removal is skipped with a warning — the rest of
//     the teardown still proceeds.
//  2. The associated tmux session is killed, if it still exists. A missing
//     tmux session is not an error: the user may have killed it manually or
//     the tmux server may have restarted.
//  3. The session row is deleted from the repository last, so any failure in
//     steps 1 or 2 leaves a retriable session row instead of an orphaned
//     worktree or tmux session paired with no DB record.
func (s *SessionService) Delete(ctx context.Context, req DeleteSessionRequest) (DeleteSessionResponse, error) {
	sess, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return DeleteSessionResponse{}, err
	}

	if sess.HasWorktree() {
		if !sess.WorktreeIsInsideRoot(s.pathsResolver.WorktreeRoot()) {
			s.logger.ErrorContext(ctx, "session worktree path outside managed root, refusing to delete",
				slog.String("id", sess.ID.String()),
				slog.String("worktree_path", sess.WorktreePath),
				slog.String("worktree_root", s.pathsResolver.WorktreeRoot()),
			)
			return DeleteSessionResponse{}, domain.ErrSessionWorktreePathOutsideRoot
		}
		if err := s.removeWorktreeForSession(ctx, sess); err != nil {
			return DeleteSessionResponse{}, err
		}
	}

	if err := s.killTmuxIfExists(ctx, sess.ID.String()); err != nil {
		return DeleteSessionResponse{}, err
	}

	if err := s.repo.Delete(ctx, sess.ID); err != nil {
		return DeleteSessionResponse{}, fmt.Errorf("delete session: %w", err)
	}

	s.logger.InfoContext(ctx, "session deleted",
		slog.String("id", sess.ID.String()),
		slog.String("name", sess.Name),
	)
	return DeleteSessionResponse{}, nil
}

func (s *SessionService) removeWorktreeForSession(ctx context.Context, sess domain.Session) error {
	project, err := s.projects.Get(ctx, sess.ProjectID)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			s.logger.WarnContext(ctx, "owning project missing, skipping git worktree removal",
				slog.String("session_id", sess.ID.String()),
				slog.String("project_id", sess.ProjectID.String()),
				slog.String("worktree_path", sess.WorktreePath),
			)
			return nil
		}
		return fmt.Errorf("lookup project: %w", err)
	}
	if err := s.git.RemoveWorktree(ctx, project.Path, sess.WorktreePath); err != nil {
		return fmt.Errorf("remove git worktree: %w", err)
	}
	return nil
}

func (s *SessionService) killTmuxIfExists(ctx context.Context, tmuxID string) error {
	if _, err := s.tmux.GetSession(ctx, tmuxID); err != nil {
		if errors.Is(err, domain.ErrTmuxSessionNotFound) {
			s.logger.InfoContext(ctx, "tmux session already gone, nothing to kill",
				slog.String("id", tmuxID),
			)
			return nil
		}
		return fmt.Errorf("inspect tmux session: %w", err)
	}
	if err := s.tmux.KillSession(ctx, tmuxID); err != nil {
		return fmt.Errorf("kill tmux session: %w", err)
	}
	return nil
}

func (s *SessionService) ensureTmuxSession(ctx context.Context, tmuxID, startDir, shellCommand string) error {
	_, err := s.tmux.GetSession(ctx, tmuxID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrTmuxSessionNotFound) {
		return fmt.Errorf("inspect tmux session: %w", err)
	}

	if _, err := s.tmux.CreateSession(ctx, tmuxID, startDir, shellCommand); err != nil {
		return fmt.Errorf("recreate tmux session: %w", err)
	}
	s.logger.InfoContext(ctx, "tmux session recreated",
		slog.String("id", tmuxID),
	)
	return nil
}

// sessionStartDir resolves the working directory a session's tmux session
// should open in. Project-backed sessions use their worktree; project-less
// sessions fall back to the user's home directory.
func sessionStartDir(sess domain.Session) (string, error) {
	if sess.HasWorktree() {
		return sess.WorktreePath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return home, nil
}
