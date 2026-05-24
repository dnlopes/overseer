package styles

import "github.com/dnlopes/overseer/internal/core/domain"

type Glyphs struct {
	Branch        string
	ProjectMode   string
	Repo          string
	PRLink        string
	PRStateOpen   string
	PRStateClosed string
	PRStateMerged string
	PRStateDraft  string
	Added         string
	Removed       string
	Files         string
	CheckPass     string
	CheckFail     string
	CheckPending  string
	Warning       string
	LabelWIP      string
	LabelTesting  string
	LabelReady    string
	LabelDone     string
	LabelDraft    string
	StatusRunning string
	StatusWaiting string
	StatusIdle    string
	StatusDead    string
	StatusUnknown string
}

func NewGlyphs(disableEmoji bool) Glyphs {
	if disableEmoji {
		return Glyphs{
			Branch:        "⎇",
			ProjectMode:   "·",
			Repo:          "⊞",
			PRLink:        "⎘",
			PRStateOpen:   "●",
			PRStateClosed: "●",
			PRStateMerged: "●",
			PRStateDraft:  "●",
			Added:         "⊕",
			Removed:       "⊖",
			Files:         "▤",
			CheckPass:     "✓",
			CheckFail:     "✗",
			CheckPending:  "◷",
			Warning:       "⚠",
			LabelWIP:      "◐",
			LabelTesting:  "◉",
			LabelReady:    "◆",
			LabelDone:     "✓",
			LabelDraft:    "◌",
			StatusRunning: "●",
			StatusWaiting: "◐",
			StatusIdle:    "○",
			StatusDead:    "■",
			StatusUnknown: "?",
		}
	}
	return Glyphs{
		Branch:        "🌿",
		ProjectMode:   "📂",
		Repo:          "📦",
		PRLink:        "🔗",
		PRStateOpen:   "🟢",
		PRStateClosed: "🔴",
		PRStateMerged: "🟣",
		PRStateDraft:  "⚪",
		Added:         "🟩",
		Removed:       "🟥",
		Files:         "📄",
		CheckPass:     "✅",
		CheckFail:     "❌",
		CheckPending:  "⏳",
		Warning:       "⚠️",
		LabelWIP:      "🚧",
		LabelDraft:    "📝",
		LabelTesting:  "🧪",
		LabelReady:    "🚀",
		LabelDone:     "✅",
		StatusRunning: "🟢",
		StatusWaiting: "🟡",
		StatusIdle:    "⚪",
		StatusDead:    "🔴",
		StatusUnknown: "❔",
	}
}

func (g Glyphs) LabelGlyph(code string) string {
	switch code {
	case "WIP":
		return g.LabelWIP
	case "draft":
		return g.LabelDraft
	case "testing":
		return g.LabelTesting
	case "ready":
		return g.LabelReady
	case "done":
		return g.LabelDone
	}
	return ""
}

func (g Glyphs) AgentStatus(kind domain.AgentStatusKind) string {
	switch kind {
	case domain.AgentStatusRunning:
		return g.StatusRunning
	case domain.AgentStatusWaiting:
		return g.StatusWaiting
	case domain.AgentStatusIdle:
		return g.StatusIdle
	case domain.AgentStatusDead:
		return g.StatusDead
	default:
		return g.StatusUnknown
	}
}
