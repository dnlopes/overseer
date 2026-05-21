package claudejsonl

import (
	"testing"
	"time"

	"github.com/dnlopes/overseer/internal/core/domain"
)

func TestResolveActivity_RecentSummary_ReturnsCompacting(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, tool := resolveActivity(scanResult{
		LastActivity:  now.Add(-5 * time.Second),
		LastSummaryAt: now.Add(-5 * time.Second),
		Status:        statusThinking,
	}, now)
	if got != domain.ActivityCompacting {
		t.Fatalf("kind = %v, want ActivityCompacting", got)
	}
	if tool != "" {
		t.Fatalf("tool = %q, want empty for non-tool activity", tool)
	}
}

func TestResolveActivity_StaleSummary_FallsThrough(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, _ := resolveActivity(scanResult{
		LastActivity:  now.Add(-2 * time.Second),
		LastSummaryAt: now.Add(-2 * time.Minute),
		Status:        statusThinking,
	}, now)
	if got != domain.ActivityThinking {
		t.Fatalf("kind = %v, want ActivityThinking (summary too old)", got)
	}
}

func TestResolveActivity_RecentTool_ReturnsStickyToolActivity(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, tool := resolveActivity(scanResult{
		LastActivity: now.Add(-5 * time.Second),
		Status:       statusThinking,
		LastTool:     toolEvent{Name: "Read", Timestamp: now.Add(-10 * time.Second)},
	}, now)
	if got != domain.ActivityReading {
		t.Fatalf("kind = %v, want ActivityReading", got)
	}
	if tool != "Read" {
		t.Fatalf("tool = %q, want Read", tool)
	}
}

func TestResolveActivity_StaleTool_FallsThrough(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, _ := resolveActivity(scanResult{
		LastActivity: now.Add(-1 * time.Second),
		Status:       statusWaitingForUser,
		LastTool:     toolEvent{Name: "Read", Timestamp: now.Add(-2 * time.Minute)},
	}, now)
	if got != domain.ActivityWaiting {
		t.Fatalf("kind = %v, want ActivityWaiting (tool stickiness expired)", got)
	}
}

func TestResolveActivity_WaitingWithinTimeout_ReturnsWaiting(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, _ := resolveActivity(scanResult{
		LastActivity: now.Add(-30 * time.Second),
		Status:       statusWaitingForUser,
	}, now)
	if got != domain.ActivityWaiting {
		t.Fatalf("kind = %v, want ActivityWaiting", got)
	}
}

func TestResolveActivity_WaitingPastTimeout_ReturnsIdle(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, _ := resolveActivity(scanResult{
		LastActivity: now.Add(-3 * time.Minute),
		Status:       statusWaitingForUser,
	}, now)
	if got != domain.ActivityIdle {
		t.Fatalf("kind = %v, want ActivityIdle (waiting past timeout)", got)
	}
}

func TestResolveActivity_SpawningWithinSpawnTimeout_ReturnsSpawning(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, tool := resolveActivity(scanResult{
		LastActivity: now.Add(-10 * time.Minute),
		Status:       statusExecutingTool,
		CurrentTool:  "Task",
	}, now)
	if got != domain.ActivitySpawning {
		t.Fatalf("kind = %v, want ActivitySpawning", got)
	}
	if tool != "Task" {
		t.Fatalf("tool = %q, want Task", tool)
	}
}

func TestResolveActivity_SpawningPastSpawnTimeout_ReturnsIdle(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, _ := resolveActivity(scanResult{
		LastActivity: now.Add(-25 * time.Minute),
		Status:       statusExecutingTool,
		CurrentTool:  "Task",
	}, now)
	if got != domain.ActivityIdle {
		t.Fatalf("kind = %v, want ActivityIdle (spawning past timeout)", got)
	}
}

func TestResolveActivity_NoActivity_ReturnsIdle(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, tool := resolveActivity(scanResult{}, now)
	if got != domain.ActivityIdle {
		t.Fatalf("kind = %v, want ActivityIdle", got)
	}
	if tool != "" {
		t.Fatalf("tool = %q, want empty", tool)
	}
}

func TestResolveActivity_OldActivity_ReturnsIdle(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, _ := resolveActivity(scanResult{
		LastActivity: now.Add(-1 * time.Minute),
		Status:       statusThinking,
	}, now)
	if got != domain.ActivityIdle {
		t.Fatalf("kind = %v, want ActivityIdle (activity past timeout)", got)
	}
}

func TestResolveActivity_RecentThinking_ReturnsThinking(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, _ := resolveActivity(scanResult{
		LastActivity: now.Add(-5 * time.Second),
		Status:       statusThinking,
	}, now)
	if got != domain.ActivityThinking {
		t.Fatalf("kind = %v, want ActivityThinking", got)
	}
}

func TestResolveActivity_RecentProcessingResult_ReturnsThinking(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, _ := resolveActivity(scanResult{
		LastActivity: now.Add(-5 * time.Second),
		Status:       statusProcessingResult,
	}, now)
	if got != domain.ActivityThinking {
		t.Fatalf("kind = %v, want ActivityThinking (processing result)", got)
	}
}

func TestResolveActivity_RecentExecutingTool_ReturnsToolKind(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, tool := resolveActivity(scanResult{
		LastActivity: now.Add(-5 * time.Second),
		Status:       statusExecutingTool,
		CurrentTool:  "Write",
	}, now)
	if got != domain.ActivityWriting {
		t.Fatalf("kind = %v, want ActivityWriting", got)
	}
	if tool != "Write" {
		t.Fatalf("tool = %q, want Write", tool)
	}
}

func TestResolveActivity_UnknownStatus_ReturnsIdle(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000_000, 0)
	got, _ := resolveActivity(scanResult{
		LastActivity: now.Add(-5 * time.Second),
		Status:       statusUnknown,
	}, now)
	if got != domain.ActivityIdle {
		t.Fatalf("kind = %v, want ActivityIdle (unknown status fallback)", got)
	}
}

func TestToolActivity_KnownNames_MapsToKind(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tool string
		want domain.ActivityKind
	}{
		{"Read", domain.ActivityReading},
		{"NotebookRead", domain.ActivityReading},
		{"Write", domain.ActivityWriting},
		{"Edit", domain.ActivityWriting},
		{"NotebookEdit", domain.ActivityWriting},
		{"Bash", domain.ActivityRunning},
		{"BashOutput", domain.ActivityRunning},
		{"KillShell", domain.ActivityRunning},
		{"Glob", domain.ActivitySearching},
		{"Grep", domain.ActivitySearching},
		{"WebFetch", domain.ActivityBrowsing},
		{"WebSearch", domain.ActivityBrowsing},
		{"Task", domain.ActivitySpawning},
		{"Agent", domain.ActivitySpawning},
	}
	for _, tc := range tests {
		t.Run(tc.tool, func(t *testing.T) {
			t.Parallel()
			if got := toolActivity(tc.tool); got != tc.want {
				t.Fatalf("toolActivity(%q) = %v, want %v", tc.tool, got, tc.want)
			}
		})
	}
}

func TestToolActivity_UnknownNonEmpty_ReturnsRunning(t *testing.T) {
	t.Parallel()
	if got := toolActivity("MysteryTool"); got != domain.ActivityRunning {
		t.Fatalf("toolActivity(MysteryTool) = %v, want ActivityRunning (safe default)", got)
	}
}

func TestToolActivity_Empty_ReturnsIdle(t *testing.T) {
	t.Parallel()
	if got := toolActivity(""); got != domain.ActivityIdle {
		t.Fatalf("toolActivity(\"\") = %v, want ActivityIdle", got)
	}
}
