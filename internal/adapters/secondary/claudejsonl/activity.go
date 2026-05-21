package claudejsonl

import (
	"time"

	"github.com/dnlopes/overseer/internal/core/domain"
)

const (
	activityTimeout = 30 * time.Second
	waitingTimeout  = 2 * time.Minute
	spawningTimeout = 20 * time.Minute
)

type jsonlStatus int

const (
	statusUnknown jsonlStatus = iota
	statusWaitingForUser
	statusExecutingTool
	statusThinking
	statusProcessingResult
)

type toolEvent struct {
	Name      string
	Timestamp time.Time
}

type scanResult struct {
	LastActivity  time.Time
	LastSummaryAt time.Time
	Status        jsonlStatus
	CurrentTool   string
	LastTool      toolEvent
}

func resolveActivity(s scanResult, now time.Time) (domain.ActivityKind, string) {
	if !s.LastSummaryAt.IsZero() && now.Sub(s.LastSummaryAt) < activityTimeout {
		return domain.ActivityCompacting, ""
	}

	if !s.LastTool.Timestamp.IsZero() && now.Sub(s.LastTool.Timestamp) < activityTimeout {
		return toolActivity(s.LastTool.Name), s.LastTool.Name
	}

	if s.Status == statusWaitingForUser {
		if !s.LastActivity.IsZero() && now.Sub(s.LastActivity) < waitingTimeout {
			return domain.ActivityWaiting, ""
		}
		return domain.ActivityIdle, ""
	}

	if s.Status == statusExecutingTool {
		kind := toolActivity(s.CurrentTool)
		if kind == domain.ActivitySpawning && !s.LastActivity.IsZero() && now.Sub(s.LastActivity) < spawningTimeout {
			return domain.ActivitySpawning, s.CurrentTool
		}
	}

	if s.LastActivity.IsZero() || now.Sub(s.LastActivity) > activityTimeout {
		return domain.ActivityIdle, ""
	}

	switch s.Status {
	case statusThinking, statusProcessingResult:
		return domain.ActivityThinking, ""
	case statusExecutingTool:
		return toolActivity(s.CurrentTool), s.CurrentTool
	}
	return domain.ActivityIdle, ""
}

func toolActivity(tool string) domain.ActivityKind {
	switch tool {
	case "Read", "NotebookRead":
		return domain.ActivityReading
	case "Write", "Edit", "NotebookEdit":
		return domain.ActivityWriting
	case "Bash", "BashOutput", "KillShell":
		return domain.ActivityRunning
	case "Glob", "Grep":
		return domain.ActivitySearching
	case "WebFetch", "WebSearch":
		return domain.ActivityBrowsing
	case "Task", "Agent":
		return domain.ActivitySpawning
	default:
		if tool != "" {
			return domain.ActivityRunning
		}
		return domain.ActivityIdle
	}
}
