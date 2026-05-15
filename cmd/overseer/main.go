package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/dashboard"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/help"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	agentstub "github.com/dnlopes/overseer/internal/adapters/secondary/agent/stub"
	yamlconfig "github.com/dnlopes/overseer/internal/adapters/secondary/config/yaml"
	gitstub "github.com/dnlopes/overseer/internal/adapters/secondary/git/stub"
	slogadapter "github.com/dnlopes/overseer/internal/adapters/secondary/logger/slog"
	jsonstorage "github.com/dnlopes/overseer/internal/adapters/secondary/storage/json"
	tmuxstub "github.com/dnlopes/overseer/internal/adapters/secondary/tmux/stub"
	servicesession "github.com/dnlopes/overseer/internal/core/service/session"
	"github.com/dnlopes/overseer/internal/shared/paths"
)

func main() {
	cfg, err := yamlconfig.Load(paths.ConfigFile())
	if err != nil {
		fmt.Fprintf(os.Stderr, "overseer: load config: %v\n", err)
		os.Exit(1)
	}

	logger, logCloser, err := slogadapter.New(cfg.Logging.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "overseer: initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logCloser.Close()

	store, err := jsonstorage.New(paths.DataFile(), logger)
	if err != nil {
		logger.Error("initialize storage", "error", err)
		os.Exit(1)
	}

	tmuxStub := &tmuxstub.Stub{}
	gitStub := &gitstub.Stub{}
	agentStub := &agentstub.Stub{}
	_ = agentStub

	createUC := servicesession.NewCreateUseCase(store, tmuxStub, gitStub, logger)
	renameUC := servicesession.NewRenameUseCase(store, logger)
	reorderUC := servicesession.NewReorderUseCase(store, logger)
	listUC := servicesession.NewListUseCase(store)

	s := styles.New()
	registry := help.NewRegistry()
	dash := dashboard.New(s, createUC, renameUC, reorderUC, listUC, registry)
	p := tea.NewProgram(altScreenModel{inner: dash})

	if _, err := p.Run(); err != nil {
		logger.Error("run tui", "error", err)
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
