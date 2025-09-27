package metrics

import (
	"runtime"
	"testing"
	"time"
)

// BenchmarkProviderCounterInc compares overhead for basic increments among providers.
func BenchmarkProviderCounterInc(b *testing.B) {
	providers := []struct {
		name string
		p    Provider
	}{
		{"noop", NewNoopProvider()},
		{"prom", NewPrometheusProvider(PrometheusProviderOptions{})},
		{"otel", NewOTelProvider(OTelProviderOptions{})},
	}
	b.Logf("Go=%s NumCPU=%d", runtime.Version(), runtime.NumCPU())
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

// BenchmarkProviderHistogramObserve measures histogram record overhead.
func BenchmarkProviderHistogramObserve(b *testing.B) {
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
			h := item.p.NewHistogram(HistogramOpts{CommonOpts: CommonOpts{Name: "bench_hist"}})
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				h.Observe(float64(i%100) / 100.0)
			}
		})
	}
}

// BenchmarkProviderTimer measures timer start + observe duration overhead.
func BenchmarkProviderTimer(b *testing.B) {
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
			ctor := item.p.NewTimer(HistogramOpts{CommonOpts: CommonOpts{Name: "bench_timer"}})
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				t := ctor()
				// simulate small work
				time.Sleep(time.Nanosecond)
				t.ObserveDuration()
			}
		})
	}
}
