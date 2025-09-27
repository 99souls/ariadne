package configx

import "testing"

func TestApplier_EventsAndMetrics(t *testing.T) {
	store := NewVersionedStore()
	sim := NewSimulator()
	ap := NewApplier(store, sim)
	dispatcher := NewDispatcher()
	metrics := &InMemoryMetrics{}
	ap.WithDispatcher(dispatcher).WithMetrics(metrics)

	coll := &InMemoryCollector{}
	dispatcher.Register(coll)

	// Successful apply
	spec := &EngineConfigSpec{Policies: &PoliciesConfigSection{BusinessRules: []*PolicyRuleSpec{{ID: "r1"}}}}
	res, err := ap.Apply(nil, spec, ApplyOptions{Actor: "tester"})
	if err != nil {
		t.Fatalf("unexpected apply err: %v", err)
	}
	if res.Version != 1 {
		t.Fatalf("expected version 1 got %d", res.Version)
	}
	if metrics.ApplySuccess != 1 || metrics.ApplyFailure != 0 {
		t.Fatalf("metrics mismatch: %+v", metrics)
	}
	if metrics.ActiveVer != 1 {
		t.Fatalf("active version metric mismatch")
	}
	foundApply := false
	for _, e := range coll.Events {
		if e.Type == "apply" && e.Version == 1 {
			foundApply = true
		}
	}
	if !foundApply {
		t.Fatalf("missing apply event: %+v", coll.Events)
	}

	// Simulation rejection path (add many rules to trip heuristic) without force
	// create many rules to exceed heuristic threshold
	many := make([]*PolicyRuleSpec, 25)
	for i := 0; i < 25; i++ {
		many[i] = &PolicyRuleSpec{ID: itoa64(int64(i))}
	}
	bad := &EngineConfigSpec{Policies: &PoliciesConfigSection{BusinessRules: many}}
	_, err = ap.Apply(spec, bad, ApplyOptions{Actor: "tester"})
	if err == nil {
		t.Fatalf("expected simulation rejection")
	}
	if metrics.ApplyFailure != 1 {
		t.Fatalf("expected failure metric increment")
	}
	foundReject := false
	for _, e := range coll.Events {
		if e.Type == "simulation_reject" {
			foundReject = true
		}
	}
	if !foundReject {
		t.Fatalf("missing simulation_reject event")
	}

	// Rollback to version 1 (noop but new version 2 created)
	res2, err := ap.Rollback(1, "tester")
	if err != nil {
		t.Fatalf("rollback err: %v", err)
	}
	if res2.Version != 2 {
		t.Fatalf("expected new version 2 got %d", res2.Version)
	}
	if metrics.Rollbacks != 1 {
		t.Fatalf("expected rollback metric increment")
	}
	foundRollback := false
	for _, e := range coll.Events {
		if e.Type == "rollback" && e.Version == 2 {
			foundRollback = true
		}
	}
	if !foundRollback {
		t.Fatalf("missing rollback event")
	}
}
