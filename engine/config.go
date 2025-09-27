package engine

import (
	engpipeline "ariadne/engine/pipeline"
	engresources "ariadne/engine/resources"
	"ariadne/pkg/models"
	"time"
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
	if rl, ok := opts.limiter.(interface{ Acquire(ctx interface{}, domain string) (interface{}, error) }); ok {
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
			Enabled:             true,
			InitialRPS:          2.0,
			MinRPS:              0.25,
			MaxRPS:              8.0,
			TokenBucketCapacity: 4.0,
			AIMDIncrease:        0.25,
			AIMDDecrease:        0.5,
			LatencyTarget:       1 * time.Second,
			LatencyDegradeFactor: 2.0,
			ErrorRateThreshold:       0.4,
			MinSamplesToTrip:         10,
			ConsecutiveFailThreshold: 5,
			OpenStateDuration:        15 * time.Second,
			HalfOpenProbes:           3,
			RetryBaseDelay:   200 * time.Millisecond,
			RetryMaxDelay:    5 * time.Second,
			RetryMaxAttempts: 3,
			StatsWindow:    30 * time.Second,
			StatsBucket:    2 * time.Second,
			DomainStateTTL: 2 * time.Minute,
			Shards:         16,
		},
		Resources: engresources.Config{
			CacheCapacity:      64,
			MaxInFlight:        16,
			CheckpointInterval: 50 * time.Millisecond,
		},
	}
}
