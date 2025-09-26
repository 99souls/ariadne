package engine

import (
	"bufio"
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"site-scraper/internal/pipeline"
	engratelimit "site-scraper/packages/engine/ratelimit"
	"site-scraper/internal/resources"
	engmodels "site-scraper/packages/engine/models"
)

// Snapshot is a unified view of engine state (initial minimal subset).
type Snapshot struct {
	StartedAt time.Time                 `json:"started_at"`
	Uptime    time.Duration             `json:"uptime"`
	Pipeline  *pipeline.PipelineMetrics `json:"pipeline,omitempty"`
	Limiter   *engratelimit.LimiterSnapshot `json:"limiter,omitempty"`
	Resources *ResourceSnapshot         `json:"resources,omitempty"`
	Resume    *ResumeSnapshot           `json:"resume,omitempty"`
}

// ResourceSnapshot surfaces basic cache / spill / checkpoint telemetry.
type ResourceSnapshot struct {
    CacheEntries     int `json:"cache_entries"`
    SpillFiles       int `json:"spill_files"`
    InFlight         int `json:"in_flight"`
    CheckpointQueued int `json:"checkpoint_queued"`
}

// ResumeSnapshot exposes resume filtering counters.
type ResumeSnapshot struct {
    SeedsBefore int   `json:"seeds_before"`
    Skipped     int64 `json:"skipped"`
}

// Engine composes the pipeline, limiter, and resource manager under a single facade.
type Engine struct {
	cfg      Config
	pl       *pipeline.Pipeline
	limiter  engratelimit.RateLimiter
	rm       *resources.Manager
	started  atomic.Bool
	startedAt time.Time
	resumeMetrics resumeState
}

type resumeState struct {
	skipped int64
	totalBefore int
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
	var limiter engratelimit.RateLimiter
	if cfg.RateLimit.Enabled {
		limiter = engratelimit.NewAdaptiveRateLimiter(cfg.RateLimit)
	}

	// Override checkpoint path if provided directly on facade config
	if cfg.CheckpointPath != "" {
		cfg.Resources.CheckpointPath = cfg.CheckpointPath
	}

	pc := (&cfg).toPipelineConfig(engineOptions{limiter: limiter, resourceManager: rm})
	pl := pipeline.NewPipeline(pc)

	e := &Engine{cfg: cfg, pl: pl, limiter: limiter, rm: rm, startedAt: time.Now()}
	e.started.Store(true)
	return e, nil
}

// Start begins processing of the provided seed URLs. It returns a read-only results channel.
func (e *Engine) Start(ctx context.Context, seeds []string) (<-chan *engmodels.CrawlResult, error) {
	if !e.started.Load() {
		return nil, errors.New("engine not started")
	}
	filtered := seeds
	if e.cfg.Resume && e.cfg.Resources.CheckpointPath != "" {
		filtered = e.filterSeeds(seeds)
	}
	results := e.pl.ProcessURLs(ctx, filtered)
	return results, nil
}

func (e *Engine) filterSeeds(seeds []string) []string {
	e.resumeMetrics.totalBefore = len(seeds)
	path := e.cfg.Resources.CheckpointPath
	file, err := os.Open(path)
	if err != nil {
		return seeds // if missing treat as fresh
	}
	defer file.Close()
	seen := make(map[string]struct{}, len(seeds))
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		seen[strings.TrimSpace(scanner.Text())] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return seeds
	}
	out := make([]string, 0, len(seeds))
	for _, s := range seeds {
		if _, ok := seen[s]; ok {
			e.resumeMetrics.skipped++
			continue
		}
		out = append(out, s)
	}
	return out
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
	if e.cfg.Resume {
		snap.Resume = &ResumeSnapshot{SeedsBefore: e.resumeMetrics.totalBefore, Skipped: e.resumeMetrics.skipped}
	}
	return snap
}

