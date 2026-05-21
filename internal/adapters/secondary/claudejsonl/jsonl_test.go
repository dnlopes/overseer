package claudejsonl

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFixture(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return v
}

func TestScanJSONL_EmptyFile_ReturnsZeroResult(t *testing.T) {
	t.Parallel()
	path := writeFixture(t)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res != (scanResult{}) {
		t.Fatalf("res = %+v, want zero scanResult", res)
	}
}

func TestScanJSONL_MissingFile_ReturnsWrappedFsErr(t *testing.T) {
	t.Parallel()

	_, err := scanJSONL(filepath.Join(t.TempDir(), "does-not-exist.jsonl"))

	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want fs.ErrNotExist", err)
	}
}

func TestScanJSONL_MalformedLinesSkipped(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`not a json`,
		`{"truncated":`,
		`{"type":"user","timestamp":"2026-05-21T00:56:59Z","message":{"role":"user","content":"hi"}}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res.Status != statusThinking {
		t.Fatalf("Status = %v, want statusThinking (malformed lines skipped, good line parsed)", res.Status)
	}
}

func TestScanJSONL_UserPlainText_StatusThinking(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`{"type":"user","timestamp":"2026-05-21T00:56:59Z","message":{"role":"user","content":"count to 5"}}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res.Status != statusThinking {
		t.Fatalf("Status = %v, want statusThinking", res.Status)
	}
	want := mustParseTime(t, "2026-05-21T00:56:59Z")
	if !res.LastActivity.Equal(want) {
		t.Fatalf("LastActivity = %v, want %v", res.LastActivity, want)
	}
}

func TestScanJSONL_UserToolResult_StatusProcessingResult(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`{"type":"user","timestamp":"2026-05-21T00:57:00Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"abc","content":"file contents..."}]}}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res.Status != statusProcessingResult {
		t.Fatalf("Status = %v, want statusProcessingResult", res.Status)
	}
}

func TestScanJSONL_AssistantTextOnly_StatusWaitingForUser(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`{"type":"user","timestamp":"2026-05-21T00:56:59Z","message":{"role":"user","content":"hi"}}`,
		`{"type":"assistant","timestamp":"2026-05-21T00:57:00Z","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]}}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res.Status != statusWaitingForUser {
		t.Fatalf("Status = %v, want statusWaitingForUser", res.Status)
	}
}

func TestScanJSONL_AssistantToolUse_StatusExecutingTool(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`{"type":"assistant","timestamp":"2026-05-21T00:57:00Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/foo"}}]}}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res.Status != statusExecutingTool {
		t.Fatalf("Status = %v, want statusExecutingTool", res.Status)
	}
	if res.CurrentTool != "Read" {
		t.Fatalf("CurrentTool = %q, want Read", res.CurrentTool)
	}
}

func TestScanJSONL_TracksMostRecentToolUse(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`{"type":"assistant","timestamp":"2026-05-21T00:57:00Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{}}]}}`,
		`{"type":"user","timestamp":"2026-05-21T00:57:01Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"a","content":"x"}]}}`,
		`{"type":"assistant","timestamp":"2026-05-21T00:57:02Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{}}]}}`,
		`{"type":"user","timestamp":"2026-05-21T00:57:03Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"b","content":"y"}]}}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res.LastTool.Name != "Bash" {
		t.Fatalf("LastTool.Name = %q, want Bash (most recent tool_use)", res.LastTool.Name)
	}
	wantTS := mustParseTime(t, "2026-05-21T00:57:02Z")
	if !res.LastTool.Timestamp.Equal(wantTS) {
		t.Fatalf("LastTool.Timestamp = %v, want %v", res.LastTool.Timestamp, wantTS)
	}
}

func TestScanJSONL_SkipsNonUserAssistantEntries(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`{"type":"permission-mode","timestamp":"2026-05-21T00:56:58Z","permissionMode":"bypassPermissions"}`,
		`{"type":"file-history-snapshot","timestamp":"2026-05-21T00:56:58.5Z"}`,
		`{"type":"system","subtype":"turn_duration","timestamp":"2026-05-21T00:57:05Z","durationMs":3005}`,
		`{"type":"user","timestamp":"2026-05-21T00:56:59Z","message":{"role":"user","content":"hi"}}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res.Status != statusThinking {
		t.Fatalf("Status = %v, want statusThinking (only user/assistant drives Status)", res.Status)
	}
	wantTS := mustParseTime(t, "2026-05-21T00:56:59Z")
	if !res.LastActivity.Equal(wantTS) {
		t.Fatalf("LastActivity = %v, want %v (only user/assistant updates LastActivity)", res.LastActivity, wantTS)
	}
}

func TestScanJSONL_SummaryUsesPreviousTimestamp(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`{"type":"user","timestamp":"2026-05-21T00:56:59Z","message":{"role":"user","content":"hi"}}`,
		`{"type":"assistant","timestamp":"2026-05-21T00:57:00Z","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]}}`,
		`{"type":"summary","summary":"the session was about greetings"}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	want := mustParseTime(t, "2026-05-21T00:57:00Z")
	if !res.LastSummaryAt.Equal(want) {
		t.Fatalf("LastSummaryAt = %v, want %v (timestamp of entry preceding summary)", res.LastSummaryAt, want)
	}
}

func TestScanJSONL_NullMessage_TreatedAsUnknown(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`{"type":"user","timestamp":"2026-05-21T00:56:59Z","message":null}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res.Status != statusUnknown {
		t.Fatalf("Status = %v, want statusUnknown (null message ignored)", res.Status)
	}
}

func TestScanJSONL_ContentNeitherStringNorArray_TreatedAsEmpty(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`{"type":"user","timestamp":"2026-05-21T00:56:59Z","message":{"role":"user","content":42}}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res.Status != statusThinking {
		t.Fatalf("Status = %v, want statusThinking (numeric content treated as empty, but still a user message)", res.Status)
	}
}

func TestScanReader_ScannerError_IsSurfaced(t *testing.T) {
	t.Parallel()

	_, err := scanReader(errReader{})

	if err == nil {
		t.Fatalf("err = nil, want scanner error to surface")
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) { return 0, errFakeRead }

var errFakeRead = errors.New("synthetic read failure")

func TestScanJSONL_RealFixtureFromLiveSession(t *testing.T) {
	t.Parallel()
	path := writeFixture(t,
		`{"type":"permission-mode","permissionMode":"bypassPermissions","sessionId":"11212fe5"}`,
		`{"type":"file-history-snapshot","messageId":"5b716020","snapshot":{}}`,
		`{"parentUuid":null,"type":"user","message":{"role":"user","content":"Count from 1 to 5"},"timestamp":"2026-05-21T00:57:49.7Z","sessionId":"11212fe5"}`,
		`{"parentUuid":"5b716020","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"1. One — not prime\n2. Two — prime"}]},"timestamp":"2026-05-21T00:57:52.7Z","sessionId":"11212fe5"}`,
		`{"type":"system","subtype":"stop_hook_summary","timestamp":"2026-05-21T00:57:52.74Z"}`,
		`{"type":"system","subtype":"turn_duration","durationMs":3005,"timestamp":"2026-05-21T00:57:52.74Z"}`,
	)

	res, err := scanJSONL(path)

	if err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if res.Status != statusWaitingForUser {
		t.Fatalf("Status = %v, want statusWaitingForUser (final assistant message was text)", res.Status)
	}
	wantTS := mustParseTime(t, "2026-05-21T00:57:52.7Z")
	if !res.LastActivity.Equal(wantTS) {
		t.Fatalf("LastActivity = %v, want %v (last assistant entry timestamp)", res.LastActivity, wantTS)
	}
}
