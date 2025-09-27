package metrics

import "testing"

// BenchmarkProviderCounterInc compares overhead for basic increments among providers.
func BenchmarkProviderCounterInc(b *testing.B) {
    providers := []struct{ name string; p Provider }{
        {"noop", NewNoopProvider()},
        {"prom", NewPrometheusProvider(PrometheusProviderOptions{})},
        {"otel", NewOTelProvider(OTelProviderOptions{})},
    }
    for _, item := range providers {
        b.Run(item.name, func(b *testing.B) {
            c := item.p.NewCounter(CounterOpts{CommonOpts: CommonOpts{Name: "bench_counter"}})
            b.ReportAllocs()
            for i := 0; i < b.N; i++ {
                c.Inc(1)
            }
        })
    }
}
