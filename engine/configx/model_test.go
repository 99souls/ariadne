package configx

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEngineConfigSpecZeroValue(t *testing.T) {
	var spec EngineConfigSpec
	if spec.Global != nil || spec.Crawling != nil || spec.Policies != nil {
		// zero-value should have all nil pointers
		b, _ := json.Marshal(spec)
		t.Fatalf("expected zero-value pointers to be nil, got %s", string(b))
	}
}

func TestVersionedConfigBasicMarshal(t *testing.T) {
	vc := &VersionedConfig{
		Version:   1,
		Spec:      &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 10}},
		Hash:      "deadbeef",
		AppliedAt: time.Unix(100, 0),
		Actor:     "tester",
		Parent:    0,
	}
	data, err := json.Marshal(vc)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty json output")
	}
	// Spot check a field presence
	if !jsonContains(data, "\"version\":1") {
		t.Fatalf("expected version field in output: %s", string(data))
	}
}

func jsonContains(b []byte, substr string) bool {
	return string(b) != "" && (len(substr) == 0 || contains(string(b), substr))
}

// contains is a tiny helper to avoid importing strings for one call.
func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	// naive search sufficient for test
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
