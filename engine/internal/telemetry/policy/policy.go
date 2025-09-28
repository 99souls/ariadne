package policy

// INTERNAL: telemetry policy (moved in C6 step 2b). Public access now via engine.Policy()/UpdateTelemetryPolicy().

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
    ProbeTTL                    time.Duration
    PipelineMinSamples          int
    PipelineDegradedRatio       float64
    PipelineUnhealthyRatio      float64
    ResourceDegradedCheckpoint  int
    ResourceUnhealthyCheckpoint int
}

type TracingPolicy struct {
    SamplePercent          float64
    ErrorBoostPercent      float64
    LatencyBoostThresholdMs int64
    LatencyBoostPercent    float64
}

type EventBusPolicy struct {
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
    if c.Health.ProbeTTL <= 0 { c.Health.ProbeTTL = 2 * time.Second }
    if c.Health.PipelineMinSamples <= 0 { c.Health.PipelineMinSamples = 10 }
    if c.Health.PipelineDegradedRatio <= 0 { c.Health.PipelineDegradedRatio = 0.50 }
    if c.Health.PipelineUnhealthyRatio <= 0 { c.Health.PipelineUnhealthyRatio = 0.80 }
    if c.Health.ResourceDegradedCheckpoint <= 0 { c.Health.ResourceDegradedCheckpoint = 256 }
    if c.Health.ResourceUnhealthyCheckpoint <= 0 { c.Health.ResourceUnhealthyCheckpoint = 512 }
    if c.Tracing.SamplePercent < 0 { c.Tracing.SamplePercent = 0 }
    if c.Tracing.SamplePercent > 100 { c.Tracing.SamplePercent = 100 }
    if c.Events.MaxSubscriberBuffer <= 0 { c.Events.MaxSubscriberBuffer = 1024 }
    return c
}

