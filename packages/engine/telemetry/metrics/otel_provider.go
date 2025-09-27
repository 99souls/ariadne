package metrics

// Iteration 6: Initial OpenTelemetry metrics bridge implementing the existing
// Provider interface. This keeps Ariadne's internal abstraction stable while
// allowing downstream deployments to opt into OTEL exporters / processors in
// later sub-iterations. Current scope: counters, gauges, histograms, timers.
// Gauges simulate Set semantics via an UpDownCounter delta application.

import (
    "context"
    "sync"
    "sync/atomic"
    "time"

    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/metric"
)

type OTelProviderOptions struct {
    ServiceName string // reserved for future resource attribution
}

// NewOTelProvider returns a metrics.Provider backed by an OTEL MeterProvider.
// Exporters, views, and resource attributes can be layered on by callers using
// the returned SDK provider (future extension). For now we keep zero-config.
func NewOTelProvider(opts OTelProviderOptions) Provider {
    mp := sdkmetric.NewMeterProvider()
    meter := mp.Meter("ariadne")
    return &otelProvider{mp: mp, meter: meter}
}

type otelProvider struct {
    mp    *sdkmetric.MeterProvider
    meter metric.Meter
}

func (p *otelProvider) NewCounter(opts CounterOpts) Counter {
    name := buildOTelName(opts.CommonOpts)
    inst, err := p.meter.Float64Counter(name, metric.WithDescription(opts.Help))
    if err != nil { return noopCounter{} }
    return &otelCounter{c: inst}
}
func (p *otelProvider) NewGauge(opts GaugeOpts) Gauge {
    name := buildOTelName(opts.CommonOpts)
    inst, err := p.meter.Float64UpDownCounter(name, metric.WithDescription(opts.Help))
    if err != nil { return noopGauge{} }
    return &otelGauge{g: inst}
}
func (p *otelProvider) NewHistogram(opts HistogramOpts) Histogram {
    name := buildOTelName(opts.CommonOpts)
    inst, err := p.meter.Float64Histogram(name, metric.WithDescription(opts.Help))
    if err != nil { return noopHistogram{} }
    return &otelHistogram{h: inst}
}
func (p *otelProvider) NewTimer(h HistogramOpts) func() Timer {
    hist := p.NewHistogram(HistogramOpts{CommonOpts: h.CommonOpts, Buckets: h.Buckets})
    return func() Timer { return &otelTimer{h: hist, start: time.Now()} }
}
func (p *otelProvider) Health(ctx context.Context) error { return nil }

// buildOTelName composes namespace/subsystem/name using '.' separators (OTEL convention tolerant).
func buildOTelName(c CommonOpts) string {
    if c.Namespace != "" && c.Subsystem != "" { return c.Namespace + "." + c.Subsystem + "." + c.Name }
    if c.Namespace != "" { if c.Name != "" { return c.Namespace + "." + c.Name }; return c.Namespace }
    if c.Subsystem != "" { if c.Name != "" { return c.Subsystem + "." + c.Name }; return c.Subsystem }
    return c.Name
}

// Instrument implementations -------------------------------------------------

type otelCounter struct { c metric.Float64Counter }
func (c *otelCounter) Inc(delta float64, labels ...string) { if delta > 0 { c.c.Add(context.Background(), delta) } }

type otelGauge struct {
    g     metric.Float64UpDownCounter
    value atomic.Value // float64
    mu    sync.Mutex
}
func (g *otelGauge) Set(v float64, labels ...string) {
    g.mu.Lock()
    prev, _ := g.value.Load().(float64)
    diff := v - prev
    g.value.Store(v)
    g.mu.Unlock()
    if diff != 0 { g.g.Add(context.Background(), diff) }
}
func (g *otelGauge) Add(delta float64, labels ...string) {
    if delta == 0 { return }
    g.mu.Lock()
    prev, _ := g.value.Load().(float64)
    g.value.Store(prev + delta)
    g.mu.Unlock()
    g.g.Add(context.Background(), delta)
}

type otelHistogram struct { h metric.Float64Histogram }
func (h *otelHistogram) Observe(value float64, labels ...string) { h.h.Record(context.Background(), value) }

type otelTimer struct { h Histogram; start time.Time }
func (t *otelTimer) ObserveDuration(labels ...string) { t.h.Observe(time.Since(t.start).Seconds(), labels...) }
