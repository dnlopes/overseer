package session

import (
	"context"
	"sort"

	domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
)

type ListRequest struct{}

type SessionGroup struct {
	ProjectName string
	Sessions    []domainsession.Session
}

type ListResponse struct {
	Groups []SessionGroup
}

type ListUseCase struct {
	repo domainsession.Repository
}

func NewListUseCase(repo domainsession.Repository) *ListUseCase {
	return &ListUseCase{repo: repo}
}

func (uc *ListUseCase) Execute(ctx context.Context, req ListRequest) (ListResponse, error) {
	sessions, err := uc.repo.List(ctx)
	if err != nil {
		return ListResponse{}, err
	}

	grouped := make(map[string]*SessionGroup)
	for _, s := range sessions {
		if _, ok := grouped[s.ProjectName]; !ok {
			grouped[s.ProjectName] = &SessionGroup{ProjectName: s.ProjectName}
		}
		grouped[s.ProjectName].Sessions = append(grouped[s.ProjectName].Sessions, s)
	}

	groups := make([]SessionGroup, 0, len(grouped))
	for _, g := range grouped {
		sort.Slice(g.Sessions, func(i, j int) bool { return g.Sessions[i].Order < g.Sessions[j].Order })
		groups = append(groups, *g)
	}

	sort.Slice(groups, func(i, j int) bool { return groups[i].ProjectName < groups[j].ProjectName })

	return ListResponse{Groups: groups}, nil
}
