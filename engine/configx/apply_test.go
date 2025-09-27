package configx

import "testing"

func TestApplyDryRun(t *testing.T) {
	store := NewVersionedStore()
	applier := NewApplier(store, NewSimulator())
	candidate := &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 5}}
	res, err := applier.Apply(nil, candidate, ApplyOptions{Actor: "tester", DryRun: true})
	if err != nil { t.Fatalf("dry run failed: %v", err) }
	if res.Version != 0 { t.Fatalf("expected version 0 for dry run got %d", res.Version) }
	if _, ok := store.Head(); ok { t.Fatalf("store should remain empty after dry run") }
}

func TestApplyCommit(t *testing.T) {
	store := NewVersionedStore()
	applier := NewApplier(store, NewSimulator())
	candidate := &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 5}, Policies: &PoliciesConfigSection{BusinessRules: []*PolicyRuleSpec{{ID: "r1"}}}}
	res, err := applier.Apply(nil, candidate, ApplyOptions{Actor: "tester"})
	if err != nil { t.Fatalf("apply failed: %v", err) }
	if res.Version != 1 { t.Fatalf("expected version 1 got %d", res.Version) }
	if res.SimImpact == nil || !res.SimImpact.Acceptable { t.Fatalf("expected acceptable simulation impact") }
}

func TestApplySimulationReject(t *testing.T) {
	store := NewVersionedStore()
	applier := NewApplier(store, NewSimulator())
	// Create large rule delta to exceed Acceptable threshold (>20 new rules triggers rejection)
	var rules []*PolicyRuleSpec
	for i := 0; i < 25; i++ { rules = append(rules, &PolicyRuleSpec{ID: itoa64(int64(i))}) }
	candidate := &EngineConfigSpec{Policies: &PoliciesConfigSection{BusinessRules: rules}}
	_, err := applier.Apply(nil, candidate, ApplyOptions{Actor: "tester"})
	if err == nil { t.Fatalf("expected simulation rejection") }
	// Forced apply should succeed
	res, err := applier.Apply(nil, candidate, ApplyOptions{Actor: "tester", Force: true})
	if err != nil || res.Version != 1 { t.Fatalf("forced apply failed: %v", err) }
}

func TestRollback(t *testing.T) {
	store := NewVersionedStore()
	applier := NewApplier(store, NewSimulator())
	first := &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 1}}
	second := &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 2}}
	_, _ = applier.Apply(nil, first, ApplyOptions{Actor: "a"})
	_, _ = applier.Apply(first, second, ApplyOptions{Actor: "b"})
	res, err := applier.Rollback(1, "rollback-actor")
	if err != nil { t.Fatalf("rollback failed: %v", err) }
	if res.Version != 3 { t.Fatalf("expected new version 3 after rollback got %d", res.Version) }
}
