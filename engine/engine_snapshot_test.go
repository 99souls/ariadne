package engine

import (
	"testing"
	"time"
)

// TestSnapshotLimiterPresence ensures limiter snapshot is populated when limiter active.
func TestSnapshotLimiterPresence(t *testing.T) {
	cfg := Config{}
	// Start engine (which constructs internal pipeline and limiter)
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	defer func() { _ = e.Stop() }()

	snap := e.Snapshot()
	if snap.Limiter == nil {
		t.Fatalf("expected limiter snapshot present")
	}
	if snap.Limiter.TotalRequests < 0 {
		t.Fatalf("invalid total requests metric")
	}
}

// TestSnapshotUptimeMonotonic ensures Uptime increases across consecutive snapshots.
func TestSnapshotUptimeMonotonic(t *testing.T) {
	cfg := Config{}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	defer func() { _ = e.Stop() }()

	s1 := e.Snapshot().Uptime
	time.Sleep(10 * time.Millisecond)
	s2 := e.Snapshot().Uptime
	if s2 <= s1 {
		t.Fatalf("expected uptime to increase: %v then %v", s1, s2)
	}
}
