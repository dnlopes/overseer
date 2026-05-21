package shared

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Emit packages a Msg as a Cmd. Use to announce events from inside Update.
//
//	return p, primitives.Emit(SessionSelectedMsg{Session: sess})
func Emit[T any](msg T) tea.Cmd {
	return func() tea.Msg { return msg }
}

// EmitAfter delays a Msg by d. Useful for debouncing or scheduling
// follow-up actions.
func EmitAfter[T any](d time.Duration, msg T) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return msg })
}

// Request runs an async operation and wraps its result in a Msg.
// fn does the work (a use case call, usually); wrap converts the
// result/error into the appropriate Msg type. Wraps in a 30s context
// by default; use RequestWithCtx for custom timeouts.
func Request[T any](
	fn func(context.Context) (T, error),
	wrap func(T, error) tea.Msg,
) tea.Cmd {
	return RequestWithTimeout(30*time.Second, fn, wrap)
}

func RequestWithTimeout[T any](
	timeout time.Duration,
	fn func(context.Context) (T, error),
	wrap func(T, error) tea.Msg,
) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		result, err := fn(ctx)
		return wrap(result, err)
	}
}
