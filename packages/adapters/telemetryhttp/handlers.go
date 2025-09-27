package telemetryhttp

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/99souls/ariadne/engine"
	telemetryhealth "github.com/99souls/ariadne/engine/telemetry/health"
	telemetrymetrics "github.com/99souls/ariadne/engine/telemetry/metrics"
)

// HealthHandlerOptions configures health/readiness handlers.
type HealthHandlerOptions struct {
	Engine *engine.Engine
	// Include detailed probe list; if false only overall is returned.
	IncludeProbes bool
	// Clock override (tests); defaults to time.Now.
	Clock func() time.Time
}

// readiness state caching for faster probes if called very frequently.

type healthResponse struct {
	Overall   telemetryhealth.Status        `json:"overall"`
	Probes    []telemetryhealth.ProbeResult `json:"probes,omitempty"`
	Generated time.Time                     `json:"generated"`
	TTL       time.Duration                 `json:"ttl"`
	Ready     *bool                         `json:"ready,omitempty"`
	Previous  string                        `json:"previous,omitempty"`
	ChangedAt *time.Time                    `json:"changed_at,omitempty"`
}

type readinessTracker struct {
	lastStatus atomic.Value // string
	changedAt  atomic.Value // time.Time
}

func (rt *readinessTracker) update(cur string, now time.Time) (prev string, changedAt *time.Time) {
	pRaw := rt.lastStatus.Load()
	if pRaw != nil {
		prev = pRaw.(string)
	}
	if prev != cur {
		rt.lastStatus.Store(cur)
		nowCopy := now
		rt.changedAt.Store(nowCopy)
		return prev, &nowCopy
	}
	cRaw := rt.changedAt.Load()
	if cRaw != nil {
		cc := cRaw.(time.Time)
		changedAt = &cc
	}
	return prev, changedAt
}

var defaultTracker readinessTracker

// NewHealthHandler returns an http.Handler exposing overall health.
func NewHealthHandler(opts HealthHandlerOptions) http.Handler {
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if opts.Engine == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "engine nil"})
			return
		}
		snap := opts.Engine.HealthSnapshot(r.Context())
		prev, changedAt := defaultTracker.update(string(snap.Overall), opts.Clock())
		resp := healthResponse{Overall: snap.Overall, Generated: snap.Generated, TTL: snap.TTL}
		if opts.IncludeProbes {
			resp.Probes = snap.Probes
		}
		if prev != "" && prev != string(snap.Overall) {
			resp.Previous = prev
		}
		if changedAt != nil {
			resp.ChangedAt = changedAt
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // always 200 for /healthz (liveness)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// NewReadinessHandler returns readiness info with 503 when unhealthy or unknown.
func NewReadinessHandler(opts HealthHandlerOptions) http.Handler {
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if opts.Engine == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "engine nil"})
			return
		}
		snap := opts.Engine.HealthSnapshot(r.Context())
		prev, changedAt := defaultTracker.update(string(snap.Overall), opts.Clock())
		ready := snap.Overall == telemetryhealth.StatusHealthy || snap.Overall == telemetryhealth.StatusDegraded
		resp := healthResponse{Overall: snap.Overall, Generated: snap.Generated, TTL: snap.TTL, Ready: &ready}
		if opts.IncludeProbes {
			resp.Probes = snap.Probes
		}
		if prev != "" && prev != string(snap.Overall) {
			resp.Previous = prev
		}
		if changedAt != nil {
			resp.ChangedAt = changedAt
		}
		w.Header().Set("Content-Type", "application/json")
		if !ready || snap.Overall == telemetryhealth.StatusUnknown {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// NewMetricsHandler wraps the provider's HTTP handler (Prometheus) or returns 404 if provider nil.
func NewMetricsHandler(p telemetrymetrics.Provider) http.Handler {
	if p == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	}
	// Only Prometheus provider currently exposes MetricsHandler; attempt type assertion.
	if promP, ok := p.(interface{ MetricsHandler() http.Handler }); ok {
		return promP.MetricsHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "metrics handler unavailable", http.StatusNotImplemented)
	})
}
