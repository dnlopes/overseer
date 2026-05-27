// Package registry provides an in-memory implementation of
// domain.AgentStatusDetectorRegistry. Wiring is constructed once at startup
// in cmd/overseer/main.go and never mutated thereafter, so the registry
// trades thread-safety for simplicity — concurrent Register calls are not
// supported.
package registry

import (
	"github.com/dnlopes/overseer/internal/core/domain"
)

// Registry maps an AgentType to its AgentStatusDetector. Last-write-wins
// for duplicates so a wiring change in main.go (e.g. swapping a pane-parser
// for a hook-based detector) is one Register call away.
type Registry struct {
	byType map[domain.AgentType]domain.AgentStatusDetector
}

func New() *Registry {
	return &Registry{byType: make(map[domain.AgentType]domain.AgentStatusDetector)}
}

func (r *Registry) Register(d domain.AgentStatusDetector) {
	r.byType[d.AgentType()] = d
}

func (r *Registry) DetectorFor(agentType domain.AgentType) (domain.AgentStatusDetector, bool) {
	if agentType == "" {
		return nil, false
	}
	d, ok := r.byType[agentType]
	return d, ok
}

var _ domain.AgentStatusDetectorRegistry = (*Registry)(nil)
