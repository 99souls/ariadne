package configx

import "testing"

func TestSimulatorBasic(t *testing.T) {
	sim := NewSimulator()
	current := &EngineConfigSpec{Policies: &PoliciesConfigSection{EnabledFlags: map[string]bool{"a": true}, BusinessRules: []*PolicyRuleSpec{{ID: "r1"}}}}
	candidate := &EngineConfigSpec{Policies: &PoliciesConfigSection{EnabledFlags: map[string]bool{"a": true, "b": true}, BusinessRules: []*PolicyRuleSpec{{ID: "r1"}, {ID: "r2"}}}}
	impact := sim.Simulate(current, candidate)
	if impact.RuleCountDelta != 1 { t.Fatalf("expected rule delta 1 got %d", impact.RuleCountDelta) }
	if len(impact.FlagsAdded) != 1 || impact.FlagsAdded[0] != "b" { t.Fatalf("expected flag 'b' added") }
	if len(impact.FlagsRemoved) != 0 { t.Fatalf("expected no removed flags") }
	if impact.ProjectedLatency <= 0 { t.Fatalf("expected positive projected latency") }
}

func TestSimulatorNoChange(t *testing.T) {
	sim := NewSimulator()
	current := &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 1}}
	candidate := &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 1}}
	impact := sim.Simulate(current, candidate)
	if impact.ChangedFields != 0 { t.Fatalf("expected zero changed sections") }
	found := false
	for _, n := range impact.Notes { if n == "no structural section changes detected" { found = true } }
	if !found { t.Fatalf("expected explanatory note for no changes") }
}
