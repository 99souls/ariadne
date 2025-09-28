package engine

import (
	"context"
	"testing"
	"time"

	telemetryhealth "github.com/99souls/ariadne/engine/telemetry/health"
)

// TestHealthChangeEvent verifies that a health_change event is emitted on status transition.
func TestHealthChangeEvent(t *testing.T) {
	cfg := Defaults()
	cfg.MetricsEnabled = false // we don't need metrics provider for event test
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("engine construction failed: %v", err)
	}
	ch := make(chan TelemetryEvent, 4)
	e.RegisterEventObserver(func(ev TelemetryEvent){
		if ev.Category == "health" && ev.Type == "health_change" {
			select { case ch <- ev: default: }
		}
	})
	current := telemetryhealth.StatusHealthy
	probe := telemetryhealth.ProbeFunc(func(ctx context.Context) telemetryhealth.ProbeResult {
		return telemetryhealth.ProbeResult{Name: "test", Status: current, CheckedAt: time.Now()}
	})
	// Replace evaluator with short TTL to enable status transition detection
	e.healthEval = telemetryhealth.NewEvaluator(10*time.Millisecond, probe)

	// First snapshot establishes baseline; no event expected for initial set.
	first := e.HealthSnapshot(context.Background())
	if first.Overall != telemetryhealth.StatusHealthy {
		t.Fatalf("expected first overall healthy got %s", first.Overall)
	}
	select {
	case ev := <-ch:
		t.Fatalf("unexpected event on initial snapshot: %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}

	// Change status to degraded triggers an event.
	current = telemetryhealth.StatusDegraded
	time.Sleep(15 * time.Millisecond) // exceed TTL
	second := e.HealthSnapshot(context.Background())
	if second.Overall != telemetryhealth.StatusDegraded {
		t.Fatalf("expected second overall degraded got %s", second.Overall)
	}
	// Allow brief scheduling window
	time.Sleep(10 * time.Millisecond)
	select {
	case ev := <-ch:
		if ev.Category != "health" || ev.Type != "health_change" {
			 t.Fatalf("unexpected event: %+v", ev)
		}
		if ev.Fields["previous"] != string(telemetryhealth.StatusHealthy) || ev.Fields["current"] != string(telemetryhealth.StatusDegraded) {
			 t.Fatalf("unexpected field transition: %+v", ev.Fields)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected health_change event not received")
	}
}
