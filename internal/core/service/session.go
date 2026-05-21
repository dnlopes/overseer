package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/shared/errs"
	"github.com/dnlopes/overseer/internal/shared/paths"
)

type SessionService struct {
	repo            domain.SessionRepository
	projects        domain.ProjectRepository
	tmux            domain.TmuxAdapter
	git             domain.GitAdapter
	pathsResolver   paths.Resolver
	defaultLauncher domain.Launcher
	defaultEditor   domain.Editor
	logger          *slog.Logger
}

// NewSessionService wires the session use-cases. defaultLauncher is used by
// AttachAgent when a session's AgentCommand is empty (pre-launcher sessions);
// defaultEditor plays the same role for OpenEditor and Session.EditorCommand.
func NewSessionService(
	repo domain.SessionRepository,
	projects domain.ProjectRepository,
	tmux domain.TmuxAdapter,
	git domain.GitAdapter,
	resolver paths.Resolver,
	defaultLauncher domain.Launcher,
	defaultEditor domain.Editor,
	logger *slog.Logger,
) *SessionService {
	return &SessionService{
		repo:            repo,
		projects:        projects,
		tmux:            tmux,
		git:             git,
		pathsResolver:   resolver,
		defaultLauncher: defaultLauncher,
		defaultEditor:   defaultEditor,
		logger:          logger,
	}
}

// --- Create ---

type CreateSessionRequest struct {
	Name          string
	ProjectID     uuid.UUID
	BaseBranch    string
	AgentCommand  string
	EditorCommand string
}

type CreateSessionResponse struct {
	Session domain.Session
}

func (s *SessionService) Create(ctx context.Context, req CreateSessionRequest) (CreateSessionResponse, error) {
	if req.BaseBranch == "" {
		return CreateSessionResponse{}, domain.ErrSessionEmptyBaseBranch
	}

	sess, err := domain.NewSession(req.Name, req.ProjectID)
	if err != nil {
		return CreateSessionResponse{}, err
	}

	if req.AgentCommand != "" {
		if err := sess.AssignAgentCommand(req.AgentCommand); err != nil {
			return CreateSessionResponse{}, err
		}
	}

	if req.EditorCommand != "" {
		if err := sess.AssignEditorCommand(req.EditorCommand); err != nil {
			return CreateSessionResponse{}, err
		}
	}

	project, err := s.projects.Get(ctx, req.ProjectID)
	if err != nil {
		return CreateSessionResponse{}, fmt.Errorf("lookup project: %w", err)
	}
	if err := sess.AssignWorktree(
		s.pathsResolver.SessionWorktreePath(sess.ID),
		req.BaseBranch,
		paths.SessionFeatureBranch(sess.ID),
	); err != nil {
		return CreateSessionResponse{}, fmt.Errorf("assign worktree: %w", err)
	}

	nextOrder, err := s.assignNextOrderForProject(ctx, sess)
	if err != nil {
		return CreateSessionResponse{}, err
	}
	sess.Order = nextOrder

	if err := s.git.CreateWorktree(ctx, project.Path, sess.BaseBranch, sess.FeatureBranch, sess.WorktreePath); err != nil {
		return CreateSessionResponse{}, fmt.Errorf("create git worktree: %w", err)
	}

	sess, err = s.spinUpTmuxAndPersist(ctx, sess, project)
	if err != nil {
		return CreateSessionResponse{}, err
	}
	return CreateSessionResponse{Session: sess}, nil
}

// spinUpTmuxAndPersist runs the post-git tail shared by Create and CheckoutBranch:
// it creates the shell + agent tmux sessions, saves the session row, and
// bumps the project's UpdatedAt for recency ordering. The agent tmux session
// uses sess.AgentCommand, falling back to the default launcher when empty —
// matching the lazy launch semantics in ensureTmuxSession.
func (s *SessionService) spinUpTmuxAndPersist(ctx context.Context, sess domain.Session, project domain.Project) (domain.Session, error) {
	if _, err := s.tmux.CreateSession(ctx, sess.ID.String(), sess.WorktreePath, ""); err != nil {
		return domain.Session{}, fmt.Errorf("create tmux session: %w", err)
	}

	agentCmd := sess.AgentCommand
	if agentCmd == "" {
		agentCmd = s.defaultLauncher.Command
	}
	if _, err := s.tmux.CreateSession(ctx, sess.ID.String()+"-agent", sess.WorktreePath, agentCmd); err != nil {
		_ = s.tmux.KillSession(ctx, sess.ID.String())
		return domain.Session{}, fmt.Errorf("create agent tmux session: %w", err)
	}

	if err := s.repo.Save(ctx, sess); err != nil {
		return domain.Session{}, fmt.Errorf("save session: %w", err)
	}

	project.UpdatedAt = sess.CreatedAt
	if err := s.projects.Save(ctx, project); err != nil {
		s.logger.WarnContext(ctx, "bump project UpdatedAt failed; recency ordering may lag",
			slog.String("project_id", project.ID.String()),
			slog.String("error", err.Error()),
		)
	}

	return sess, nil
}

// --- CheckoutBranch ---

type CheckoutBranchRequest struct {
	Name          string
	ProjectID     uuid.UUID
	Branch        string
	AgentCommand  string
	EditorCommand string
}

type CheckoutBranchResponse struct {
	Session domain.Session
}

func (s *SessionService) CheckoutBranch(ctx context.Context, req CheckoutBranchRequest) (CheckoutBranchResponse, error) {
	if req.Branch == "" {
		return CheckoutBranchResponse{}, domain.ErrSessionEmptyBaseBranch
	}

	sess, err := domain.NewCheckoutSession(req.Name, req.ProjectID)
	if err != nil {
		return CheckoutBranchResponse{}, err
	}

	if req.AgentCommand != "" {
		if err := sess.AssignAgentCommand(req.AgentCommand); err != nil {
			return CheckoutBranchResponse{}, err
		}
	}

	if req.EditorCommand != "" {
		if err := sess.AssignEditorCommand(req.EditorCommand); err != nil {
			return CheckoutBranchResponse{}, err
		}
	}

	project, err := s.projects.Get(ctx, req.ProjectID)
	if err != nil {
		return CheckoutBranchResponse{}, fmt.Errorf("lookup project: %w", err)
	}
	if err := sess.AssignWorktree(
		s.pathsResolver.SessionWorktreePath(sess.ID),
		req.Branch,
		paths.SessionTrackingBranch(sess.ID),
	); err != nil {
		return CheckoutBranchResponse{}, fmt.Errorf("assign worktree: %w", err)
	}

	nextOrder, err := s.assignNextOrderForProject(ctx, sess)
	if err != nil {
		return CheckoutBranchResponse{}, err
	}
	sess.Order = nextOrder

	if err := s.git.CreateTrackingWorktree(ctx, project.Path, sess.BaseBranch, sess.FeatureBranch, sess.WorktreePath); err != nil {
		return CheckoutBranchResponse{}, fmt.Errorf("create tracking worktree: %w", err)
	}

	sess, err = s.spinUpTmuxAndPersist(ctx, sess, project)
	if err != nil {
		return CheckoutBranchResponse{}, err
	}
	return CheckoutBranchResponse{Session: sess}, nil
}

// assignNextOrderForProject computes the next Order value for a session
// inside its project, and returns ErrSessionAlreadyExists if a sibling
// session shares the same name. Shared by Create and CheckoutBranch.
func (s *SessionService) assignNextOrderForProject(ctx context.Context, sess domain.Session) (int, error) {
	existing, err := s.repo.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("list sessions: %w", err)
	}
	nextOrder := 1
	for _, candidate := range existing {
		if candidate.ProjectID != sess.ProjectID {
			continue
		}
		if candidate.Name == sess.Name {
			return 0, domain.ErrSessionAlreadyExists
		}
		if candidate.Order >= nextOrder {
			nextOrder = candidate.Order + 1
		}
	}
	return nextOrder, nil
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
	if err := s.ensureTmuxSession(ctx, tmuxID, sess, ""); err != nil {
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
	agentCmd := sess.AgentCommand
	if agentCmd == "" {
		agentCmd = s.defaultLauncher.Command
	}
	if agentCmd == "" {
		return AttachAgentResponse{}, domain.ErrSessionNoAgentCommandAvailable
	}
	if err := s.ensureTmuxSession(ctx, agentTmuxID, sess, agentCmd); err != nil {
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

// --- OpenEditor ---

type OpenEditorRequest struct {
	ID uuid.UUID
}

type OpenEditorResponse struct {
	Command *exec.Cmd
}

// OpenEditor launches the configured editor at the session's worktree as a
// detached background process: the TUI keeps running, no alt-screen flash.
//
// The returned *exec.Cmd has already been Start()ed — do NOT call Start/Run
// on it again; it is exposed only for observability. A goroutine Wait()s on
// the child to reap it.
//
// exec.Command, NOT exec.CommandContext, so the editor outlives ctx.
// Terminal editors (vim/nvim/helix) are NOT supported by this fire-and-forget
// model — they need the terminal we never release.
func (s *SessionService) OpenEditor(ctx context.Context, req OpenEditorRequest) (OpenEditorResponse, error) {
	sess, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return OpenEditorResponse{}, err
	}

	editorCmd := sess.EditorCommand
	if editorCmd == "" {
		editorCmd = s.defaultEditor.Command
	}
	if editorCmd == "" {
		return OpenEditorResponse{}, domain.ErrSessionNoEditorCommandAvailable
	}

	workDir, err := sessionStartDir(sess)
	if err != nil {
		return OpenEditorResponse{}, err
	}

	parts := strings.Fields(editorCmd)
	args := append(parts[1:], workDir)
	cmd := exec.Command(parts[0], args...)
	cmd.Dir = workDir

	if err := cmd.Start(); err != nil {
		return OpenEditorResponse{}, fmt.Errorf("start editor: %w", err)
	}
	go func() { _ = cmd.Wait() }()

	s.logger.InfoContext(ctx, "editor launched",
		slog.String("id", sess.ID.String()),
		slog.String("editor", parts[0]),
		slog.String("dir", workDir),
	)
	return OpenEditorResponse{Command: cmd}, nil
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

func (s *SessionService) ensureTmuxSession(ctx context.Context, tmuxID string, sess domain.Session, shellCommand string) error {
	_, err := s.tmux.GetSession(ctx, tmuxID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrTmuxSessionNotFound) {
		return fmt.Errorf("inspect tmux session: %w", err)
	}

	startDir, err := sessionStartDir(sess)
	if err != nil {
		return err
	}
	if _, err := s.tmux.CreateSession(ctx, tmuxID, startDir, shellCommand); err != nil {
		return fmt.Errorf("recreate tmux session: %w", err)
	}
	s.logger.InfoContext(ctx, "tmux session recreated",
		slog.String("id", tmuxID),
	)
	return nil
}

// sessionStartDir returns the working directory a session's tmux session
// opens in — always the session's worktree, since every session is now
// project-backed.
func sessionStartDir(sess domain.Session) (string, error) {
	if !sess.HasWorktree() {
		return "", domain.ErrSessionWorktreeFieldsMismatch
	}
	return sess.WorktreePath, nil
}
