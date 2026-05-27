package claudecode

// Pattern strings live here so they can be revised independently of the
// classification logic in detector.go. Each was confirmed against a real
// `tmux capture-pane -p -e` snapshot from Claude Code; see
// testdata/*.txt for the source samples.
//
// Order of preference matters: classify checks waiting signals before
// running because some waiting fixtures still contain past-tense spinner
// text from earlier turns. Waiting signals never coincide with the
// running signal in real captures — the bottom status bar swaps between
// them — but checking the more specific pattern first is a cheap
// safeguard against future Claude-Code UI changes.
const (
	signalRunningInterrupt = "esc to interrupt"

	signalWaitingProceed       = "Do you want to proceed?"
	signalWaitingConfirm       = "Enter to confirm"
	signalWaitingEscCancel     = "Esc to cancel"
	signalWaitingTabToAmend    = "Tab to amend"
	signalWaitingEnterToSelect = "Enter to select"
)
