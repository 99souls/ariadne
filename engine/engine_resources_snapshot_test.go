package engine

import "testing"

// TestEngineResourceSnapshotPresence verifies that the ResourceSnapshot is only
// populated when the resource manager is actually constructed (i.e. when the
// facade ResourcesConfig is meaningfully enabled). This guards against future
// regressions that might always allocate a manager (inflating surface / cost)
// or incorrectly populate snapshot fields when disabled.
func TestEngineResourceSnapshotPresence(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*Config)
		expectRes bool
	}{
		{
			name: "disabled-zero-values", // no enabling fields set beyond defaults
			mutate: func(c *Config) {
				// Force resources to zero to be explicit (even if Defaults later changes)
				c.Resources = ResourcesConfig{}
			},
			expectRes: false,
		},
		{
			name: "enabled-cache-capacity", // minimal enabling signal
			mutate: func(c *Config) {
				c.Resources = ResourcesConfig{CacheCapacity: 1}
			},
			expectRes: true,
		},
		{
			name: "enabled-checkpoint-only", // enabling via checkpoint path (no cache/inflight)
			mutate: func(c *Config) {
				c.Resources = ResourcesConfig{CheckpointPath: t.TempDir() + "/checkpoint.log"}
			},
			expectRes: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Defaults()
			// Normalize to explicit zero (guard in case Defaults sets non-zero resource fields)
			cfg.Resources = ResourcesConfig{}
			if tc.mutate != nil {
				tc.mutate(&cfg)
			}
			e, err := New(cfg)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			snap := e.Snapshot()
			if tc.expectRes && snap.Resources == nil {
				t.Fatalf("expected ResourceSnapshot, got nil")
			}
			if !tc.expectRes && snap.Resources != nil {
				t.Fatalf("expected no ResourceSnapshot, got %+v", *snap.Resources)
			}
			if tc.expectRes && snap.Resources != nil {
				rs := snap.Resources
				if rs.CacheEntries != 0 || rs.SpillFiles != 0 || rs.InFlight != 0 || rs.CheckpointQueued != 0 {
					t.Fatalf("expected zeroed counters, got %+v", *rs)
				}
			}
		})
	}
}
