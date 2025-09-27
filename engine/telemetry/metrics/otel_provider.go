package metrics

// Iteration 6: Initial OpenTelemetry metrics bridge implementing the existing
// Provider interface. This keeps Ariadne's internal abstraction stable while
// allowing downstream deployments to opt into OTEL exporters / processors in
// later sub-iterations. Current scope: counters, gauges, histograms, timers.
// Gauges simulate Set semantics via an UpDownCounter delta application.

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type OTelProviderOptions struct {
	ServiceName      string // reserved for future resource attribution
	CardinalityLimit int    // warn threshold like prom provider (0 => default 100)
}

// NewOTelProvider returns a metrics.Provider backed by an OTEL MeterProvider.
// Exporters, views, and resource attributes can be layered on by callers using
// the returned SDK provider (future extension). For now we keep zero-config.
func NewOTelProvider(opts OTelProviderOptions) Provider {
	mp := sdkmetric.NewMeterProvider()
	meter := mp.Meter("ariadne")
	limit := opts.CardinalityLimit
	if limit <= 0 {
		limit = 100
	}
	warnCtr, _ := meter.Float64Counter("ariadne.internal.cardinality_exceeded.total", metric.WithDescription("count of metrics whose label cardinality exceeded limit (mirrors Prometheus counter)"))
	return &otelProvider{mp: mp, meter: meter, cardLimit: limit, cardinality: make(map[string]map[string]struct{}), exceededOnce: make(map[string]struct{}), warnCounter: warnCtr}
}

type otelProvider struct {
	mp    *sdkmetric.MeterProvider
	meter metric.Meter

	mu          sync.Mutex
	cardinality map[string]map[string]struct{} // metric name -> distinct label value combos
	cardLimit   int

	exceededOnce map[string]struct{}
	warnCounter  metric.Float64Counter
}

func (p *otelProvider) NewCounter(opts CounterOpts) Counter {
	name := buildOTelName(opts.CommonOpts)
	inst, err := p.meter.Float64Counter(name, metric.WithDescription(opts.Help))
	if err != nil {
		return noopCounter{}
	}
	return &otelCounter{c: inst, labelKeys: opts.Labels, provider: p, id: name}
}
func (p *otelProvider) NewGauge(opts GaugeOpts) Gauge {
	name := buildOTelName(opts.CommonOpts)
	inst, err := p.meter.Float64UpDownCounter(name, metric.WithDescription(opts.Help))
	if err != nil {
		return noopGauge{}
	}
	return &otelGauge{g: inst, labelKeys: opts.Labels, provider: p, id: name}
}
func (p *otelProvider) NewHistogram(opts HistogramOpts) Histogram {
	name := buildOTelName(opts.CommonOpts)
	inst, err := p.meter.Float64Histogram(name, metric.WithDescription(opts.Help))
	if err != nil {
		return noopHistogram{}
	}
	return &otelHistogram{h: inst, labelKeys: opts.Labels, provider: p, id: name}
}
func (p *otelProvider) NewTimer(h HistogramOpts) func() Timer {
	hist := p.NewHistogram(HistogramOpts{CommonOpts: h.CommonOpts, Buckets: h.Buckets})
	return func() Timer { return &otelTimer{h: hist, start: time.Now()} }
}
func (p *otelProvider) Health(ctx context.Context) error { return nil }

// buildOTelName composes namespace/subsystem/name using '.' separators (OTEL convention tolerant).
func buildOTelName(c CommonOpts) string {
	if c.Namespace != "" && c.Subsystem != "" {
		return c.Namespace + "." + c.Subsystem + "." + c.Name
	}
	if c.Namespace != "" {
		if c.Name != "" {
			return c.Namespace + "." + c.Name
		}
		return c.Namespace
	}
	if c.Subsystem != "" {
		if c.Name != "" {
			return c.Subsystem + "." + c.Name
		}
		return c.Subsystem
	}
	return c.Name
}

// Instrument implementations -------------------------------------------------

type otelCounter struct {
	c         metric.Float64Counter
	labelKeys []string
	provider  *otelProvider
	id        string
}

func (c *otelCounter) Inc(delta float64, labels ...string) {
	if delta <= 0 {
		return
	}
	c.provider.cardinalityTrack(c.id, labels)
	ctx := context.Background()
	if len(c.labelKeys) == 0 || len(labels) == 0 {
		c.c.Add(ctx, delta)
		return
	}
	attrs := toAttributes(c.labelKeys, labels)
	c.c.Add(ctx, delta, metric.WithAttributes(attrs...))
}

type otelGauge struct {
	g         metric.Float64UpDownCounter
	value     atomic.Value // float64
	mu        sync.Mutex
	labelKeys []string
	provider  *otelProvider
	id        string
}

func (g *otelGauge) Set(v float64, labels ...string) {
	g.mu.Lock()
	prev, _ := g.value.Load().(float64)
	diff := v - prev
	g.value.Store(v)
	g.mu.Unlock()
	if diff != 0 {
		g.provider.cardinalityTrack(g.id, labels)
		ctx := context.Background()
		if len(g.labelKeys) == 0 || len(labels) == 0 {
			g.g.Add(ctx, diff)
			return
		}
		attrs := toAttributes(g.labelKeys, labels)
		g.g.Add(ctx, diff, metric.WithAttributes(attrs...))
	}
}
func (g *otelGauge) Add(delta float64, labels ...string) {
	if delta == 0 {
		return
	}
	g.mu.Lock()
	prev, _ := g.value.Load().(float64)
	g.value.Store(prev + delta)
	g.mu.Unlock()
	g.provider.cardinalityTrack(g.id, labels)
	ctx := context.Background()
	if len(g.labelKeys) == 0 || len(labels) == 0 {
		g.g.Add(ctx, delta)
		return
	}
	attrs := toAttributes(g.labelKeys, labels)
	g.g.Add(ctx, delta, metric.WithAttributes(attrs...))
}

type otelHistogram struct {
	h         metric.Float64Histogram
	labelKeys []string
	provider  *otelProvider
	id        string
}

func (h *otelHistogram) Observe(value float64, labels ...string) {
	h.provider.cardinalityTrack(h.id, labels)
	ctx := context.Background()
	if len(h.labelKeys) == 0 || len(labels) == 0 {
		h.h.Record(ctx, value)
		return
	}
	attrs := toAttributes(h.labelKeys, labels)
	h.h.Record(ctx, value, metric.WithAttributes(attrs...))
}

type otelTimer struct {
	h     Histogram
	start time.Time
}

func (t *otelTimer) ObserveDuration(labels ...string) {
	t.h.Observe(time.Since(t.start).Seconds(), labels...)
}

// toAttributes converts parallel key/value slices into attribute KeyValues.
func toAttributes(keys, values []string) []attribute.KeyValue {
	n := len(keys)
	if len(values) < n {
		n = len(values)
	}
	if n == 0 {
		return nil
	}
	out := make([]attribute.KeyValue, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, attribute.String(keys[i], values[i]))
	}
	return out
}

// cardinalityTrack mirrors Prometheus provider logic (best effort) for OTEL backend.
func (p *otelProvider) cardinalityTrack(id string, labelValues []string) {
	if p.cardLimit <= 0 || len(labelValues) == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	set := p.cardinality[id]
	if set == nil {
		set = make(map[string]struct{})
		p.cardinality[id] = set
	}
	key := fmt.Sprint(labelValues)
	if _, ok := set[key]; !ok {
		set[key] = struct{}{}
		if len(set) > p.cardLimit {
			if _, warned := p.exceededOnce[id]; !warned {
				p.exceededOnce[id] = struct{}{}
				p.warnCounter.Add(context.Background(), 1, metric.WithAttributes(attribute.String("metric", id)))
				fmt.Printf("[telemetry][otel] WARNING metric %s exceeded cardinality limit (%d)\n", id, p.cardLimit)
			}
		}
	}
}
