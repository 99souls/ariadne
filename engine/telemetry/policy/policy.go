package policy

import "time"

// TelemetryPolicy centralizes runtime-tunable telemetry knobs. It is designed to be
// swapped atomically (callers hold an immutable snapshot pointer) to avoid locks
// on hot paths. All durations are expected to be positive; zero values fall back
// to defaults established in Default().
type TelemetryPolicy struct {
	Health  HealthPolicy
	Tracing TracingPolicy
	Events  EventBusPolicy
}

type HealthPolicy struct {
	// ProbeTTL controls snapshot caching; evaluations inside this window reuse last result.
	ProbeTTL time.Duration
	// PipelineMinSamples suppresses early noise until enough pages processed.
	PipelineMinSamples int
	// PipelineDegradedRatio threshold above which pipeline health becomes degraded.
	PipelineDegradedRatio float64
	// PipelineUnhealthyRatio threshold above which pipeline health becomes unhealthy.
	PipelineUnhealthyRatio float64
	// ResourceDegradedCheckpoint backlog size considered degraded.
	ResourceDegradedCheckpoint int
	// ResourceUnhealthyCheckpoint backlog size considered unhealthy.
	ResourceUnhealthyCheckpoint int
}

type TracingPolicy struct {
	// SamplePercent 0-100 range; >0 enables sampling (simple percentage sampler for now).
	SamplePercent float64
	// ErrorBoostPercent additional percentage added when previous span marked error (future hook).
	ErrorBoostPercent float64
	// LatencyBoostThresholdMs if last span duration >= threshold add LatencyBoostPercent sampling (future hook).
	LatencyBoostThresholdMs int64
	// LatencyBoostPercent additional percent sampling for high-latency spans.
	LatencyBoostPercent float64
}

type EventBusPolicy struct {
	// MaxSubscriberBuffer upper bound per subscriber channel; 0 -> default.
	MaxSubscriberBuffer int
}

// Default returns a TelemetryPolicy populated with the current heuristics previously
// hard-coded in engine.go (Iteration 4). Adjust carefully; downstream alerting may
// assume these semantics.
func Default() TelemetryPolicy {
	return TelemetryPolicy{
		Health: HealthPolicy{
			ProbeTTL:                    2 * time.Second,
			PipelineMinSamples:          10,
			PipelineDegradedRatio:       0.50,
			PipelineUnhealthyRatio:      0.80,
			ResourceDegradedCheckpoint:  256,
			ResourceUnhealthyCheckpoint: 512,
		},
		Tracing: TracingPolicy{SamplePercent: 20},
		Events:  EventBusPolicy{MaxSubscriberBuffer: 1024},
	}
}

// Normalize ensures sane bounds without mutating original; returns a cleaned copy.
func (p TelemetryPolicy) Normalize() TelemetryPolicy {
	c := p
	if c.Health.ProbeTTL <= 0 {
		c.Health.ProbeTTL = 2 * time.Second
	}
	if c.Health.PipelineMinSamples <= 0 {
		c.Health.PipelineMinSamples = 10
	}
	if c.Health.PipelineDegradedRatio <= 0 {
		c.Health.PipelineDegradedRatio = 0.50
	}
	if c.Health.PipelineUnhealthyRatio <= 0 {
		c.Health.PipelineUnhealthyRatio = 0.80
	}
	if c.Health.ResourceDegradedCheckpoint <= 0 {
		c.Health.ResourceDegradedCheckpoint = 256
	}
	if c.Health.ResourceUnhealthyCheckpoint <= 0 {
		c.Health.ResourceUnhealthyCheckpoint = 512
	}
	if c.Tracing.SamplePercent < 0 {
		c.Tracing.SamplePercent = 0
	}
	if c.Tracing.SamplePercent > 100 {
		c.Tracing.SamplePercent = 100
	}
	if c.Events.MaxSubscriberBuffer <= 0 {
		c.Events.MaxSubscriberBuffer = 1024
	}
	return c
}
