package metrics

import "testing"

func TestOTelProviderBasic(t *testing.T) {
    p := NewOTelProvider(OTelProviderOptions{})
    c := p.NewCounter(CounterOpts{CommonOpts: CommonOpts{Name: "otel_test_counter"}})
    c.Inc(1)
    g := p.NewGauge(GaugeOpts{CommonOpts: CommonOpts{Name: "otel_test_gauge"}})
    g.Set(10)
    g.Add(5)
    h := p.NewHistogram(HistogramOpts{CommonOpts: CommonOpts{Name: "otel_test_hist"}})
    h.Observe(1.5)
    ctor := p.NewTimer(HistogramOpts{CommonOpts: CommonOpts{Name: "otel_test_timer"}})
    tm := ctor()
    tm.ObserveDuration()
    // No panic implies success for initial bridge.
}
