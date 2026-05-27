package session

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/testutil"
)

func TestKillPreviewForm_EscapeCancelsPopup(t *testing.T) {
	svc := newCreateFormSessionService(t)
	form := NewKillPreviewForm(styles.New(), svc, uuid.New(), "alpha", "Shell")

	_, cmd := form.Update(escKeyPress())

	if cmd == nil {
		t.Fatalf("Update(esc) command = nil, want cancel emit")
	}
	if _, ok := cmd().(shared.KillPreviewPopupCloseMsg); !ok {
		t.Fatalf("Update(esc) msg type = %T, want shared.KillPreviewPopupCloseMsg", cmd())
	}
}

func TestKillPreviewForm_NCancelsPopup(t *testing.T) {
	svc := newCreateFormSessionService(t)
	form := NewKillPreviewForm(styles.New(), svc, uuid.New(), "alpha", "Shell")

	_, cmd := form.Update(formKeyPress("n"))

	if cmd == nil {
		t.Fatalf("Update(n) command = nil, want cancel emit")
	}
	if _, ok := cmd().(shared.KillPreviewPopupCloseMsg); !ok {
		t.Fatalf("Update(n) msg type = %T, want shared.KillPreviewPopupCloseMsg", cmd())
	}
}

func TestKillPreviewForm_EnterConfirmsKill_Shell(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	svc, repo, _, tmux, _ := newCreateFormSessionServiceWithMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(makeTmuxSession(sess.ID.String()), nil).Once()
	tmux.EXPECT().KillSession(mock.Anything, sess.ID.String()).Return(nil).Once()

	form := NewKillPreviewForm(styles.New(), svc, sess.ID, sess.Name, "Shell")

	_, cmd := form.Update(formKeyPress("enter"))

	if cmd == nil {
		t.Fatalf("Update(enter) command = nil, want kill command")
	}
	if _, ok := cmd().(shared.PreviewSessionKilledMsg); !ok {
		t.Fatalf("Update(enter) msg type = %T, want shared.PreviewSessionKilledMsg", cmd())
	}
}

func TestKillPreviewForm_EnterConfirmsKill_Agent(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	svc, repo, _, tmux, _ := newCreateFormSessionServiceWithMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()+"-agent").
		Return(makeTmuxSession(sess.ID.String()+"-agent"), nil).Once()
	tmux.EXPECT().KillSession(mock.Anything, sess.ID.String()+"-agent").Return(nil).Once()

	form := NewKillPreviewForm(styles.New(), svc, sess.ID, sess.Name, "Agent")

	_, cmd := form.Update(formKeyPress("enter"))

	if cmd == nil {
		t.Fatalf("Update(enter) command = nil, want kill command")
	}
	if _, ok := cmd().(shared.PreviewSessionKilledMsg); !ok {
		t.Fatalf("Update(enter) msg type = %T, want shared.PreviewSessionKilledMsg", cmd())
	}
}

func TestKillPreviewForm_ViewUsesDangerStyling(t *testing.T) {
	svc := newCreateFormSessionService(t)
	form := NewKillPreviewForm(styles.New(), svc, uuid.New(), "alpha", "Shell")

	view := form.View().Content

	if !strings.Contains(view, "Kill preview session") {
		t.Errorf("View() missing title, got %q", view)
	}
	if !strings.Contains(view, "alpha") {
		t.Errorf("View() missing session name, got %q", view)
	}
	if !strings.Contains(view, "Shell") {
		t.Errorf("View() missing preview kind, got %q", view)
	}
}
