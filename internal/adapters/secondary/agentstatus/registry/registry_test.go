package registry

import (
	"context"
	"testing"

	"github.com/dnlopes/overseer/internal/core/domain"
)

func TestRegistry_DetectorFor_ReturnsRegisteredDetector(t *testing.T) {
	d := &stubDetector{agentType: domain.AgentTypeClaudeCode}
	r := New()
	r.Register(d)

	got, ok := r.DetectorFor(domain.AgentTypeClaudeCode)
	if !ok {
		t.Fatal("DetectorFor(claude-code): ok = false, want true")
	}
	if got != d {
		t.Fatalf("DetectorFor(claude-code) = %v, want %v", got, d)
	}
}

func TestRegistry_DetectorFor_UnknownType_ReturnsFalse(t *testing.T) {
	r := New()
	r.Register(&stubDetector{agentType: domain.AgentTypeClaudeCode})

	_, ok := r.DetectorFor(domain.AgentTypeOpenCode)
	if ok {
		t.Fatal("DetectorFor(opencode): ok = true, want false (not registered)")
	}
}

func TestRegistry_DetectorFor_EmptyType_ReturnsFalse(t *testing.T) {
	r := New()
	if _, ok := r.DetectorFor(""); ok {
		t.Fatal("DetectorFor(\"\"): ok = true, want false")
	}
}

func TestRegistry_Register_OverridesExistingType(t *testing.T) {
	first := &stubDetector{agentType: domain.AgentTypeClaudeCode, tag: "first"}
	second := &stubDetector{agentType: domain.AgentTypeClaudeCode, tag: "second"}

	r := New()
	r.Register(first)
	r.Register(second)

	got, ok := r.DetectorFor(domain.AgentTypeClaudeCode)
	if !ok {
		t.Fatal("DetectorFor: not registered after second Register")
	}
	if got.(*stubDetector).tag != "second" {
		t.Fatalf("DetectorFor returned tag = %q, want %q", got.(*stubDetector).tag, "second")
	}
}

func TestRegistry_Register_MultipleAgentTypesCoexist(t *testing.T) {
	claude := &stubDetector{agentType: domain.AgentTypeClaudeCode, tag: "claude"}
	opencode := &stubDetector{agentType: domain.AgentTypeOpenCode, tag: "opencode"}

	r := New()
	r.Register(claude)
	r.Register(opencode)

	gotClaude, ok := r.DetectorFor(domain.AgentTypeClaudeCode)
	if !ok || gotClaude != claude {
		t.Fatalf("DetectorFor(claude-code) = %v, ok = %v; want %v, true", gotClaude, ok, claude)
	}
	gotOpenCode, ok := r.DetectorFor(domain.AgentTypeOpenCode)
	if !ok || gotOpenCode != opencode {
		t.Fatalf("DetectorFor(opencode) = %v, ok = %v; want %v, true", gotOpenCode, ok, opencode)
	}
}

func TestRegistry_New_EmptyRegistry_NoDetectors(t *testing.T) {
	r := New()
	for _, at := range []domain.AgentType{
		domain.AgentTypeClaudeCode,
		domain.AgentTypeOpenCode,
		domain.AgentTypeUnknown,
		"",
	} {
		if _, ok := r.DetectorFor(at); ok {
			t.Fatalf("DetectorFor(%q): ok = true on empty registry", at)
		}
	}
}

func TestRegistry_SatisfiesDomainPort(t *testing.T) {
	var _ domain.AgentStatusDetectorRegistry = (*Registry)(nil)
}

type stubDetector struct {
	agentType domain.AgentType
	tag       string
}

func (s *stubDetector) AgentType() domain.AgentType { return s.agentType }
func (s *stubDetector) Detect(_ context.Context, _ domain.Session) (domain.AgentStatus, error) {
	return domain.AgentStatus{}, nil
}
