package configx

import "testing"

func TestValidateSpec(t *testing.T) {
	if err := ValidateSpec(&EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 1}}); err != nil {
		// basic valid
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateSpec(&EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: -1}}); err != ErrNegativeConcurrency {
		if err == nil { t.Fatalf("expected negative concurrency error") } else { t.Fatalf("unexpected: %v", err) }
	}
	if err := ValidateSpec(&EngineConfigSpec{Rollout: &RolloutSpec{Mode: "percentage", Percentage: 101}}); err != ErrPercentageOutOfRange {
		if err == nil { t.Fatalf("expected percentage out of range error") } else { t.Fatalf("unexpected: %v", err) }
	}
	if err := ValidateSpec(&EngineConfigSpec{Rollout: &RolloutSpec{Mode: "invalid"}}); err != ErrInvalidRolloutMode {
		if err == nil { t.Fatalf("expected invalid mode error") } else { t.Fatalf("unexpected: %v", err) }
	}
	if err := ValidateSpec(&EngineConfigSpec{Global: &GlobalConfigSection{RetryPolicy: &RetryPolicySpec{MaxRetries: -1}}}); err != ErrNegativeRetryConfig {
		if err == nil { t.Fatalf("expected negative retry config error") } else { t.Fatalf("unexpected: %v", err) }
	}
}
