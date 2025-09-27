package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
)

var metricNameRE = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

// PrometheusProvider implements Provider backed by a Prometheus registry.
type PrometheusProvider struct {
	reg        *prom.Registry
	mu         sync.RWMutex
	counters   map[string]*prom.CounterVec
	gauges     map[string]*prom.GaugeVec
	histograms map[string]*prom.HistogramVec
	problems   []error

	// cardinality tracking (metric -> distinct label value set size)
	cardinality map[string]map[string]struct{}
	cardLimit   int

	exceededOnce map[string]struct{}
	warnCounter  *prom.CounterVec

	// http handler cached
	handler http.Handler
}

// PrometheusProviderOptions config
type PrometheusProviderOptions struct {
	Registry         *prom.Registry // optional custom registry
	CardinalityLimit int            // warn threshold; 0 => default 100
}

// NewPrometheusProvider creates a new provider.
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
	_ = reg.Register(warn) // ignore AlreadyRegisteredError; best effort
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

// MetricsHandler returns an HTTP handler exposing /metrics
func (p *PrometheusProvider) MetricsHandler() http.Handler { return p.handler }

func (p *PrometheusProvider) buildFQName(c CommonOpts) (string, error) {
	if c.Name == "" {
		return "", errors.New("metric name required")
	}
	// Prometheus fqname rules: optional namespace_subsystem_name
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
		// If already registered, retrieve existing
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
		// default latency buckets (in seconds) similar to prom http default
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
	// Optimization: create (or reuse) histogram once, then only capture start time per timer.
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

// cardinalityTrack records a label value combination, best effort only.
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

// Instrument wrappers --------------------------------------------------------

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
