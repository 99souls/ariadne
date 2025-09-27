package metrics

import (
    "context"
    "testing"
    "time"
)

// BenchmarkIntegratedWorkload simulates a simplified crawl processing loop
// emitting representative metrics mix to approximate aggregate overhead.
// It is NOT a business correctness benchmark; only telemetry cost focus.
func BenchmarkIntegratedWorkload(b *testing.B) {
    providers := []struct {
        name string
        p    Provider
    }{
        {"noop", NewNoopProvider()},
        {"prom", NewPrometheusProvider(PrometheusProviderOptions{})},
        {"otel", NewOTelProvider(OTelProviderOptions{})},
    }
    for _, item := range providers {
        b.Run(item.name, func(b *testing.B) {
            // Simulate representative instruments
            pages := item.p.NewCounter(CounterOpts{CommonOpts: CommonOpts{Name: "bench_pages", Labels: []string{"outcome"}}})
            assets := item.p.NewCounter(CounterOpts{CommonOpts: CommonOpts{Name: "bench_assets", Labels: []string{"type"}}})
            failures := item.p.NewCounter(CounterOpts{CommonOpts: CommonOpts{Name: "bench_failures", Labels: []string{"error_class"}}})
            latencyHist := item.p.NewHistogram(HistogramOpts{CommonOpts: CommonOpts{Name: "bench_page_latency"}})
            timerCtor := item.p.NewTimer(HistogramOpts{CommonOpts: CommonOpts{Name: "bench_stage"}})
            ctx := context.Background()
            _ = ctx
            b.ReportAllocs()
            for i := 0; i < b.N; i++ {
                // Simulate a page with several asset operations and stages
                pages.Inc(1, "success")
                // per-page stages
                for s := 0; s < 3; s++ {
                    t := timerCtor()
                    // emulate small work
                    time.Sleep(time.Nanosecond)
                    t.ObserveDuration()
                }
                // assets
                for a := 0; a < 5; a++ {
                    assets.Inc(1, "image")
                }
                if i%50 == 0 { // occasional failure pattern
                    failures.Inc(1, "timeout")
                }
                latencyHist.Observe(float64((i%100))/1000.0) // synthetic duration
            }
        })
    }
}
