package engine

import (
	"time"

	engpipeline "github.com/99souls/ariadne/engine/internal/pipeline"
	intresources "github.com/99souls/ariadne/engine/internal/resources"
	"github.com/99souls/ariadne/engine/models"
)

// ResourcesConfig is the public facade configuration for resource management.
// Experimental: Shape and semantics may change before v1.0.
// Mirrors internal/resources.Config but kept separate to permit future reduction.
type ResourcesConfig struct {
	CacheCapacity      int
	MaxInFlight        int
	SpillDirectory     string
	CheckpointPath     string
	CheckpointInterval time.Duration
}

func (rc ResourcesConfig) toInternal() intresources.Config {
	return intresources.Config{
		CacheCapacity:      rc.CacheCapacity,
		MaxInFlight:        rc.MaxInFlight,
		SpillDirectory:     rc.SpillDirectory,
		CheckpointPath:     rc.CheckpointPath,
		CheckpointInterval: rc.CheckpointInterval,
	}
}

// Config is the public configuration surface for the Engine facade.
// Experimental: Field set, names, and semantics may change before v1.0.
// Most fields are pass-through tuning knobs for underlying subsystems and
// will be reviewed for reduction / consolidation ahead of a stable baseline.
type Config struct {
	// DiscoveryWorkers controls the concurrency of the seed/url discovery stage.
	// Experimental: May be folded into a single unified worker setting.
	DiscoveryWorkers int
	// ExtractionWorkers controls HTML extraction / parsing concurrency.
	// Experimental.
	ExtractionWorkers int
	// ProcessingWorkers controls post-extraction processing (enrichment, classification).
	// Experimental.
	ProcessingWorkers int
	// OutputWorkers controls the number of workers writing results to sinks.
	// Experimental.
	OutputWorkers int
	// BufferSize tunes internal channel buffering between stages.
	// Experimental: Subject to removal if adaptive backpressure is introduced.
	BufferSize int

	// RetryBaseDelay is the initial backoff delay for transient fetch failures.
	// Experimental: Retry model may be replaced by policy struct.
	RetryBaseDelay time.Duration
	// RetryMaxDelay caps the exponential backoff delay.
	// Experimental.
	RetryMaxDelay time.Duration
	// RetryMaxAttempts caps the number of retry attempts for a single fetch.
	// Experimental.
	RetryMaxAttempts int

	// RateLimit configures adaptive per-domain rate limiting.
	// Experimental: Location may change (likely to move fully under ratelimit/).
	RateLimit models.RateLimitConfig

	// Resources configures in-memory caches and spill / checkpoint behavior.
	// Experimental: Will be narrowed to higher-level policy before v1.0; implementation hidden.
	Resources ResourcesConfig

	// Resume enables filtering of already-processed seeds based on a checkpoint file.
	// Experimental: Mechanism & file format may change.
	Resume bool
	// CheckpointPath overrides Resources.CheckpointPath when non-empty.
	// Experimental.
	CheckpointPath string

	// AssetPolicy defines behavior for asset handling (Phase 5D).
	// Experimental: Entire asset subsystem is under active iteration.
	AssetPolicy AssetPolicy

	// MetricsEnabled toggles metrics collection / instrumentation.
	// Experimental: May be replaced by a Telemetry struct.
	MetricsEnabled bool
	// PrometheusListenAddr optional HTTP listen address for a Prometheus scrape endpoint.
	// Experimental: CLI layer may become the canonical place to expose endpoints.
	PrometheusListenAddr string
	// MetricsBackend selects metrics implementation: "prom" (default), "otel", or "noop".
	// Experimental: Backend selection mechanism may change.
	MetricsBackend string
}

// toPipelineConfig adapts the facade Config to the internal pipeline config.
// engineOptions are internal construction options resolved by New().
type engineOptions struct {
	limiter         interface{} // ratelimit.RateLimiter (avoid import cycle comments here)
	resourceManager *intresources.Manager
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
// Defaults returns a Config with conservative starting values.
// Experimental: Returned default values may be tuned between minor versions pre-v1.
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
		Resources: ResourcesConfig{
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
