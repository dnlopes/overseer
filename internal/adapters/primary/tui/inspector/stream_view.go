package inspector

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/service"
)

// streamView previews a tmux pane that streams content in real time (Agent or
// Shell). Behaviour is identical between the two; only the targeted tmux
// session and the placeholder strings differ.
type streamView struct {
	kind            viewKind
	label           string
	previewKind     service.PreviewKind
	notReadyMessage string

	sessionID     uuid.UUID
	width, height int
	content       string
	ready         bool
	err           error
	service       service.SessionService
	styles        *styles.Styles
	pollInterval  time.Duration
}

func newAgentView(svc service.SessionService, s *styles.Styles, pollInterval time.Duration) *streamView {
	return &streamView{
		kind:            viewKindAgent,
		label:           "Agent",
		previewKind:     service.PreviewKindAgent,
		notReadyMessage: "Agent not started — press ⏎ to launch",
		service:         svc,
		styles:          s,
		pollInterval:    pollInterval,
	}
}

func newShellView(svc service.SessionService, s *styles.Styles, pollInterval time.Duration) *streamView {
	return &streamView{
		kind:            viewKindShell,
		label:           "Shell",
		previewKind:     service.PreviewKindShell,
		notReadyMessage: "Shell session unavailable",
		service:         svc,
		styles:          s,
		pollInterval:    pollInterval,
	}
}

func (v *streamView) Label() string { return v.label }

func (v *streamView) Init() tea.Cmd {
	return v.capture()
}

func (v *streamView) Update(msg tea.Msg) (View, tea.Cmd) {
	captured, ok := msg.(previewCapturedMsg)
	if !ok {
		return v, nil
	}
	if captured.kind != v.kind {
		return v, nil
	}
	if captured.sessionID != v.sessionID {
		return v, nil
	}
	if v.sessionID == uuid.Nil {
		return v, nil
	}
	if captured.err != nil {
		v.err = captured.err
	} else {
		v.content = captured.content
		v.ready = captured.sessionReady
		v.err = nil
	}
	return v, v.scheduleNext()
}

func (v *streamView) Body() string {
	if v.sessionID == uuid.Nil {
		return components.CenteredContent(v.styles, v.styles.EmptyState.Title.Render("Select a session to preview"), v.width, v.height)
	}
	if v.err != nil {
		return components.CenteredContent(v.styles, v.styles.EmptyState.Title.Render("Preview error: "+v.err.Error()), v.width, v.height)
	}
	if !v.ready {
		return components.CenteredContent(v.styles, v.styles.EmptyState.Title.Render(v.notReadyMessage), v.width, v.height)
	}
	return v.content
}

func (v *streamView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

func (v *streamView) SetSession(sessionID uuid.UUID) {
	v.sessionID = sessionID
	v.content = ""
	v.ready = false
	v.err = nil
}

// capture builds a Cmd that fetches the current pane content for the view's
// sessionID. The sessionID, width, and height are captured at Cmd creation
// time so messages produced after a session change or resize are dropped by
// the staleness check in Update, terminating the old polling chain. Width
// and height are forwarded so the service can resize the tmux pane before
// capturing, forcing the agent app to redraw at the preview's canvas size.
func (v *streamView) capture() tea.Cmd {
	sessID := v.sessionID
	svc := v.service
	kind := v.kind
	previewKind := v.previewKind
	width := v.width
	height := v.height
	return func() tea.Msg {
		if sessID == uuid.Nil {
			return previewCapturedMsg{kind: kind, sessionID: sessID}
		}
		resp, err := svc.PreviewSession(context.Background(), service.PreviewSessionRequest{
			ID:     sessID,
			Kind:   previewKind,
			Width:  width,
			Height: height,
		})
		return previewCapturedMsg{
			kind:         kind,
			sessionID:    sessID,
			content:      resp.Content,
			sessionReady: resp.SessionReady,
			err:          err,
		}
	}
}

func (v *streamView) scheduleNext() tea.Cmd {
	next := v.capture()
	return tea.Tick(v.pollInterval, func(time.Time) tea.Msg {
		return next()
	})
}
