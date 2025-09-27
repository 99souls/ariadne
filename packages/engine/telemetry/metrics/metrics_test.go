package metrics

import (
    "net/http/httptest"
    "testing"
)

func TestNoopProviderBasic(t *testing.T) {
    p := NewNoopProvider()
    c := p.NewCounter(CounterOpts{ CommonOpts: CommonOpts{Name:"test_counter"} })
    g := p.NewGauge(GaugeOpts{ CommonOpts: CommonOpts{Name:"test_gauge"} })
    h := p.NewHistogram(HistogramOpts{ CommonOpts: CommonOpts{Name:"test_hist"} })
    timerCtor := p.NewTimer(HistogramOpts{ CommonOpts: CommonOpts{Name:"test_timer_seconds"} })

    c.Inc(5)
    g.Set(10)
    g.Add(-3)
    h.Observe(123)
    timer := timerCtor()
    timer.ObserveDuration()
}

func TestPrometheusProviderRegistration(t *testing.T) {
    p := NewPrometheusProvider(PrometheusProviderOptions{})
    c := p.NewCounter(CounterOpts{ CommonOpts: CommonOpts{Name:"events_total", Help:"total events", Labels: []string{"type"}} })
    c.Inc(1, "test")

    // ensure handler renders without panic
    rr := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/metrics", nil)
    p.MetricsHandler().ServeHTTP(rr, req)
    if rr.Code != 200 { t.Fatalf("expected 200, got %d", rr.Code) }
    if len(rr.Body.Bytes()) == 0 { t.Fatal("expected some metrics output") }
}
