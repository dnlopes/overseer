package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/dashboard"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/adapters/secondary/agent"
	"github.com/dnlopes/overseer/internal/adapters/secondary/git"
	"github.com/dnlopes/overseer/internal/adapters/secondary/storage"
	"github.com/dnlopes/overseer/internal/adapters/secondary/tmux"
	"github.com/dnlopes/overseer/internal/core/service"
	"github.com/dnlopes/overseer/internal/shared/config"
	"github.com/dnlopes/overseer/internal/shared/logger"
	"github.com/dnlopes/overseer/internal/shared/paths"
)

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

	tmuxStub := &tmux.Stub{}
	gitStub := &git.Stub{}
	agentStub := &agent.Stub{}
	_ = agentStub

	sessionSvc := service.NewSessionService(store, tmuxStub, gitStub, log)

	s := styles.New()
	registry := dashboard.NewHelpRegistry()
	dash := dashboard.New(s, sessionSvc, registry)
	p := tea.NewProgram(altScreenModel{inner: dash})

	if _, err := p.Run(); err != nil {
		log.Error("run tui", "error", err)
		os.Exit(1)
	}
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
