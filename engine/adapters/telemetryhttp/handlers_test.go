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

type healthPayload struct { Overall string `json:"overall"`; Ready *bool `json:"ready,omitempty"` }

func buildEngineWithProbe(t *testing.T, status *telemetryhealth.Status) *engine.Engine {
    cfg := engine.Config{}
    e, err := engine.New(cfg); if err != nil { t.Fatalf("engine new: %v", err) }
    probe := telemetryhealth.ProbeFunc(func(ctx context.Context) telemetryhealth.ProbeResult { return telemetryhealth.ProbeResult{Name:"synthetic", Status:*status, CheckedAt:time.Now()} })
    eval := telemetryhealth.NewEvaluator(5*time.Millisecond, probe)
    eval.ForceInvalidate()
    e.HealthEvaluatorForTest(eval)
    return e
}

func TestHealthHandlerBasic_Migrated(t *testing.T){
    st := telemetryhealth.StatusHealthy
    e := buildEngineWithProbe(t, &st)
    h := NewHealthHandler(HealthHandlerOptions{Engine:e, IncludeProbes:true})
    r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
    w := httptest.NewRecorder(); h.ServeHTTP(w,r)
    if w.Code != 200 { t.Fatalf("expected 200 got %d", w.Code) }
    var payload healthPayload; _ = json.Unmarshal(w.Body.Bytes(), &payload)
    if payload.Overall != "healthy" { t.Fatalf("expected healthy got %s", payload.Overall) }
}

func TestReadinessHandler_Migrated(t *testing.T){
    cur := telemetryhealth.StatusUnhealthy
    e := buildEngineWithProbe(t, &cur)
    h := NewReadinessHandler(HealthHandlerOptions{Engine:e})
    r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
    w1 := httptest.NewRecorder(); h.ServeHTTP(w1,r)
    if w1.Code != http.StatusServiceUnavailable { t.Fatalf("expected 503 got %d", w1.Code) }
    cur = telemetryhealth.StatusHealthy; time.Sleep(6*time.Millisecond)
    w2 := httptest.NewRecorder(); h.ServeHTTP(w2,r)
    if w2.Code != http.StatusOK { t.Fatalf("expected 200 got %d", w2.Code) }
}
