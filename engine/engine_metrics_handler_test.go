package engine

import "testing"

// TestMetricsHandlerAvailability verifies that the facade only exposes a non-nil
// HTTP handler when metrics are enabled and the backend is Prometheus. Other
// backends (otel, noop) or disabled metrics should yield a nil handler.
func TestMetricsHandlerAvailability(t *testing.T) {
    cases := []struct {
        name          string
        enabled       bool
        backend       string
        expectHandler bool
    }{
        {name: "disabled_prom", enabled: false, backend: "prometheus", expectHandler: false},
        {name: "enabled_prom", enabled: true, backend: "prometheus", expectHandler: true},
        {name: "enabled_prom_short", enabled: true, backend: "prom", expectHandler: true},
        {name: "enabled_otel", enabled: true, backend: "otel", expectHandler: false},
        {name: "enabled_noop", enabled: true, backend: "noop", expectHandler: false},
        {name: "enabled_unknown_defaults_to_prom", enabled: true, backend: "", expectHandler: true},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            cfg := Defaults()
            cfg.MetricsEnabled = tc.enabled
            cfg.MetricsBackend = tc.backend
            eng, err := New(cfg)
            if err != nil {
                t.Fatalf("engine new: %v", err)
            }
            h := eng.MetricsHandler()
            if (h != nil) != tc.expectHandler {
                t.Fatalf("expected handler presence=%v got %v (enabled=%v backend=%s)", tc.expectHandler, h != nil, tc.enabled, tc.backend)
            }
        })
    }
}
