package telemetryhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/99souls/ariadne/engine"
	telemetryhealth "github.com/99souls/ariadne/engine/telemetry/health"
)

type healthPayload struct {
	Overall   string     `json:"overall"`
	Ready     *bool      `json:"ready,omitempty"`
	Previous  string     `json:"previous,omitempty"`
	ChangedAt *time.Time `json:"changed_at,omitempty"`
}

// buildEngineWithProbe replaces the engine's evaluator with one driven by the provided status pointer.
func buildEngineWithProbe(t *testing.T, status *telemetryhealth.Status) *engine.Engine {
	cfg := engine.Config{}
	e, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine new: %v", err)
	}
	probe := telemetryhealth.ProbeFunc(func(ctx context.Context) telemetryhealth.ProbeResult {
		return telemetryhealth.ProbeResult{Name: "synthetic", Status: *status, CheckedAt: time.Now()}
	})
	custom := telemetryhealth.NewEvaluator(10*time.Millisecond, probe)
	custom.ForceInvalidate()
	e.HealthEvaluatorForTest(custom)
	return e
}

func TestHealthHandlerBasic(t *testing.T) {
	st := telemetryhealth.StatusHealthy
	e := buildEngineWithProbe(t, &st)
	h := NewHealthHandler(HealthHandlerOptions{Engine: e, IncludeProbes: true})
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var payload healthPayload
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Overall != "healthy" {
		t.Fatalf("expected healthy got %s", payload.Overall)
	}
}

func TestReadinessHandlerTransitions(t *testing.T) {
	cur := telemetryhealth.StatusUnhealthy
	e := buildEngineWithProbe(t, &cur)
	h := NewReadinessHandler(HealthHandlerOptions{Engine: e})
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	// First call unhealthy -> 503
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r)
	if w1.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 got %d", w1.Code)
	}
	var p1 healthPayload
	_ = json.Unmarshal(w1.Body.Bytes(), &p1)
	if p1.Ready != nil && *p1.Ready {
		t.Fatalf("expected ready=false")
	}
	// Transition to degraded -> 200 readiness
	cur = telemetryhealth.StatusDegraded
	time.Sleep(12 * time.Millisecond) // exceed TTL
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 after improvement got %d", w2.Code)
	}
	var p2 healthPayload
	_ = json.Unmarshal(w2.Body.Bytes(), &p2)
	if p2.Ready == nil || !*p2.Ready {
		t.Fatalf("expected ready=true")
	}
	if p2.Previous != "unhealthy" {
		t.Fatalf("expected previous=unhealthy got %s", p2.Previous)
	}
	if p2.ChangedAt == nil {
		t.Fatalf("expected changed_at timestamp")
	}
}

// Sanity check status constants (API stability guard)
func TestStatusStrings(t *testing.T) {
	if telemetryhealth.StatusHealthy != "healthy" {
		t.Fatal("status mismatch")
	}
}
