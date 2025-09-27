package health

import (
	"context"
	"testing"
	"time"
)

func TestEvaluatorCachingAndRollup(t *testing.T) {
	var calls int
	p := ProbeFunc(func(ctx context.Context) ProbeResult { calls++; return Healthy("unit") })
	ev := NewEvaluator(200*time.Millisecond, p)
	s1 := ev.Evaluate(context.Background())
	s2 := ev.Evaluate(context.Background())
	if calls != 1 {
		t.Fatalf("expected caching (1 call) got %d", calls)
	}
	if s1.Overall != StatusHealthy || s2.Overall != StatusHealthy {
		t.Fatalf("expected healthy rollup")
	}
	time.Sleep(220 * time.Millisecond)
	_ = ev.Evaluate(context.Background())
	if calls != 2 {
		t.Fatalf("expected second evaluation after ttl")
	}
}

func TestEvaluatorRollupDegraded(t *testing.T) {
	p1 := ProbeFunc(func(ctx context.Context) ProbeResult { return Healthy("a") })
	p2 := ProbeFunc(func(ctx context.Context) ProbeResult { return Degraded("b", "lag") })
	ev := NewEvaluator(0, p1, p2)
	s := ev.Evaluate(context.Background())
	if s.Overall != StatusDegraded {
		t.Fatalf("expected degraded overall got %s", s.Overall)
	}
}

func TestEvaluatorRollupUnhealthy(t *testing.T) {
	p1 := ProbeFunc(func(ctx context.Context) ProbeResult { return Healthy("a") })
	p2 := ProbeFunc(func(ctx context.Context) ProbeResult { return Unhealthy("b", "down") })
	ev := NewEvaluator(0, p1, p2)
	s := ev.Evaluate(context.Background())
	if s.Overall != StatusUnhealthy {
		t.Fatalf("expected unhealthy overall got %s", s.Overall)
	}
}
