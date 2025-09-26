package engine

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"site-scraper/internal/pipeline"
	"site-scraper/internal/ratelimit"
	"site-scraper/internal/resources"
	"site-scraper/pkg/models"
)

// Snapshot is a unified view of engine state (initial minimal subset).
type Snapshot struct {
	StartedAt time.Time                 `json:"started_at"`
	Uptime    time.Duration             `json:"uptime"`
	Pipeline  *pipeline.PipelineMetrics `json:"pipeline,omitempty"`
	Limiter   *ratelimit.LimiterSnapshot `json:"limiter,omitempty"`
	Resources *ResourceSnapshot         `json:"resources,omitempty"`
}

// ResourceSnapshot surfaces basic cache / spill / checkpoint telemetry.
type ResourceSnapshot struct {
	CacheEntries     int `json:"cache_entries"`
	SpillFiles       int `json:"spill_files"`
	InFlight         int `json:"in_flight"`
	CheckpointQueued int `json:"checkpoint_queued"`
}

// Engine composes the pipeline, limiter, and resource manager under a single facade.
type Engine struct {
	cfg      Config
	pl       *pipeline.Pipeline
	limiter  ratelimit.RateLimiter
	rm       *resources.Manager
	started  atomic.Bool
	startedAt time.Time
}

// Option functional option for customization.
type Option func(*Config)

// New constructs an Engine with supplied config and options.
func New(cfg Config, opts ...Option) (*Engine, error) {
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}

	// Build resource manager if configured
	var rm *resources.Manager
	if cfg.Resources.CacheCapacity > 0 || cfg.Resources.MaxInFlight > 0 || cfg.Resources.CheckpointPath != "" {
		manager, err := resources.NewManager(cfg.Resources)
		if err != nil {
			return nil, err
		}
		rm = manager
	}

	// Build rate limiter
	var limiter ratelimit.RateLimiter
	if cfg.RateLimit.Enabled {
		limiter = ratelimit.NewAdaptiveRateLimiter(cfg.RateLimit)
	}

	pc := (&cfg).toPipelineConfig(engineOptions{limiter: limiter, resourceManager: rm})
	pl := pipeline.NewPipeline(pc)

	e := &Engine{cfg: cfg, pl: pl, limiter: limiter, rm: rm, startedAt: time.Now()}
	e.started.Store(true)
	return e, nil
}

// Start begins processing of the provided seed URLs. It returns a read-only results channel.
func (e *Engine) Start(ctx context.Context, seeds []string) (<-chan *models.CrawlResult, error) {
	if !e.started.Load() {
		return nil, errors.New("engine not started")
	}
	// Underlying pipeline already started at construction; we just feed URLs now.
	results := e.pl.ProcessURLs(ctx, seeds)
	return results, nil
}

// Stop gracefully stops the engine and underlying components.
func (e *Engine) Stop() error {
	if e.pl != nil {
		e.pl.Stop()
	}
	if c, ok := e.limiter.(interface{ Close() error }); ok {
		_ = c.Close()
	}
	if e.rm != nil {
		_ = e.rm.Close()
	}
	return nil
}

// Snapshot returns a unified state view.
func (e *Engine) Snapshot() Snapshot {
	snap := Snapshot{StartedAt: e.startedAt}
	if e.startedAt.IsZero() {
		snap.StartedAt = time.Now()
	}
	snap.Uptime = time.Since(snap.StartedAt)
	if e.pl != nil {
		snap.Pipeline = e.pl.Metrics()
	}
	if e.limiter != nil {
		ls := e.limiter.Snapshot()
		snap.Limiter = &ls
	}
	if e.rm != nil {
		rs := e.rm.Stats()
		snap.Resources = &ResourceSnapshot{
			CacheEntries:     rs.CacheEntries,
			SpillFiles:       rs.SpillFiles,
			InFlight:         rs.InFlight,
			CheckpointQueued: rs.CheckpointQueued,
		}
	}
	return snap
}

