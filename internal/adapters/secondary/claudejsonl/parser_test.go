package claudejsonl_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/secondary/claudejsonl"
	"github.com/dnlopes/overseer/internal/adapters/secondary/lsof"
	"github.com/dnlopes/overseer/internal/core/domain"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

type fakeTmux struct {
	mu       sync.Mutex
	pidByID  map[string]int
	errByID  map[string]error
	defaultE error
	calls    int
}

func (f *fakeTmux) GetPanePID(_ context.Context, tmuxID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if err, ok := f.errByID[tmuxID]; ok {
		return 0, err
	}
	if f.defaultE != nil {
		return 0, f.defaultE
	}
	if pid, ok := f.pidByID[tmuxID]; ok {
		return pid, nil
	}
	return 12345, nil
}

type fakeLsof struct {
	mu      sync.Mutex
	queue   []lsofReply
	defReply lsofReply
}

type lsofReply struct {
	res lsof.Resolution
	err error
}

func (f *fakeLsof) ResolveAgentSession(_ context.Context, _ int) (lsof.Resolution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.queue) > 0 {
		r := f.queue[0]
		f.queue = f.queue[1:]
		return r.res, r.err
	}
	return f.defReply.res, f.defReply.err
}

func newRef(t *testing.T) domain.AgentSessionRef {
	t.Helper()
	id := uuid.New()
	return domain.AgentSessionRef{
		SessionID:          id,
		AgentTmuxSessionID: id.String() + "-agent",
		WorktreePath:       "/tmp/worktree",
	}
}

func writeJSONL(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// nowFunc cannot be set externally — exported test helpers are not part of the
// production API. We instead time-travel the JSONL fixture: every test that
// asserts a non-Idle activity writes a timestamp close to time.Now().
func nowMinus(d time.Duration) string {
	return time.Now().Add(-d).UTC().Format(time.RFC3339Nano)
}

func TestParser_Observe_TmuxSessionMissing_ReturnsErrAgentNotRunning(t *testing.T) {
	t.Parallel()
	tmux := &fakeTmux{defaultE: domain.ErrTmuxSessionNotFound}
	parser := claudejsonl.New(tmux, &fakeLsof{}, discardLogger())

	_, err := parser.Observe(context.Background(), newRef(t))

	if !errors.Is(err, domain.ErrAgentNotRunning) {
		t.Fatalf("err = %v, want domain.ErrAgentNotRunning", err)
	}
}

func TestParser_Observe_TmuxInfraError_WrappedReturn(t *testing.T) {
	t.Parallel()
	infraErr := errors.New("tmux crashed")
	tmux := &fakeTmux{defaultE: infraErr}
	parser := claudejsonl.New(tmux, &fakeLsof{}, discardLogger())

	_, err := parser.Observe(context.Background(), newRef(t))

	if !errors.Is(err, infraErr) {
		t.Fatalf("err = %v, want chain to contain %v", err, infraErr)
	}
	if !strings.Contains(err.Error(), "resolve tmux pane pid") {
		t.Fatalf("err = %v, want wrap with context", err)
	}
}

func TestParser_Observe_LsofNoDescendantsNoCache_ReturnsErrAgentNotRunning(t *testing.T) {
	t.Parallel()
	tmux := &fakeTmux{}
	ls := &fakeLsof{defReply: lsofReply{err: lsof.ErrNoDescendants}}
	parser := claudejsonl.New(tmux, ls, discardLogger())

	_, err := parser.Observe(context.Background(), newRef(t))

	if !errors.Is(err, domain.ErrAgentNotRunning) {
		t.Fatalf("err = %v, want domain.ErrAgentNotRunning", err)
	}
}

func TestParser_Observe_LsofNoStoreNoCache_ReturnsErrAgentStoreNotResolved(t *testing.T) {
	t.Parallel()
	tmux := &fakeTmux{}
	ls := &fakeLsof{defReply: lsofReply{err: lsof.ErrNoAgentStore}}
	parser := claudejsonl.New(tmux, ls, discardLogger())

	_, err := parser.Observe(context.Background(), newRef(t))

	if !errors.Is(err, domain.ErrAgentStoreNotResolved) {
		t.Fatalf("err = %v, want domain.ErrAgentStoreNotResolved", err)
	}
}

func TestParser_Observe_LsofInfraError_WrappedReturn(t *testing.T) {
	t.Parallel()
	infraErr := errors.New("lsof exploded")
	tmux := &fakeTmux{}
	ls := &fakeLsof{defReply: lsofReply{err: infraErr}}
	parser := claudejsonl.New(tmux, ls, discardLogger())

	_, err := parser.Observe(context.Background(), newRef(t))

	if !errors.Is(err, infraErr) {
		t.Fatalf("err = %v, want chain to contain %v", err, infraErr)
	}
	if errors.Is(err, domain.ErrAgentNotRunning) || errors.Is(err, domain.ErrAgentStoreNotResolved) {
		t.Fatalf("err = %v, want non-sentinel wrap", err)
	}
}

func TestParser_Observe_LsofReturnsOpenCodeKind_ReturnsWrappedError(t *testing.T) {
	t.Parallel()
	path := writeJSONL(t, `{"type":"user","timestamp":"`+nowMinus(0)+`","message":{"role":"user","content":"hi"}}`)
	tmux := &fakeTmux{}
	ls := &fakeLsof{defReply: lsofReply{res: lsof.Resolution{Kind: lsof.AgentKindOpenCode, Path: path, PID: 42}}}
	parser := claudejsonl.New(tmux, ls, discardLogger())

	_, err := parser.Observe(context.Background(), newRef(t))

	if err == nil {
		t.Fatal("err = nil, want wrong-kind error")
	}
	if !strings.Contains(err.Error(), "agent kind") {
		t.Fatalf("err = %v, want message about wrong agent kind", err)
	}
}

func TestParser_Observe_LsofHitClaude_ReturnsActivityFromJSONL(t *testing.T) {
	t.Parallel()
	path := writeJSONL(t,
		`{"type":"user","timestamp":"`+nowMinus(20*time.Second)+`","message":{"role":"user","content":"hi"}}`,
		`{"type":"assistant","timestamp":"`+nowMinus(5*time.Second)+`","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{}}]}}`,
	)
	tmux := &fakeTmux{}
	ls := &fakeLsof{defReply: lsofReply{res: lsof.Resolution{Kind: lsof.AgentKindClaude, Path: path, PID: 42}}}
	parser := claudejsonl.New(tmux, ls, discardLogger())

	got, err := parser.Observe(context.Background(), newRef(t))

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.Kind != domain.ActivityReading {
		t.Fatalf("Kind = %v, want ActivityReading", got.Kind)
	}
	if got.Tool != "Read" {
		t.Fatalf("Tool = %q, want Read", got.Tool)
	}
}

func TestParser_Observe_LsofMissesAfterPriorHit_UsesCachedPath(t *testing.T) {
	t.Parallel()
	path := writeJSONL(t,
		`{"type":"assistant","timestamp":"`+nowMinus(5*time.Second)+`","message":{"role":"assistant","content":[{"type":"text","text":"done"}]}}`,
	)
	tmux := &fakeTmux{}
	ls := &fakeLsof{
		queue: []lsofReply{
			{res: lsof.Resolution{Kind: lsof.AgentKindClaude, Path: path, PID: 42}},
		},
		defReply: lsofReply{err: lsof.ErrNoAgentStore},
	}
	parser := claudejsonl.New(tmux, ls, discardLogger())
	ref := newRef(t)

	first, err := parser.Observe(context.Background(), ref)
	if err != nil {
		t.Fatalf("first Observe err = %v", err)
	}
	if first.Kind != domain.ActivityWaiting {
		t.Fatalf("first Kind = %v, want ActivityWaiting", first.Kind)
	}

	second, err := parser.Observe(context.Background(), ref)
	if err != nil {
		t.Fatalf("second Observe err = %v (lsof miss + cache hit should succeed)", err)
	}
	if second.Kind != domain.ActivityWaiting {
		t.Fatalf("second Kind = %v, want ActivityWaiting (cache reused)", second.Kind)
	}
}

func TestParser_Observe_LsofHitDifferentPath_UpdatesCache(t *testing.T) {
	t.Parallel()
	path1 := writeJSONL(t,
		`{"type":"assistant","timestamp":"`+nowMinus(5*time.Second)+`","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{}}]}}`,
	)
	path2 := writeJSONL(t,
		`{"type":"assistant","timestamp":"`+nowMinus(5*time.Second)+`","message":{"role":"assistant","content":[{"type":"tool_use","name":"Write","input":{}}]}}`,
	)
	tmux := &fakeTmux{}
	ls := &fakeLsof{queue: []lsofReply{
		{res: lsof.Resolution{Kind: lsof.AgentKindClaude, Path: path1, PID: 42}},
		{res: lsof.Resolution{Kind: lsof.AgentKindClaude, Path: path2, PID: 42}},
		{err: lsof.ErrNoAgentStore},
	}}
	parser := claudejsonl.New(tmux, ls, discardLogger())
	ref := newRef(t)

	first, err := parser.Observe(context.Background(), ref)
	if err != nil {
		t.Fatalf("first Observe err = %v", err)
	}
	if first.Tool != "Read" {
		t.Fatalf("first Tool = %q, want Read (from path1)", first.Tool)
	}

	second, err := parser.Observe(context.Background(), ref)
	if err != nil {
		t.Fatalf("second Observe err = %v", err)
	}
	if second.Tool != "Write" {
		t.Fatalf("second Tool = %q, want Write (cache updated to path2)", second.Tool)
	}

	third, err := parser.Observe(context.Background(), ref)
	if err != nil {
		t.Fatalf("third Observe err = %v", err)
	}
	if third.Tool != "Write" {
		t.Fatalf("third Tool = %q, want Write (cache holds path2 after lsof miss)", third.Tool)
	}
}

func TestParser_Observe_CachedPathDeleted_InvalidatesAndReturnsErrStoreNotResolved(t *testing.T) {
	t.Parallel()
	path := writeJSONL(t,
		`{"type":"assistant","timestamp":"`+nowMinus(5*time.Second)+`","message":{"role":"assistant","content":[{"type":"text","text":"done"}]}}`,
	)
	tmux := &fakeTmux{}
	ls := &fakeLsof{
		queue: []lsofReply{
			{res: lsof.Resolution{Kind: lsof.AgentKindClaude, Path: path, PID: 42}},
		},
		defReply: lsofReply{err: lsof.ErrNoAgentStore},
	}
	parser := claudejsonl.New(tmux, ls, discardLogger())
	ref := newRef(t)

	if _, err := parser.Observe(context.Background(), ref); err != nil {
		t.Fatalf("first Observe err = %v", err)
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove cached file: %v", err)
	}

	_, err := parser.Observe(context.Background(), ref)
	if !errors.Is(err, domain.ErrAgentStoreNotResolved) {
		t.Fatalf("err = %v, want domain.ErrAgentStoreNotResolved after cached file deleted", err)
	}

	// Cache must have been invalidated: a fresh lsof miss should now return
	// store-not-resolved (caching the missing path would have returned a
	// stat error instead).
	_, err = parser.Observe(context.Background(), ref)
	if !errors.Is(err, domain.ErrAgentStoreNotResolved) {
		t.Fatalf("err after invalidation = %v, want domain.ErrAgentStoreNotResolved", err)
	}
}

func TestParser_Observe_JSONLWithoutTimestamps_FallsBackToFileMtime(t *testing.T) {
	t.Parallel()
	path := writeJSONL(t, `{"type":"system","subtype":"turn_duration","durationMs":3005}`)

	if err := os.Chtimes(path, time.Now().Add(-2*time.Second), time.Now().Add(-2*time.Second)); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	tmux := &fakeTmux{}
	ls := &fakeLsof{defReply: lsofReply{res: lsof.Resolution{Kind: lsof.AgentKindClaude, Path: path, PID: 42}}}
	parser := claudejsonl.New(tmux, ls, discardLogger())

	got, err := parser.Observe(context.Background(), newRef(t))

	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.Kind != domain.ActivityIdle {
		t.Fatalf("Kind = %v, want ActivityIdle (no user/assistant entries; mtime fresh but Status unknown)", got.Kind)
	}
}

func TestParser_Observe_LsofMissCacheStaleEnoughForIdleStatus(t *testing.T) {
	t.Parallel()
	path := writeJSONL(t,
		`{"type":"user","timestamp":"`+nowMinus(2*time.Minute)+`","message":{"role":"user","content":"old"}}`,
		`{"type":"assistant","timestamp":"`+nowMinus(90*time.Second)+`","message":{"role":"assistant","content":[{"type":"text","text":"old reply"}]}}`,
	)
	tmux := &fakeTmux{}
	ls := &fakeLsof{
		queue:    []lsofReply{{res: lsof.Resolution{Kind: lsof.AgentKindClaude, Path: path, PID: 42}}},
		defReply: lsofReply{err: lsof.ErrNoAgentStore},
	}
	parser := claudejsonl.New(tmux, ls, discardLogger())
	ref := newRef(t)

	if _, err := parser.Observe(context.Background(), ref); err != nil {
		t.Fatalf("seed Observe err = %v", err)
	}

	got, err := parser.Observe(context.Background(), ref)
	if err != nil {
		t.Fatalf("Observe err = %v", err)
	}
	if got.Kind != domain.ActivityWaiting && got.Kind != domain.ActivityIdle {
		t.Fatalf("Kind = %v, want ActivityWaiting (within 2m) or ActivityIdle", got.Kind)
	}
}
