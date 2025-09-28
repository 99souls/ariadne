package metrics

// INTERNAL: Metrics provider abstraction + built-in implementations (Prometheus, OTEL, noop).
// Public telemetry/metrics package was removed in C8 to shrink API surface; embedders now
// select backend solely via engine.Config (MetricsEnabled + MetricsBackend) and obtain any
// HTTP exposition through Engine.MetricsHandler().

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Provider is the minimal metrics provider contract used internally.
type Provider interface {
	NewCounter(opts CounterOpts) Counter
	NewGauge(opts GaugeOpts) Gauge
	NewHistogram(opts HistogramOpts) Histogram
	NewTimer(h HistogramOpts) func() Timer
	Health(ctx context.Context) error
}

type Counter interface {
	Inc(delta float64, labels ...string)
}
type Gauge interface {
	Set(v float64, labels ...string)
	Add(delta float64, labels ...string)
}
type Histogram interface {
	Observe(v float64, labels ...string)
}
type Timer interface{ ObserveDuration(labels ...string) }

type CommonOpts struct {
	Namespace, Subsystem, Name, Help string
	Labels                           []string
}
type CounterOpts struct{ CommonOpts }
type GaugeOpts struct{ CommonOpts }
type HistogramOpts struct {
	CommonOpts
	Buckets []float64
}

// ----------------------- Noop Provider ------------------------------------
type noopProvider struct{}
type noopCounter struct{}
type noopGauge struct{}
type noopHistogram struct{}
type noopTimer struct{}

func NewNoopProvider() Provider                              { return &noopProvider{} }
func (p *noopProvider) NewCounter(CounterOpts) Counter       { return noopCounter{} }
func (p *noopProvider) NewGauge(GaugeOpts) Gauge             { return noopGauge{} }
func (p *noopProvider) NewHistogram(HistogramOpts) Histogram { return noopHistogram{} }
func (p *noopProvider) NewTimer(HistogramOpts) func() Timer {
	return func() Timer { return noopTimer{} }
}
func (p *noopProvider) Health(context.Context) error { return nil }
func (noopCounter) Inc(float64, ...string)           {}
func (noopGauge) Set(float64, ...string)             {}
func (noopGauge) Add(float64, ...string)             {}
func (noopHistogram) Observe(float64, ...string)     {}
func (noopTimer) ObserveDuration(...string)          {}

// ----------------------- Prometheus Provider ------------------------------
var metricNameRE = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

type PrometheusProvider struct {
	reg        *prom.Registry
	mu         sync.RWMutex
	counters   map[string]*prom.CounterVec
	gauges     map[string]*prom.GaugeVec
	histograms map[string]*prom.HistogramVec
	problems   []error

	cardinality map[string]map[string]struct{}
	cardLimit   int

	exceededOnce map[string]struct{}
	warnCounter  *prom.CounterVec

	handler http.Handler
}

type PrometheusProviderOptions struct {
	Registry         *prom.Registry
	CardinalityLimit int
}

func NewPrometheusProvider(opts PrometheusProviderOptions) *PrometheusProvider {
	reg := opts.Registry
	if reg == nil {
		reg = prom.NewRegistry()
	}
	limit := opts.CardinalityLimit
	if limit <= 0 {
		limit = 100
	}
	warn := prom.NewCounterVec(prom.CounterOpts{Name: "ariadne_internal_cardinality_exceeded_total", Help: "count of metrics whose label cardinality exceeded limit"}, []string{"metric"})
	_ = reg.Register(warn)
	p := &PrometheusProvider{
		reg:          reg,
		counters:     make(map[string]*prom.CounterVec),
		gauges:       make(map[string]*prom.GaugeVec),
		histograms:   make(map[string]*prom.HistogramVec),
		cardinality:  make(map[string]map[string]struct{}),
		cardLimit:    limit,
		exceededOnce: make(map[string]struct{}),
		warnCounter:  warn,
		handler:      promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
	}
	return p
}

func (p *PrometheusProvider) MetricsHandler() http.Handler { return p.handler }

func (p *PrometheusProvider) buildFQName(c CommonOpts) (string, error) {
	if c.Name == "" {
		return "", errors.New("metric name required")
	}
	parts := []string{}
	if c.Namespace != "" {
		parts = append(parts, c.Namespace)
	}
	if c.Subsystem != "" {
		parts = append(parts, c.Subsystem)
	}
	parts = append(parts, c.Name)
	fq := parts[0]
	for i := 1; i < len(parts); i++ {
		fq += "_" + parts[i]
	}
	if !metricNameRE.MatchString(fq) {
		return "", fmt.Errorf("invalid metric name: %s", fq)
	}
	return fq, nil
}

func (p *PrometheusProvider) NewCounter(opts CounterOpts) Counter {
	fq, err := p.buildFQName(opts.CommonOpts)
	if err != nil {
		return noopCounter{}
	}
	p.mu.RLock()
	cv := p.counters[fq]
	p.mu.RUnlock()
	if cv != nil {
		return &promCounter{cv: cv, provider: p, id: fq}
	}
	vec := prom.NewCounterVec(prom.CounterOpts{Name: fq, Help: opts.Help}, opts.Labels)
	if err := p.reg.Register(vec); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			vec = are.ExistingCollector.(*prom.CounterVec)
		} else {
			p.recordProblem(err)
			return noopCounter{}
		}
	}
	p.mu.Lock()
	p.counters[fq] = vec
	p.mu.Unlock()
	return &promCounter{cv: vec, provider: p, id: fq}
}
func (p *PrometheusProvider) NewGauge(opts GaugeOpts) Gauge {
	fq, err := p.buildFQName(opts.CommonOpts)
	if err != nil {
		return noopGauge{}
	}
	p.mu.RLock()
	gv := p.gauges[fq]
	p.mu.RUnlock()
	if gv != nil {
		return &promGauge{gv: gv, provider: p, id: fq}
	}
	vec := prom.NewGaugeVec(prom.GaugeOpts{Name: fq, Help: opts.Help}, opts.Labels)
	if err := p.reg.Register(vec); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			vec = are.ExistingCollector.(*prom.GaugeVec)
		} else {
			p.recordProblem(err)
			return noopGauge{}
		}
	}
	p.mu.Lock()
	p.gauges[fq] = vec
	p.mu.Unlock()
	return &promGauge{gv: vec, provider: p, id: fq}
}
func (p *PrometheusProvider) NewHistogram(opts HistogramOpts) Histogram {
	fq, err := p.buildFQName(opts.CommonOpts)
	if err != nil {
		return noopHistogram{}
	}
	p.mu.RLock()
	hv := p.histograms[fq]
	p.mu.RUnlock()
	if hv != nil {
		return &promHistogram{hv: hv, provider: p, id: fq}
	}
	buckets := opts.Buckets
	if len(buckets) == 0 {
		buckets = prom.DefBuckets
	}
	vec := prom.NewHistogramVec(prom.HistogramOpts{Name: fq, Help: opts.Help, Buckets: buckets}, opts.Labels)
	if err := p.reg.Register(vec); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			vec = are.ExistingCollector.(*prom.HistogramVec)
		} else {
			p.recordProblem(err)
			return noopHistogram{}
		}
	}
	p.mu.Lock()
	p.histograms[fq] = vec
	p.mu.Unlock()
	return &promHistogram{hv: vec, provider: p, id: fq}
}
func (p *PrometheusProvider) NewTimer(h HistogramOpts) func() Timer {
	hist := p.NewHistogram(h)
	return func() Timer { return &promTimer{hist: hist, start: time.Now()} }
}
func (p *PrometheusProvider) Health(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.problems) == 0 {
		return nil
	}
	return fmt.Errorf("prometheus provider encountered %d problems (first: %v)", len(p.problems), p.problems[0])
}
func (p *PrometheusProvider) recordProblem(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.problems = append(p.problems, err)
}
func (p *PrometheusProvider) cardinalityTrack(id string, labelValues []string) {
	if p.cardLimit <= 0 {
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
				if p.warnCounter != nil {
					p.warnCounter.WithLabelValues(id).Inc()
				}
				fmt.Printf("[telemetry][prom] WARNING metric %s exceeded cardinality limit (%d)\n", id, p.cardLimit)
			}
		}
	}
}

type promCounter struct {
	cv       *prom.CounterVec
	provider *PrometheusProvider
	id       string
}

func (c *promCounter) Inc(delta float64, labels ...string) {
	if delta <= 0 {
		return
	}
	c.provider.cardinalityTrack(c.id, labels)
	c.cv.WithLabelValues(labels...).Add(delta)
}

type promGauge struct {
	gv       *prom.GaugeVec
	provider *PrometheusProvider
	id       string
}

func (g *promGauge) Set(value float64, labels ...string) {
	g.provider.cardinalityTrack(g.id, labels)
	g.gv.WithLabelValues(labels...).Set(value)
}
func (g *promGauge) Add(delta float64, labels ...string) {
	if delta == 0 {
		return
	}
	g.provider.cardinalityTrack(g.id, labels)
	g.gv.WithLabelValues(labels...).Add(delta)
}

type promHistogram struct {
	hv       *prom.HistogramVec
	provider *PrometheusProvider
	id       string
}

func (h *promHistogram) Observe(value float64, labels ...string) {
	h.provider.cardinalityTrack(h.id, labels)
	h.hv.WithLabelValues(labels...).Observe(value)
}

type promTimer struct {
	hist  Histogram
	start time.Time
}

func (t *promTimer) ObserveDuration(labels ...string) {
	t.hist.Observe(time.Since(t.start).Seconds(), labels...)
}

// ----------------------- OpenTelemetry Provider ---------------------------
type OTelProviderOptions struct {
	ServiceName      string
	CardinalityLimit int
}

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
	mp           *sdkmetric.MeterProvider
	meter        metric.Meter
	mu           sync.Mutex
	cardinality  map[string]map[string]struct{}
	cardLimit    int
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
	value     atomic.Value
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
func (p *otelProvider) cardinalityTrack(id string, labelValues []string) {
	if p.cardLimit <= 0 {
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
				if p.warnCounter != nil {
					p.warnCounter.Add(context.Background(), 1, metric.WithAttributes(attribute.String("metric", id)))
				}
			}
		}
	}
}
