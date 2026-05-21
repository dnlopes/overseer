package lsof_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dnlopes/overseer/internal/adapters/secondary/lsof"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newAdapter(t *testing.T) *lsof.Adapter {
	t.Helper()
	a, err := lsof.New(discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return a
}

// forkHolder forks a `sh -c "tail -f $path & wait $!"` subprocess that keeps
// the named file open via the child `tail` process. Returns the shell PID,
// which is the parent of the tail process and a safe rootPID to hand to
// ResolveAgentSession (its only descendant is the tail process we control).
//
// Cleanup is registered via t.Cleanup and kills the entire process group so
// the orphaned tail does not leak. We Setpgid on the shell so the kill
// targets both shell and tail.
func forkHolder(t *testing.T, path string) int {
	t.Helper()
	// `exec tail` would replace the shell with tail (no descendant), which
	// defeats the test setup. We deliberately keep `tail` as a child of the
	// shell so the shell has at least one descendant to discover.
	script := "tail -f " + path + " & wait $!"
	cmd := exec.Command("sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start shell: %v", err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
	})

	// Wait until tail has actually opened the file (otherwise lsof sees nothing).
	waitForFDOpen(t, path, 2*time.Second)
	return cmd.Process.Pid
}

// forkLeaf forks a long-running subprocess with no children (a single `sleep`)
// and returns its PID. Used to test the "no descendants" branch.
func forkLeaf(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	return cmd.Process.Pid
}

// waitForFDOpen polls the system lsof until path appears in some process's
// open-file list, or fails the test after deadline. Used so the forked
// `tail` has actually opened the sentinel file before we run our adapter.
func waitForFDOpen(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	canonical := mustEvalSymlinks(t, path)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, _ := exec.Command("lsof", canonical).Output()
		if len(out) > 0 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to be opened by any process", path)
}

func mustEvalSymlinks(t *testing.T, p string) string {
	t.Helper()
	c, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return c
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

// --- New ---

func TestAdapter_New_ReturnsAdapterWhenLsofPresent(t *testing.T) {
	a := newAdapter(t)
	if a == nil {
		t.Fatal("New() returned nil adapter")
	}
}

// --- ResolveAgentSession: no-descendants branch ---

func TestAdapter_ResolveAgentSession_LeafPID_ReturnsErrNoDescendants(t *testing.T) {
	a := newAdapter(t)
	leaf := forkLeaf(t)

	_, err := a.ResolveAgentSession(context.Background(), leaf)
	if !errors.Is(err, lsof.ErrNoDescendants) {
		t.Fatalf("ResolveAgentSession(leafPID) error = %v, want ErrNoDescendants", err)
	}
}

func TestAdapter_ResolveAgentSession_NonexistentPID_ReturnsErrNoDescendants(t *testing.T) {
	a := newAdapter(t)
	// Pick a PID extremely unlikely to exist. PIDs on macOS are int32; 999999
	// is past the practical wraparound window on a freshly booted machine
	// (default PID_MAX is 99999 on macOS). Even if it does exist, the test
	// simply asserts the function does not crash on missing roots.
	_, err := a.ResolveAgentSession(context.Background(), 999999)
	if !errors.Is(err, lsof.ErrNoDescendants) {
		t.Fatalf("ResolveAgentSession(missingPID) error = %v, want ErrNoDescendants", err)
	}
}

// --- ResolveAgentSession: JSONL / Claude branch ---

func TestAdapter_ResolveAgentSession_DescendantHoldsJSONL_ReturnsClaude(t *testing.T) {
	a := newAdapter(t)
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "session-overseer-test.jsonl")
	writeFile(t, sentinel, "{}\n")
	root := forkHolder(t, sentinel)

	res, err := a.ResolveAgentSession(context.Background(), root)
	if err != nil {
		t.Fatalf("ResolveAgentSession() error = %v, want nil", err)
	}
	if res.Kind != lsof.AgentKindClaude {
		t.Fatalf("Resolution.Kind = %v, want AgentKindClaude", res.Kind)
	}
	wantPath := mustEvalSymlinks(t, sentinel)
	if res.Path != wantPath {
		t.Fatalf("Resolution.Path = %q, want %q", res.Path, wantPath)
	}
	if res.PID <= 0 || res.PID == root {
		t.Fatalf("Resolution.PID = %d, want a descendant of %d (not the root itself)", res.PID, root)
	}
}

// --- ResolveAgentSession: SQLite / OpenCode branch ---

func TestAdapter_ResolveAgentSession_DescendantHoldsDB_ReturnsOpenCode(t *testing.T) {
	a := newAdapter(t)
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "opencode.db")
	writeFile(t, sentinel, "")
	root := forkHolder(t, sentinel)

	res, err := a.ResolveAgentSession(context.Background(), root)
	if err != nil {
		t.Fatalf("ResolveAgentSession() error = %v, want nil", err)
	}
	if res.Kind != lsof.AgentKindOpenCode {
		t.Fatalf("Resolution.Kind = %v, want AgentKindOpenCode", res.Kind)
	}
	wantPath := mustEvalSymlinks(t, sentinel)
	if res.Path != wantPath {
		t.Fatalf("Resolution.Path = %q, want %q", res.Path, wantPath)
	}
}

// --- ResolveAgentSession: ignore SQLite auxiliary files ---

func TestAdapter_ResolveAgentSession_OnlyDBWalOrShm_ReturnsErrNoAgentStore(t *testing.T) {
	a := newAdapter(t)
	dir := t.TempDir()
	// Two siblings: a -wal file and a -shm file. Neither should be classified
	// as an OpenCode session: only the main `*.db` qualifies. We hold both
	// files via two tail processes; the wal/shm extensions must be filtered.
	wal := filepath.Join(dir, "opencode.db-wal")
	shm := filepath.Join(dir, "opencode.db-shm")
	writeFile(t, wal, "")
	writeFile(t, shm, "")
	// One shell wraps a tail on each of the two sentinel files so both stay
	// open simultaneously under a single rootPID.
	script := "tail -f " + wal + " & tail -f " + shm + " & wait"
	cmd := exec.Command("sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start shell: %v", err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
	})
	waitForFDOpen(t, wal, 2*time.Second)

	_, err := a.ResolveAgentSession(context.Background(), cmd.Process.Pid)
	if !errors.Is(err, lsof.ErrNoAgentStore) {
		t.Fatalf("ResolveAgentSession() with only wal/shm error = %v, want ErrNoAgentStore", err)
	}
}

// --- ResolveAgentSession: descendants exist but hold no relevant files ---

func TestAdapter_ResolveAgentSession_DescendantsButNoStore_ReturnsErrNoAgentStore(t *testing.T) {
	a := newAdapter(t)
	dir := t.TempDir()
	// `.txt` is neither `.jsonl` nor `.db`: the resolver must ignore it.
	sentinel := filepath.Join(dir, "irrelevant.txt")
	writeFile(t, sentinel, "")
	root := forkHolder(t, sentinel)

	_, err := a.ResolveAgentSession(context.Background(), root)
	if !errors.Is(err, lsof.ErrNoAgentStore) {
		t.Fatalf("ResolveAgentSession() with .txt holder error = %v, want ErrNoAgentStore", err)
	}
}

// --- ResolveAgentSession: deeper descendants are walked ---

func TestAdapter_ResolveAgentSession_GrandchildHoldsJSONL_ReturnsClaude(t *testing.T) {
	a := newAdapter(t)
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "deep.jsonl")
	writeFile(t, sentinel, "")
	// Three-level chain: outerSh -> innerSh -> tail. The resolver must walk
	// past the immediate child and find the grandchild that holds the file.
	script := "sh -c 'tail -f " + sentinel + " & wait $!' & wait $!"
	cmd := exec.Command("sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start outer shell: %v", err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
	})
	waitForFDOpen(t, sentinel, 2*time.Second)

	res, err := a.ResolveAgentSession(context.Background(), cmd.Process.Pid)
	if err != nil {
		t.Fatalf("ResolveAgentSession() error = %v, want nil", err)
	}
	if res.Kind != lsof.AgentKindClaude {
		t.Fatalf("Resolution.Kind = %v, want AgentKindClaude", res.Kind)
	}
	if !strings.HasSuffix(res.Path, "deep.jsonl") {
		t.Fatalf("Resolution.Path = %q, want suffix deep.jsonl", res.Path)
	}
}

// --- ResolveAgentSession: context cancellation propagates ---

func TestAdapter_ResolveAgentSession_CancelledContext_ReturnsContextError(t *testing.T) {
	a := newAdapter(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Use PID 1 (launchd/init) which has many descendants; without cancellation
	// the call would proceed. With ctx already cancelled, the first exec invocation
	// must short-circuit and surface context.Canceled.
	_, err := a.ResolveAgentSession(ctx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ResolveAgentSession(cancelledCtx) error = %v, want context.Canceled", err)
	}
}
