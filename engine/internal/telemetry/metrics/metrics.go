package metrics

// INTERNAL: Consolidated metrics provider abstraction moved from public telemetry/metrics.
// Public surface now selects backend via engine.Config (MetricsBackend) only; no direct
// provider construction by embedders pre-v1.0. We retain the minimal Provider interface
// required by internal subsystems (events bus, health gauges, pipeline instrumentation).

import "context"

// Provider is the minimal metrics provider contract used internally.
type Provider interface {
    NewCounter(opts CounterOpts) Counter
    NewGauge(opts GaugeOpts) Gauge
    NewHistogram(opts HistogramOpts) Histogram
    NewTimer(h HistogramOpts) func() Timer
    Health(ctx context.Context) error
}

type Counter interface{ Inc(delta float64, labels ...string) }
type Gauge interface{ Set(v float64, labels ...string); Add(delta float64, labels ...string) }
type Histogram interface{ Observe(v float64, labels ...string) }
type Timer interface{ ObserveDuration(labels ...string) }

type CommonOpts struct { Namespace, Subsystem, Name, Help string; Labels []string }
type CounterOpts struct{ CommonOpts }
type GaugeOpts struct{ CommonOpts }
type HistogramOpts struct { CommonOpts; Buckets []float64 }

// noop provider ------------------------------------------------------------------
type noopProvider struct{}
type noopCounter struct{}
type noopGauge struct{}
type noopHistogram struct{}
type noopTimer struct{}

func NewNoopProvider() Provider { return &noopProvider{} }
func (p *noopProvider) NewCounter(CounterOpts) Counter { return noopCounter{} }
func (p *noopProvider) NewGauge(GaugeOpts) Gauge { return noopGauge{} }
func (p *noopProvider) NewHistogram(HistogramOpts) Histogram { return noopHistogram{} }
func (p *noopProvider) NewTimer(HistogramOpts) func() Timer { return func() Timer { return noopTimer{} } }
func (p *noopProvider) Health(context.Context) error { return nil }
func (noopCounter) Inc(float64, ...string) {}
func (noopGauge) Set(float64, ...string) {}
func (noopGauge) Add(float64, ...string) {}
func (noopHistogram) Observe(float64, ...string) {}
func (noopTimer) ObserveDuration(...string) {}
