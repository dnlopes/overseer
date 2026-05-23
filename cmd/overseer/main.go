package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/dashboard"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/jobs"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/adapters/secondary/agentstatus/claudecode"
	agentstatusregistry "github.com/dnlopes/overseer/internal/adapters/secondary/agentstatus/registry"
	"github.com/dnlopes/overseer/internal/adapters/secondary/git"
	githubcli "github.com/dnlopes/overseer/internal/adapters/secondary/github"
	"github.com/dnlopes/overseer/internal/adapters/secondary/storage"
	"github.com/dnlopes/overseer/internal/adapters/secondary/tmux"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
	"github.com/dnlopes/overseer/internal/shared/config"
	"github.com/dnlopes/overseer/internal/shared/logger"
	"github.com/dnlopes/overseer/internal/shared/paths"
)

const pullRequestRefreshInterval = time.Minute

func main() {
	cfg, err := config.Load(paths.ConfigFile())
	if err != nil {
		fmt.Fprintf(os.Stderr, "overseer: load config: %v\n", err)
		os.Exit(1)
	}

	resolver := paths.NewResolver(cfg.Storage.DataDir)

	log, logCloser, err := logger.New(resolver.LogFile(), cfg.Logging.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "overseer: initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logCloser.Close()

	launchers, err := cfg.DomainLaunchers()
	if err != nil {
		log.Error("resolve launchers", "error", err)
		os.Exit(1)
	}
	var defaultLauncher domain.Launcher
	if len(launchers) > 0 {
		defaultLauncher = launchers[0]
	}

	editors, err := cfg.DomainEditors()
	if err != nil {
		log.Error("resolve editors", "error", err)
		os.Exit(1)
	}
	var defaultEditor domain.Editor
	if len(editors) > 0 {
		defaultEditor = editors[0]
	}

	labels, err := cfg.DomainLabels()
	if err != nil {
		log.Error("resolve labels", "error", err)
		os.Exit(1)
	}

	store, err := storage.New(resolver.DataFile(), log)
	if err != nil {
		log.Error("initialize storage", "error", err)
		os.Exit(1)
	}

	tmuxAdapter, err := tmux.New(log)
	if err != nil {
		log.Error("initialize tmux", "error", err)
		os.Exit(1)
	}

	gitAdapter, err := git.New(log)
	if err != nil {
		log.Error("initialize git", "error", err)
		os.Exit(1)
	}

	githubAdapter := githubcli.New(log)

	sessionSvc := service.NewSessionService(store.Sessions(), store.Projects(), tmuxAdapter, gitAdapter, resolver, defaultLauncher, defaultEditor, log)
	projectSvc := service.NewProjectService(store.Projects(), gitAdapter, log)
	prSvc := service.NewPullRequestService(githubAdapter, log)

	agentStatusRegistry := agentstatusregistry.New()
	agentStatusRegistry.Register(claudecode.NewPaneDetector(tmuxAdapter))
	agentStatusSvc := service.NewAgentStatusService(store.Sessions(), tmuxAdapter, agentStatusRegistry, log)

	prJob := buildPullRequestJob(sessionSvc, projectSvc, prSvc)
	schedulerJobs := []jobs.Job{prJob}
	if cfg.AgentStatus.Enabled {
		schedulerJobs = append(schedulerJobs, buildAgentStatusDebugJob(agentStatusSvc, cfg.AgentStatus.RefreshInterval, log))
	}
	scheduler := jobs.New(schedulerJobs...)

	previewRefresh, err := cfg.PreviewRefreshDuration()
	if err != nil {
		log.Error("resolve preview refresh interval", "error", err)
		os.Exit(1)
	}

	s := styles.NewWithTheme(cfg.Theme, cfg.DisableEmoji)
	dash := dashboard.New(s, *sessionSvc, *projectSvc, scheduler, launchers, editors, labels, cfg.Dashboard.MinWidth, cfg.Dashboard.MinHeight, previewRefresh)
	p := tea.NewProgram(altScreenModel{inner: dash})

	if _, err := p.Run(); err != nil {
		log.Error("run tui", "error", err)
		os.Exit(1)
	}
}

func buildPullRequestJob(
	sessionSvc *service.SessionService,
	projectSvc *service.ProjectService,
	prSvc *service.PullRequestService,
) jobs.Job {
	return jobs.Job{
		ID:       "pr-status-refresh",
		Interval: pullRequestRefreshInterval,
		Run: func() tea.Cmd {
			return shared.Request(
				func(ctx context.Context) (pollData, error) {
					sessionsResp, err := sessionSvc.List(ctx, service.ListSessionsRequest{})
					if err != nil {
						return pollData{}, fmt.Errorf("list sessions: %w", err)
					}
					projectsResp, err := projectSvc.List(ctx, service.ListProjectsRequest{})
					if err != nil {
						return pollData{}, fmt.Errorf("list projects: %w", err)
					}
					return pollData{Sessions: sessionsResp.Sessions, Projects: projectsResp.Projects}, nil
				},
				func(data pollData, err error) tea.Msg {
					if err != nil {
						return nil
					}
					return shared.JobsBatchMsg{Cmds: fanOutPRFetches(prSvc, data)}
				},
			)
		},
	}
}

// buildAgentStatusDebugJob returns a scheduler job that calls
// AgentStatusService.PollAll on every tick and logs each session's
// status at INFO. This is the Slice-2 deliverable: status detection is
// proven end-to-end via the log, with no TUI wiring yet. Slice 3
// replaces this job with one that emits shared.AgentStatusesUpdatedMsg
// into the Bubble Tea event loop for rendering.
func buildAgentStatusDebugJob(svc *service.AgentStatusService, interval time.Duration, log *slog.Logger) jobs.Job {
	return jobs.Job{
		ID:       "agent-status-debug",
		Interval: interval,
		Run: func() tea.Cmd {
			return shared.Request(
				func(ctx context.Context) (service.PollAllAgentStatusesResponse, error) {
					return svc.PollAll(ctx, service.PollAllAgentStatusesRequest{})
				},
				func(resp service.PollAllAgentStatusesResponse, err error) tea.Msg {
					if err != nil {
						log.Warn("agent-status-debug poll failed", "error", err)
						return nil
					}
					for id, st := range resp.Statuses {
						log.Info("agent-status-debug",
							"session_id", id.String(),
							"kind", string(st.Kind),
							"source", st.Source,
							"reason", st.Reason,
						)
					}
					return nil
				},
			)
		},
	}
}

type pollData struct {
	Sessions []domain.Session
	Projects []domain.Project
}

func fanOutPRFetches(prSvc *service.PullRequestService, data pollData) []tea.Cmd {
	projectByID := make(map[uuid.UUID]domain.Project, len(data.Projects))
	for _, p := range data.Projects {
		projectByID[p.ID] = p
	}
	cmds := make([]tea.Cmd, 0, len(data.Sessions))
	for _, sess := range data.Sessions {
		project, ok := projectByID[sess.ProjectID]
		if !ok {
			continue
		}
		if !sess.HasWorktree() {
			continue
		}
		sid, branch, repoPath := sess.ID, sess.Branch, project.Path
		cmds = append(cmds, shared.Request(
			func(ctx context.Context) (domain.PullRequest, error) {
				resp, err := prSvc.GetForBranch(ctx, service.GetPullRequestForBranchRequest{
					RepoPath: repoPath,
					Branch:   branch,
				})
				return resp.PullRequest, err
			},
			func(pr domain.PullRequest, err error) tea.Msg {
				return shared.PRStatusUpdatedMsg{SessionID: sid, PR: pr, Err: err}
			},
		))
	}
	return cmds
}

type altScreenModel struct {
	inner tea.Model
}

func (m altScreenModel) Init() tea.Cmd {
	return m.inner.Init()
}

func (m altScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	inner, cmd := m.inner.Update(msg)
	m.inner = inner
	return m, cmd
}

func (m altScreenModel) View() tea.View {
	v := m.inner.View()
	v.AltScreen = true
	return v
}
