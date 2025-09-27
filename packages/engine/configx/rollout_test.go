package configx

import "testing"

func TestRolloutEvaluator_Full(t *testing.T) {
	s := NewVersionedStore()
	spec := &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 1}, Rollout: &RolloutSpec{Mode: "full"}}
	vc, err := s.Append(spec, "actor", "", 0)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	ev := NewRolloutEvaluator(s)
	if got := ev.ActiveVersionForDomain("any.com"); got != vc.Version {
		t.Fatalf("expected head version")
	}
}

func TestRolloutEvaluator_Percentage(t *testing.T) {
	s := NewVersionedStore()
	// base version
	base, _ := s.Append(&EngineConfigSpec{Global: &GlobalConfigSection{LoggingLevel: "info"}}, "actor", "", 0)
	// staged rollout 50%
	head, _ := s.Append(&EngineConfigSpec{Global: &GlobalConfigSection{LoggingLevel: "debug"}, Rollout: &RolloutSpec{Mode: "percentage", Percentage: 25}}, "actor", "", base.Version)
	ev := NewRolloutEvaluator(s)
	// Deterministic FNV32a mapping: we just assert each chosen domain is either base or head; at least one difference to ensure hashing.
	domains := []string{"alpha.com", "beta.com", "gamma.com", "delta.com", "epsilon.com", "zeta.com", "eta.com", "theta.com", "iota.com", "kappa.com"}
	var sawBase, sawHead bool
	for _, d := range domains {
		v := ev.ActiveVersionForDomain(d)
		switch v {
		case head.Version:
			sawHead = true
		case base.Version:
			sawBase = true
		default:
									 t.Fatalf("unexpected version %d for domain %s", v, d)
		}
	}
	if !sawBase || !sawHead {
		t.Fatalf("expected mixture base=%v head=%v", sawBase, sawHead)
	}
}

func TestRolloutEvaluator_Cohort(t *testing.T) {
	s := NewVersionedStore()
	base, _ := s.Append(&EngineConfigSpec{Global: &GlobalConfigSection{LoggingLevel: "info"}}, "actor", "", 0)
	head, _ := s.Append(&EngineConfigSpec{Global: &GlobalConfigSection{LoggingLevel: "debug"}, Rollout: &RolloutSpec{Mode: "cohort", CohortDomains: []string{"target.com"}}}, "actor", "", base.Version)
	ev := NewRolloutEvaluator(s)
	if v := ev.ActiveVersionForDomain("target.com"); v != head.Version {
		t.Fatalf("target domain should get head version")
	}
	if v := ev.ActiveVersionForDomain("other.com"); v != base.Version {
		t.Fatalf("non-cohort domain should get base version got %d want %d", v, base.Version)
	}
}
