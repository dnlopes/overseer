package shared

import "github.com/dnlopes/overseer/internal/core/domain"

type SessionCreatedMsg struct{ Session domain.Session }

type SessionSelectedMsg struct{ ID string }

type SessionsLoadedMsg struct {
	Sessions []domain.Session
	Err      error
}

type NewSessionPopupCloseMsg struct{}

type SessionCreateErrMsg struct{ Err error }
