package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/dashboard"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/jobs"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
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

	log, logCloser, err := logger.New(cfg.Logging.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "overseer: initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logCloser.Close()

	store, err := storage.New(paths.DataFile(), log)
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

	sessionSvc := service.NewSessionService(store.Sessions(), store.Projects(), tmuxAdapter, gitAdapter, log)
	projectSvc := service.NewProjectService(store.Projects(), gitAdapter, log)
	prSvc := service.NewPullRequestService(githubAdapter, log)

	prJob := buildPullRequestJob(sessionSvc, projectSvc, prSvc)
	scheduler := jobs.New(prJob)

	s := styles.New()
	dash := dashboard.New(s, *sessionSvc, *projectSvc, scheduler)
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
		sid, branch, repoPath := sess.ID, sess.Name, project.Path
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
