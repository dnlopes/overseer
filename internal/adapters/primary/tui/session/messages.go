package session

import "github.com/dnlopes/overseer/internal/core/domain"

type SessionCreatedMsg struct{ Session domain.Session }

type SessionRenamedMsg struct{ Session domain.Session }

type CancelFormMsg struct{}
