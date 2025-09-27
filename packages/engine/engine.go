package engine

import (
	"bufio"
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"time"

	engmodels "ariadne/packages/engine/models"
	engpipeline "ariadne/packages/engine/pipeline"
	engratelimit "ariadne/packages/engine/ratelimit"
	engresources "ariadne/packages/engine/resources"
)

// Snapshot is a unified view of engine state (initial minimal subset).
type Snapshot struct {
	StartedAt time.Time                     `json:"started_at"`
	Uptime    time.Duration                 `json:"uptime"`
	Pipeline  *engpipeline.PipelineMetrics  `json:"pipeline,omitempty"`
	Limiter   *engratelimit.LimiterSnapshot `json:"limiter,omitempty"`
	Resources *ResourceSnapshot             `json:"resources,omitempty"`
	Resume    *ResumeSnapshot               `json:"resume,omitempty"`
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
	cfg           Config
	pl            *engpipeline.Pipeline
	limiter       engratelimit.RateLimiter
	rm            *engresources.Manager
	started       atomic.Bool
	startedAt     time.Time
	resumeMetrics resumeState
	strategies    interface{} // Placeholder for broader strategy sets (future)
	assetStrategy AssetStrategy
}

type resumeState struct {
	skipped     int64
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
	var rm *engresources.Manager
	if cfg.Resources.CacheCapacity > 0 || cfg.Resources.MaxInFlight > 0 || cfg.Resources.CheckpointPath != "" {
		manager, err := engresources.NewManager(cfg.Resources)
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
	pl := engpipeline.NewPipeline(pc)

	e := &Engine{cfg: cfg, pl: pl, limiter: limiter, rm: rm, startedAt: time.Now()}

	// Phase 5D Iteration 5: initialize asset strategy if enabled
	if cfg.AssetPolicy.Enabled {
		// For now always use DefaultAssetStrategy; future customization could be injected
		// via options or EngineStrategies.
		as := &DefaultAssetStrategy{}
		if err := cfg.AssetPolicy.Validate(); err != nil {
			return nil, err
		}
		e.assetStrategy = as
		// Inject hook into pipeline for per-page processing.
		if e.pl != nil {
			policy := cfg.AssetPolicy
			e.pl.Config().AssetProcessingHook = func(ctx context.Context, page *engmodels.Page) (*engmodels.Page, error) {
				if page == nil || page.Content == "" {
					return page, nil
				}
				refs, err := as.Discover(ctx, page)
				if err != nil || len(refs) == 0 {
					return page, err
				}
				actions, err := as.Decide(ctx, refs, policy)
				if err != nil || len(actions) == 0 {
					return page, err
				}
				mats, err := as.Execute(ctx, actions, policy)
				if err != nil || len(mats) == 0 {
					return page, err
				}
				return as.Rewrite(ctx, page, mats, policy)
			}
		}
	}
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
	defer func() { _ = file.Close() }()
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

// EngineStrategies defines business logic components for dependency injection
// This is the foundation for Phase 5A Step 4: Strategy-Aware Engine Constructor
type EngineStrategies struct {
	Fetcher     interface{} // Placeholder for crawler.Fetcher interface
	Processors  interface{} // Placeholder for []processor.Processor slice
	OutputSinks interface{} // Placeholder for []output.OutputSink slice
}

// NewWithStrategies creates an engine with custom business logic strategies
// This is a foundational implementation for Phase 5A Step 4
func NewWithStrategies(cfg Config, strategies EngineStrategies, opts ...Option) (*Engine, error) {
	// Build engine using existing constructor
	engine, err := New(cfg, opts...)
	if err != nil {
		return nil, err
	}

	// Store strategies for future use in pipeline integration
	engine.strategies = strategies
	return engine, nil
}
