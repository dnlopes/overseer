package opencode

// Pattern strings live here so they can be revised independently of the
// classification logic in detector.go. Each was confirmed against a real
// `tmux capture-pane -p -e` snapshot from OpenCode 1.15.10; see
// testdata/*.txt for the source samples.
//
// Order of preference matters: classify checks waiting signals before
// running because the permission-required modal in OpenCode replaces the
// bottom status bar entirely (no spinner is shown while a permission is
// being asked) — but checking the more specific Waiting marker first is
// a cheap safeguard against future OpenCode UI tweaks that might leave a
// stale "esc interrupt" line in the scrollback alongside a modal.
//
// Note vs Claude Code: OpenCode renders the bottom status line as
//   `■■⬝⬝⬝⬝⬝⬝  esc interrupt`
// (literally "esc interrupt", without "to") whereas Claude Code uses
// "esc to interrupt". Both detectors strip ANSI before matching so the
// color escapes interleaved between "esc" and "interrupt" don't break
// substring search.
const (
	signalRunningInterrupt = "esc interrupt"

	signalWaitingPermissionRequired = "Permission required"
	signalWaitingAllowOnce          = "Allow once"
	signalWaitingReject             = "Reject"
)
