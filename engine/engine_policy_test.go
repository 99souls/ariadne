package engine

import (
	"context"
	"testing"
	"time"

	engpipeline "github.com/99souls/ariadne/engine/internal/pipeline"
)

// TestPolicyUpdateAffectsPipelineProbe ensures updating TelemetryPolicy changes health classification thresholds.
func TestPolicyUpdateAffectsPipelineProbe(t *testing.T) {
	cfg := Config{}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	// Simulate pipeline metrics
	m := &engpipeline.PipelineMetrics{TotalProcessed: 20, TotalFailed: 5} // 25% failure ratio
	e.pl.SetMetricsForTest(m)
	// snapshot with default policy (degraded threshold 50%) should be healthy
	snap := e.HealthSnapshot(context.Background())
	if string(snap.Overall) != "healthy" { // healthy expected
		t.Fatalf("expected healthy with default policy, got %s", snap.Overall)
	}
	// Update policy lowering degraded threshold to 0.20 so ratio now exceeds degraded but not unhealthy (keep unhealthy at 0.8)
	p := DefaultTelemetryPolicy()
	p.Health.PipelineDegradedRatio = 0.20
	p.Health.PipelineUnhealthyRatio = 0.60
	p.Health.ProbeTTL = 1 * time.Millisecond // force fast expiry
	e.UpdateTelemetryPolicy(&p)
	// sleep to allow TTL to expire
	time.Sleep(2 * time.Millisecond)
	snap2 := e.HealthSnapshot(context.Background())
	if string(snap2.Overall) != "degraded" {
		t.Fatalf("expected degraded after policy update, got %s", snap2.Overall)
	}
}
