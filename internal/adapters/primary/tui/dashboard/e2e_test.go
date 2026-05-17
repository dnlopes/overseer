package dashboard

import (
	"context"
	"io"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	teatestv2 "github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
	"github.com/dnlopes/overseer/internal/testutil"
)

// inMemoryRepo is a thread-safe stateful repository. Unlike MockSessionRepository,
// it persists saved sessions across List/Get calls, enabling full CRUD flows.
type inMemoryRepo struct {
	mu       sync.Mutex
	sessions map[uuid.UUID]domain.Session
}

func newInMemoryRepo(seeds ...domain.Session) *inMemoryRepo {
	r := &inMemoryRepo{sessions: make(map[uuid.UUID]domain.Session)}
	for _, s := range seeds {
		r.sessions[s.ID] = s
	}
	return r
}

func (r *inMemoryRepo) Save(_ context.Context, s domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.ID] = s
	return nil
}

func (r *inMemoryRepo) Get(_ context.Context, id uuid.UUID) (domain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	return s, nil
}

func (r *inMemoryRepo) List(_ context.Context) ([]domain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]domain.Session, 0, len(r.sessions))
	for _, s := range r.sessions {
		result = append(result, s)
	}
	return result, nil
}

func (r *inMemoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, id)
	return nil
}

type e2eTmux struct{}

func (e *e2eTmux) CreateSession(_ context.Context, _ string) (string, error) { return "stub", nil }
func (e *e2eTmux) KillSession(_ context.Context, _ string) error             { return nil }

type e2eGit struct{}

func (e *e2eGit) CreateWorktree(_ context.Context, _, _ string) error { return nil }
func (e *e2eGit) RemoveWorktree(_ context.Context, _ string) error    { return nil }

func e2eLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func buildE2EModel(repo domain.SessionRepository) Model {
	svc := service.NewSessionService(repo, &e2eTmux{}, &e2eGit{}, e2eLogger())
	return New(styles.New(), svc, NewHelpRegistry())
}

func e2eKey(text string) tea.KeyPressMsg {
	switch text {
	case "tab":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})
	case "enter":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	case "return":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyReturn})
	case "line-feed":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	case "ctrl+j":
		return tea.KeyPressMsg(tea.Key{Code: 'j', Mod: tea.ModCtrl})
	case "esc":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc})
	case "ctrl+k":
		return tea.KeyPressMsg(tea.Key{Code: 'k', Mod: tea.ModCtrl})
	case "ctrl+u":
		return tea.KeyPressMsg(tea.Key{Code: 'u', Mod: tea.ModCtrl})
	default:
		return tea.KeyPressMsg(tea.Key{Code: []rune(text)[0], Text: text})
	}
}

func readFinalOutput(t *testing.T, tm *teatestv2.TestModel) []byte {
	t.Helper()
	r := tm.FinalOutput(t, teatestv2.WithFinalTimeout(5*time.Second))
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read final output: %v", err)
	}
	return out
}

func TestE2E_CreateFlow(t *testing.T) {
	testutil.Setup(t)

	repo := newInMemoryRepo()
	m := buildE2EModel(repo)
	tm := testutil.NewHarness(t, m, 80, 24)

	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("n"))
	time.Sleep(100 * time.Millisecond)

	tm.Type("my-session")
	tm.Send(e2eKey("tab"))
	tm.Type("overseer")
	tm.Send(e2eKey("enter"))
	time.Sleep(200 * time.Millisecond)

	tm.Send(e2eKey("q"))
	_ = readFinalOutput(t, tm)
	sessions, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after create, got %d", len(sessions))
	}
	if sessions[0].Name != "my-session" {
		t.Fatalf("expected session name 'my-session', got %q", sessions[0].Name)
	}
}

func TestE2E_CreateFlowWithReturnKey(t *testing.T) {
	testutil.Setup(t)

	repo := newInMemoryRepo()
	m := buildE2EModel(repo)
	tm := testutil.NewHarness(t, m, 80, 24)

	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("n"))
	time.Sleep(100 * time.Millisecond)

	tm.Type("my-session")
	tm.Send(e2eKey("tab"))
	tm.Type("overseer")
	tm.Send(e2eKey("return"))
	time.Sleep(200 * time.Millisecond)

	tm.Send(e2eKey("q"))
	_ = readFinalOutput(t, tm)
	sessions, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after create, got %d", len(sessions))
	}
	if sessions[0].Name != "my-session" {
		t.Fatalf("expected session name 'my-session', got %q", sessions[0].Name)
	}
}

func TestE2E_CreateFlowEmptyProjectShowsErrorWithLineFeed(t *testing.T) {
	testutil.Setup(t)

	repo := newInMemoryRepo()
	m := buildE2EModel(repo)
	tm := testutil.NewHarness(t, m, 80, 24)

	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("n"))
	time.Sleep(100 * time.Millisecond)

	tm.Type("my-session")
	tm.Send(e2eKey("line-feed"))
	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("esc"))
	tm.Send(e2eKey("q"))
	out := string(readFinalOutput(t, tm))
	if !strings.Contains(out, "project name is required") {
		t.Fatalf("expected project validation error, got:\n%s", out)
	}
	sessions, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected no sessions after invalid create, got %d", len(sessions))
	}
}

func TestE2E_CreateFlowEmptyNameShowsErrorWithCtrlJ(t *testing.T) {
	testutil.Setup(t)

	repo := newInMemoryRepo()
	m := buildE2EModel(repo)
	tm := testutil.NewHarness(t, m, 80, 24)

	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("n"))
	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("ctrl+j"))
	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("esc"))
	tm.Send(e2eKey("q"))
	out := string(readFinalOutput(t, tm))
	if !strings.Contains(out, "session name is required") {
		t.Fatalf("expected session validation error, got:\n%s", out)
	}
	sessions, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected no sessions after invalid create, got %d", len(sessions))
	}
}

func TestE2E_RenameFlow(t *testing.T) {
	testutil.Setup(t)

	seed := testutil.MakeSession("original", "proj")
	repo := newInMemoryRepo(seed)
	m := buildE2EModel(repo)
	tm := testutil.NewHarness(t, m, 80, 24)

	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("r"))
	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("ctrl+u"))
	tm.Type("renamed-session")
	tm.Send(e2eKey("enter"))
	time.Sleep(200 * time.Millisecond)

	tm.Send(e2eKey("q"))
	_ = readFinalOutput(t, tm)
	sessions, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after rename, got %d", len(sessions))
	}
	if sessions[0].Name != "renamed-session" {
		t.Fatalf("expected 'renamed-session', got %q", sessions[0].Name)
	}
}

func TestE2E_ReorderFlow(t *testing.T) {
	testutil.Setup(t)

	s1 := testutil.MakeSession("sess-1", "myproject")
	s1.Order = 1
	s2 := testutil.MakeSession("sess-2", "myproject")
	s2.Order = 2
	s3 := testutil.MakeSession("sess-3", "myproject")
	s3.Order = 3
	repo := newInMemoryRepo(s1, s2, s3)
	m := buildE2EModel(repo)
	tm := testutil.NewHarness(t, m, 80, 24)

	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("J"))

	// Wait for the async reorder + list-reload goroutine; in-memory repo finishes in <1 ms.
	time.Sleep(300 * time.Millisecond)

	tm.Send(e2eKey("q"))
	out := readFinalOutput(t, tm)

	finalM := tm.FinalModel(t, teatestv2.WithFinalTimeout(5*time.Second)).(Model)
	selected, ok := finalM.sessionsList.SelectedSession()
	if !ok {
		t.Fatal("expected a selected session after reorder")
	}
	if selected.Name != "sess-1" {
		t.Fatalf("expected selection to follow sess-1 after reorder down, got %q", selected.Name)
	}
	if finalM.sessionsList.Cursor() != 1 {
		t.Fatalf("expected cursor to follow reordered session at index 1, got %d", finalM.sessionsList.Cursor())
	}

	sessions, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].Order < sessions[j].Order })
	if got := []string{sessions[0].Name, sessions[1].Name, sessions[2].Name}; got[0] != "sess-2" || got[1] != "sess-1" || got[2] != "sess-3" {
		t.Fatalf("expected persisted order [sess-2 sess-1 sess-3], got %v", got)
	}
	_ = out
}

func TestE2E_FocusCycling(t *testing.T) {
	testutil.Setup(t)

	repo := newInMemoryRepo()
	m := buildE2EModel(repo)
	tm := testutil.NewHarness(t, m, 80, 24)

	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("tab"))
	time.Sleep(50 * time.Millisecond)
	tm.Send(e2eKey("tab"))
	time.Sleep(50 * time.Millisecond)

	tm.Send(e2eKey("q"))
	out := readFinalOutput(t, tm)

	finalM := tm.FinalModel(t, teatestv2.WithFinalTimeout(5*time.Second)).(Model)
	if finalM.activePane != PaneSessions {
		t.Fatalf("expected PaneSessions after Tab cycle, got %d", finalM.activePane)
	}
	_ = out
}

func TestE2E_QuitCleanly(t *testing.T) {
	testutil.Setup(t)

	repo := newInMemoryRepo()
	m := buildE2EModel(repo)
	tm := testutil.NewHarness(t, m, 80, 24)

	time.Sleep(100 * time.Millisecond)

	tm.Send(e2eKey("q"))
	_ = readFinalOutput(t, tm)
}
