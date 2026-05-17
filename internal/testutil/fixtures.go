package testutil

import "github.com/dnlopes/overseer/internal/core/domain"

func MakeSession(name, project string) domain.Session {
	s, err := domain.NewSession(name, project)
	if err != nil {
		panic(err)
	}
	return s
}
