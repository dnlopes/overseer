package session

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/testutil"
)

var errAdapter = errors.New("adapter blew up")

func escKeyPress() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEsc}
}

func makeTmuxSession(id string) domain.TmuxSession {
	now := time.Now()
	return domain.TmuxSession{ID: id, CreatedAt: now, UpdatedAt: now}
}

func TestDeleteForm_EscapeCancelsPopup(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	form := NewDeleteForm(styles.New(), newCreateFormSessionService(t), sess)

	_, cmd := form.Update(escKeyPress())

	if cmd == nil {
		t.Fatalf("Update(esc) command = nil, want cancel emit")
	}
	if _, ok := cmd().(shared.NewSessionDeletePopupCloseMsg); !ok {
		t.Fatalf("Update(esc) msg type = %T, want shared.NewSessionDeletePopupCloseMsg", cmd())
	}
}

func TestDeleteForm_NCancelsPopup(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.New())
	form := NewDeleteForm(styles.New(), newCreateFormSessionService(t), sess)

	_, cmd := form.Update(formKeyPress("n"))

	if cmd == nil {
		t.Fatalf("Update(n) command = nil, want cancel emit")
	}
	if _, ok := cmd().(shared.NewSessionDeletePopupCloseMsg); !ok {
		t.Fatalf("Update(n) msg type = %T, want shared.NewSessionDeletePopupCloseMsg", cmd())
	}
}

func TestDeleteForm_EnterAlsoConfirms(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.Nil)
	svc, repo, _, tmux, _ := newCreateFormSessionServiceWithMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(makeTmuxSession(sess.ID.String()), nil).Once()
	tmux.EXPECT().KillSession(mock.Anything, sess.ID.String()).Return(nil).Once()
	repo.EXPECT().Delete(mock.Anything, sess.ID).Return(nil).Once()

	form := NewDeleteForm(styles.New(), svc, sess)

	_, cmd := form.Update(formKeyPress("enter"))

	if cmd == nil {
		t.Fatalf("Update(enter) command = nil, want delete command")
	}
	if _, ok := cmd().(shared.SessionDeletedMsg); !ok {
		t.Fatalf("Update(enter) msg type = %T, want shared.SessionDeletedMsg", cmd())
	}
}

func TestDeleteForm_ViewUsesDangerStyling(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.Nil)
	form := NewDeleteForm(styles.New(), newCreateFormSessionService(t), sess)

	view := form.View().Content
	wantANSI := "\x1b[38;2;239;68;68m"
	if !strings.Contains(view, wantANSI) {
		t.Fatalf("View() missing danger-red ANSI sequence %q in output: %q", wantANSI, view)
	}
}

func TestDeleteForm_YConfirmCallsServiceAndEmitsDeletedMsg(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.Nil)
	svc, repo, _, tmux, _ := newCreateFormSessionServiceWithMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, sess.ID.String()).
		Return(makeTmuxSession(sess.ID.String()), nil).Once()
	tmux.EXPECT().KillSession(mock.Anything, sess.ID.String()).Return(nil).Once()
	repo.EXPECT().Delete(mock.Anything, sess.ID).Return(nil).Once()

	form := NewDeleteForm(styles.New(), svc, sess)

	_, cmd := form.Update(formKeyPress("y"))

	if cmd == nil {
		t.Fatalf("Update(y) command = nil, want delete command")
	}
	msg := cmd()
	if _, ok := msg.(shared.SessionDeletedMsg); !ok {
		t.Fatalf("Update(y) msg type = %T, want shared.SessionDeletedMsg", msg)
	}
}

func TestDeleteForm_YConfirmServiceErrorEmitsErrMsg(t *testing.T) {
	sess := testutil.MakeSession("alpha", uuid.Nil)
	svc, repo, _, _, _ := newCreateFormSessionServiceWithMocks(t)
	repo.EXPECT().Get(mock.Anything, sess.ID).Return(sess, errAdapter).Once()

	form := NewDeleteForm(styles.New(), svc, sess)

	_, cmd := form.Update(formKeyPress("y"))

	if cmd == nil {
		t.Fatalf("Update(y) command = nil, want error command")
	}
	errMsg, ok := cmd().(shared.SessionDeleteErrMsg)
	if !ok {
		t.Fatalf("Update(y) msg type = %T, want shared.SessionDeleteErrMsg", cmd())
	}
	if errMsg.Err == nil {
		t.Fatalf("Update(y) errMsg.Err = nil, want non-nil")
	}
}

func TestDeleteForm_ViewMentionsSessionNameAndWorktreeConsequences(t *testing.T) {
	sess := testutil.MakeSessionWithWorktree("alpha", uuid.New(), "/data/worktrees/abc", "main", "overseer/alpha")
	form := NewDeleteForm(styles.New(), newCreateFormSessionService(t), sess)

	view := form.View().Content
	if !strings.Contains(view, "alpha") {
		t.Fatalf("View() missing session name 'alpha': %q", view)
	}
	if !strings.Contains(view, "worktree") {
		t.Fatalf("View() missing worktree warning for project-backed session: %q", view)
	}
}

func TestDeleteForm_ViewMentionsTmuxConsequencesForProjectlessSession(t *testing.T) {
	sess := testutil.MakeSession("orphan", uuid.Nil)
	form := NewDeleteForm(styles.New(), newCreateFormSessionService(t), sess)

	view := form.View().Content
	if strings.Contains(view, "worktree") {
		t.Fatalf("View() mentioned worktree for project-less session: %q", view)
	}
	if !strings.Contains(view, "tmux") {
		t.Fatalf("View() missing tmux warning: %q", view)
	}
}
