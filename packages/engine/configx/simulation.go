package configx

import "time"

// SimulationImpact is a placeholder structure describing projected effects.
type SimulationImpact struct {
	ChangedFields    int               `json:"changed_fields"`
	ProjectedLatency time.Duration     `json:"projected_latency"`
	RuleCountDelta   int               `json:"rule_count_delta"`
	FlagsAdded       []string          `json:"flags_added,omitempty"`
	FlagsRemoved     []string          `json:"flags_removed,omitempty"`
	Notes            []string          `json:"notes,omitempty"`
	Acceptable       bool              `json:"acceptable"`
}

// Simulator computes a mock impact analysis between current and candidate specs.
type Simulator struct{}

func NewSimulator() *Simulator { return &Simulator{} }

// Simulate compares two specs and produces a deterministic (mock) impact report.
// Later iterations can plug in metric baselines & heuristics.
func (s *Simulator) Simulate(current, candidate *EngineConfigSpec) *SimulationImpact {
	impact := &SimulationImpact{}
	if current == nil && candidate != nil { impact.ChangedFields = countNonNilSections(candidate) }
	if current != nil && candidate != nil { impact.ChangedFields = diffSectionCount(current, candidate) }
	impact.RuleCountDelta = ruleDelta(current, candidate)
	impact.FlagsAdded, impact.FlagsRemoved = diffFlags(current, candidate)
	// Simple latency heuristic: +1ms per added business rule (mock)
	impact.ProjectedLatency = time.Duration(max(0, impact.RuleCountDelta)) * time.Millisecond
	// Heuristic: Impact acceptable if projected latency increase < 10ms and added rules < 20.
	if impact.ProjectedLatency < 10*time.Millisecond && impact.RuleCountDelta < 20 {
		impact.Acceptable = true
	}
	if impact.ChangedFields == 0 {
		impact.Notes = append(impact.Notes, "no structural section changes detected")
	}
	return impact
}

func countNonNilSections(s *EngineConfigSpec) int {
	c := 0
	if s.Global != nil { c++ }
	if s.Crawling != nil { c++ }
	if s.Processing != nil { c++ }
	if s.Output != nil { c++ }
	if s.Policies != nil { c++ }
	if s.Rollout != nil { c++ }
	return c
}

func diffSectionCount(a, b *EngineConfigSpec) int {
	return abs(countNonNilSections(b) - countNonNilSections(a))
}

func ruleDelta(a, b *EngineConfigSpec) int {
	var ac, bc int
	if a != nil && a.Policies != nil { ac = len(a.Policies.BusinessRules) }
	if b != nil && b.Policies != nil { bc = len(b.Policies.BusinessRules) }
	return bc - ac
}

func diffFlags(a, b *EngineConfigSpec) (added, removed []string) {
	am := map[string]bool{}
	bm := map[string]bool{}
	if a != nil && a.Policies != nil && a.Policies.EnabledFlags != nil {
		for k := range a.Policies.EnabledFlags { am[k] = true }
	}
	if b != nil && b.Policies != nil && b.Policies.EnabledFlags != nil {
		for k := range b.Policies.EnabledFlags { bm[k] = true }
	}
	for k := range bm { if !am[k] { added = append(added, k) } }
	for k := range am { if !bm[k] { removed = append(removed, k) } }
	return
}

func abs(x int) int { if x < 0 { return -x }; return x }
func max(a, b int) int { if a > b { return a }; return b }
