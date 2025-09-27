package metrics

import "context"

// Counter represents a monotonically increasing value.
type Counter interface {
	Inc(delta float64, labels ...string)
}

// Gauge represents a value that can go up or down.
type Gauge interface {
	Set(value float64, labels ...string)
	Add(delta float64, labels ...string)
}

// Histogram records observations into buckets and tracks count + sum.
type Histogram interface {
	Observe(value float64, labels ...string)
}

// Timer is a helper handle for measuring latency.
type Timer interface {
	// ObserveDuration records the time elapsed since the timer was created in seconds.
	ObserveDuration(labels ...string)
}

// Provider is the top-level metrics provider abstraction.
type Provider interface {
	NewCounter(opts CounterOpts) Counter
	NewGauge(opts GaugeOpts) Gauge
	NewHistogram(opts HistogramOpts) Histogram
	NewTimer(h HistogramOpts) func() Timer // returns a constructor that snapshots start time lazily
	// Health returns an error if provider is degraded/unhealthy (e.g., registration failures).
	Health(ctx context.Context) error
}

// Common option fields embedded into each metric option struct.
type CommonOpts struct {
	Namespace string   // logical grouping/prefix, optional
	Subsystem string   // secondary prefix, optional
	Name      string   // required base metric name (snake_case)
	Help      string   // human readable help text
	Labels    []string // label key list ordering defines the variadic value ordering
}

// CounterOpts options for counters.
type CounterOpts struct{ CommonOpts }

// GaugeOpts options for gauges.
type GaugeOpts struct{ CommonOpts }

// HistogramOpts options for histograms / timers.
type HistogramOpts struct {
	CommonOpts
	Buckets []float64 // optional custom bucket boundaries
}

// Noop implementations -------------------------------------------------------

type noopProvider struct{}

type noopCounter struct{}

type noopGauge struct{}

type noopHistogram struct{}

type noopTimer struct{}

// NewNoopProvider returns a provider that does nothing.
func NewNoopProvider() Provider { return &noopProvider{} }

func (p *noopProvider) NewCounter(opts CounterOpts) Counter       { return noopCounter{} }
func (p *noopProvider) NewGauge(opts GaugeOpts) Gauge             { return noopGauge{} }
func (p *noopProvider) NewHistogram(opts HistogramOpts) Histogram { return noopHistogram{} }
func (p *noopProvider) NewTimer(h HistogramOpts) func() Timer {
	return func() Timer { return noopTimer{} }
}
func (p *noopProvider) Health(ctx context.Context) error { return nil }

func (noopCounter) Inc(delta float64, labels ...string)       {}
func (noopGauge) Set(value float64, labels ...string)         {}
func (noopGauge) Add(delta float64, labels ...string)         {}
func (noopHistogram) Observe(value float64, labels ...string) {}
func (noopTimer) ObserveDuration(labels ...string)            {}
