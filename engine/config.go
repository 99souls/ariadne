package engine

import (
	"time"

	engpipeline "github.com/99souls/ariadne/engine/internal/pipeline"
	"github.com/99souls/ariadne/engine/models"
	engresources "github.com/99souls/ariadne/engine/resources"
)

// Config is the public configuration surface for the Engine facade. It intentionally
// narrows and normalizes underlying component configs while allowing advanced
// callers to inject custom implementations via functional options.
type Config struct {
	// Worker settings
	DiscoveryWorkers  int
	ExtractionWorkers int
	ProcessingWorkers int
	OutputWorkers     int
	BufferSize        int

	// Retry policy
	RetryBaseDelay   time.Duration
	RetryMaxDelay    time.Duration
	RetryMaxAttempts int

	// Adaptive rate limiting
	RateLimit models.RateLimitConfig

	// Resource management
	Resources engresources.Config

	// Resume settings
	Resume         bool
	CheckpointPath string // Overrides Resources.CheckpointPath if set

	// AssetPolicy defines behavior for asset handling (Phase 5D). Additive; if disabled
	// the processor behaves as legacy (no strategy invocation). Wiring occurs in Phase 5D iterations.
	AssetPolicy AssetPolicy

	// --- Phase 5E (Telemetry) incremental surface ---
	// MetricsEnabled toggles the new metrics provider wiring (prometheus export) when true.
	// Default remains false to avoid changing existing behavior unless explicitly enabled.
	MetricsEnabled bool
	// PrometheusListenAddr optional address for metrics HTTP exposure (e.g. ":2112").
	// If empty and MetricsEnabled is true, metrics are still collected but caller must expose handler.
	PrometheusListenAddr string
	// MetricsBackend selects the implementation when MetricsEnabled is true. Supported:
	//   "prom" (default) - built-in Prometheus registry
	//   "otel"          - OpenTelemetry bridge (iteration 6 experimental)
	//   "noop"          - explicit no-op (overrides MetricsEnabled true)
	// Unknown values fall back to the default (prom).
	MetricsBackend string
}

// toPipelineConfig adapts the facade Config to the internal pipeline config.
// engineOptions are internal construction options resolved by New().
type engineOptions struct {
	limiter         interface{} // ratelimit.RateLimiter (avoid import cycle comments here)
	resourceManager *engresources.Manager
}

func (c Config) toPipelineConfig(opts engineOptions) *engpipeline.PipelineConfig {
	pc := &engpipeline.PipelineConfig{
		DiscoveryWorkers:  c.DiscoveryWorkers,
		ExtractionWorkers: c.ExtractionWorkers,
		ProcessingWorkers: c.ProcessingWorkers,
		OutputWorkers:     c.OutputWorkers,
		BufferSize:        c.BufferSize,
		RetryBaseDelay:    c.RetryBaseDelay,
		RetryMaxDelay:     c.RetryMaxDelay,
		RetryMaxAttempts:  c.RetryMaxAttempts,
		RateLimiter:       nil,
		ResourceManager:   opts.resourceManager,
	}
	if rl, ok := opts.limiter.(interface {
		Acquire(ctx interface{}, domain string) (interface{}, error)
	}); ok {
		_ = rl // placeholder to suppress unused warning if build tags differ
	}
	return pc
}

// Defaults returns a Config with reasonable defaults.
func Defaults() Config {
	return Config{
		DiscoveryWorkers:  2,
		ExtractionWorkers: 4,
		ProcessingWorkers: 2,
		OutputWorkers:     1,
		BufferSize:        128,
		RetryBaseDelay:    200 * time.Millisecond,
		RetryMaxDelay:     5 * time.Second,
		RetryMaxAttempts:  3,
		RateLimit: models.RateLimitConfig{
			Enabled:                  true,
			InitialRPS:               2.0,
			MinRPS:                   0.25,
			MaxRPS:                   8.0,
			TokenBucketCapacity:      4.0,
			AIMDIncrease:             0.25,
			AIMDDecrease:             0.5,
			LatencyTarget:            1 * time.Second,
			LatencyDegradeFactor:     2.0,
			ErrorRateThreshold:       0.4,
			MinSamplesToTrip:         10,
			ConsecutiveFailThreshold: 5,
			OpenStateDuration:        15 * time.Second,
			HalfOpenProbes:           3,
			RetryBaseDelay:           200 * time.Millisecond,
			RetryMaxDelay:            5 * time.Second,
			RetryMaxAttempts:         3,
			StatsWindow:              30 * time.Second,
			StatsBucket:              2 * time.Second,
			DomainStateTTL:           2 * time.Minute,
			Shards:                   16,
		},
		Resources: engresources.Config{
			CacheCapacity:      64,
			MaxInFlight:        16,
			CheckpointInterval: 50 * time.Millisecond,
		},
		AssetPolicy: AssetPolicy{ // conservative defaults
			Enabled:        false,           // off until strategy implemented
			MaxBytes:       5 * 1024 * 1024, // 5MB per page aggregate cap (initial placeholder)
			MaxPerPage:     64,
			InlineMaxBytes: 2048,
			Optimize:       false,
			RewritePrefix:  "/assets/",
			AllowTypes:     []string{"img", "script", "stylesheet"},
			MaxConcurrent:  4, // Iteration 7: default worker pool size
		},
		// Telemetry defaults (Phase 5E): remain disabled to preserve prior footprint
		MetricsEnabled:       false,
		PrometheusListenAddr: "",
		MetricsBackend:       "prom",
	}
}

// AssetPolicy configures the asset subsystem when enabled. Iteration 1 surface; enforcement &
// validation logic comes in later iterations.
type AssetPolicy struct {
	Enabled        bool
	MaxBytes       int64
	MaxPerPage     int
	InlineMaxBytes int64
	Optimize       bool
	RewritePrefix  string
	AllowTypes     []string
	BlockTypes     []string
	MaxConcurrent  int // Iteration 7: parallel Execute worker count (>=1). 0 => auto
}
