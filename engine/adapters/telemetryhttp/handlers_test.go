package telemetryhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	telemetryhealth "github.com/99souls/ariadne/engine/telemetry/health"
)

type healthPayload struct {
	Overall string `json:"overall"`
	Ready   *bool  `json:"ready,omitempty"`
}

// stubHealthSource implements HealthSource with a controllable snapshot.
type stubHealthSource struct{ snap telemetryhealth.Snapshot }

func (s *stubHealthSource) setStatus(st telemetryhealth.Status) {
	s.snap = telemetryhealth.Snapshot{Overall: st, Generated: time.Now(), TTL: 5 * time.Millisecond}
}

func (s *stubHealthSource) HealthSnapshot(ctx context.Context) telemetryhealth.Snapshot {
	return s.snap
}

func TestHealthHandlerBasic_Migrated(t *testing.T) {
	src := &stubHealthSource{}
	src.setStatus(telemetryhealth.StatusHealthy)
	h := NewHealthHandler(HealthHandlerOptions{Source: src, IncludeProbes: true})
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var payload healthPayload
	_ = json.Unmarshal(w.Body.Bytes(), &payload)
	if payload.Overall != "healthy" {
		t.Fatalf("expected healthy got %s", payload.Overall)
	}
}

func TestReadinessHandler_Migrated(t *testing.T) {
	src := &stubHealthSource{}
	src.setStatus(telemetryhealth.StatusUnhealthy)
	h := NewReadinessHandler(HealthHandlerOptions{Source: src})
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r)
	if w1.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 got %d", w1.Code)
	}
	src.setStatus(telemetryhealth.StatusHealthy)
	time.Sleep(6 * time.Millisecond)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w2.Code)
	}
}
