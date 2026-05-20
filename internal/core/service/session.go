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

// defaultAgentCommand is the launcher used by AttachAgent when the session
// has no AgentCommand assigned (i.e. it was created before agent attach
// landed). Keeps pre-existing sessions usable without a forced migration.
const defaultAgentCommand = "opencode"

type SessionService struct {
	repo     domain.SessionRepository
	projects domain.ProjectRepository
	tmux     domain.TmuxAdapter
	git      domain.GitAdapter
	logger   *slog.Logger
}

func NewSessionService(repo domain.SessionRepository, projects domain.ProjectRepository, tmux domain.TmuxAdapter, git domain.GitAdapter, logger *slog.Logger) *SessionService {
	return &SessionService{repo: repo, projects: projects, tmux: tmux, git: git, logger: logger}
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
			paths.SessionWorktreePath(sess.ID),
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

	agentCmd := sess.AgentCommand
	if agentCmd == "" {
		agentCmd = defaultAgentCommand
	}
	if _, err := s.tmux.CreateSession(ctx, sess.ID.String()+"-agent", startDir, agentCmd); err != nil {
		_ = s.tmux.KillSession(ctx, sess.ID.String())
		return CreateSessionResponse{}, fmt.Errorf("create agent tmux session: %w", err)
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
		agentCmd = defaultAgentCommand
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

// --- PreviewSession ---

// PreviewKind selects which tmux session attached to an Overseer session is
// captured by PreviewSession.
type PreviewKind int

const (
	PreviewKindShell PreviewKind = iota
	PreviewKindAgent
)

type PreviewSessionRequest struct {
	ID     uuid.UUID
	Kind   PreviewKind
	Width  int
	Height int
}

// PreviewSessionResponse carries a snapshot of the targeted tmux pane.
// SessionReady is false when the tmux session does not yet exist (typically
// the agent session before its first attach); callers should render a
// placeholder rather than treat this as an error.
type PreviewSessionResponse struct {
	Content      string
	SessionReady bool
}

func (s *SessionService) PreviewSession(ctx context.Context, req PreviewSessionRequest) (PreviewSessionResponse, error) {
	sess, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return PreviewSessionResponse{}, err
	}

	tmuxID := sess.ID.String()
	if req.Kind == PreviewKindAgent {
		tmuxID += "-agent"
	}

	if req.Width > 0 && req.Height > 0 {
		if err := s.tmux.ResizeWindow(ctx, tmuxID, req.Width, req.Height); err != nil {
			if errors.Is(err, domain.ErrTmuxSessionNotFound) {
				return PreviewSessionResponse{SessionReady: false}, nil
			}
			return PreviewSessionResponse{}, fmt.Errorf("resize pane: %w", err)
		}
	}

	content, err := s.tmux.CapturePane(ctx, tmuxID)
	if errors.Is(err, domain.ErrTmuxSessionNotFound) {
		return PreviewSessionResponse{SessionReady: false}, nil
	}
	if err != nil {
		return PreviewSessionResponse{}, fmt.Errorf("capture pane: %w", err)
	}
	return PreviewSessionResponse{Content: content, SessionReady: true}, nil
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
