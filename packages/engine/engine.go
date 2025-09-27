package engine

import (
	"bufio"
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	engmodels "ariadne/packages/engine/models"
	engpipeline "ariadne/packages/engine/pipeline"
	engratelimit "ariadne/packages/engine/ratelimit"
	engresources "ariadne/packages/engine/resources"
	telemEvents "ariadne/packages/engine/telemetry/events"
	telemetryhealth "ariadne/packages/engine/telemetry/health"
	telemetrymetrics "ariadne/packages/engine/telemetry/metrics"
	telempolicy "ariadne/packages/engine/telemetry/policy"
	telemetrytracing "ariadne/packages/engine/telemetry/tracing"
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
	assetMetrics  *AssetMetrics
	assetEvents   []AssetEvent // simple in-memory buffer for now (Iteration 6 minimal impl)
	assetEventsMu sync.Mutex   // Iteration 7 part 2: protect slice under concurrency

	// Phase 5E: metrics provider (initially optional; nil if disabled)
	metricsProvider telemetrymetrics.Provider
	// Phase 5E Iteration 2: event bus (always initialized; metrics provider may be noop)
	eventBus telemEvents.Bus
	// Phase 5E Iteration 3: tracer (simple in-process, may be noop based on future config)
	tracer telemetrytracing.Tracer
	// Phase 5E Iteration 4: health evaluator
	healthEval *telemetryhealth.Evaluator
	// health status instrumentation
	healthStatusGauge telemetrymetrics.Gauge
	lastHealth        atomic.Value // stores telemetryhealth.Status as string

	// Telemetry policy (atomic snapshot). Nil => use internal defaults from policy.Default().
	telemetryPolicy atomic.Pointer[telempolicy.TelemetryPolicy]
}

// Policy returns the current telemetry policy snapshot (never nil; returns default if unset)
func (e *Engine) Policy() telempolicy.TelemetryPolicy {
	if p := e.telemetryPolicy.Load(); p != nil {
		return *p
	}
	def := telempolicy.Default()
	return def
}

// MetricsProvider returns the active metrics provider (may be nil if disabled).
func (e *Engine) MetricsProvider() telemetrymetrics.Provider { return e.metricsProvider }

// HealthEvaluatorForTest allows tests to replace the evaluator (not concurrency-safe for production use).
func (e *Engine) HealthEvaluatorForTest(ev *telemetryhealth.Evaluator) {
	if e == nil || ev == nil {
		return
	}
	e.healthEval = ev
}

// UpdateTelemetryPolicy atomically swaps the active policy. Nil input resets to defaults.
// Safe for concurrent use. Probes pick up new thresholds on next evaluation cycle.
func (e *Engine) UpdateTelemetryPolicy(p *telempolicy.TelemetryPolicy) {
	if e == nil {
		return
	}
	var snap telempolicy.TelemetryPolicy
	if p == nil {
		snap = telempolicy.Default()
	} else {
		snap = p.Normalize()
	}
	old := e.Policy()
	e.telemetryPolicy.Store(&snap)
	// If probe TTL changed, rebuild evaluator so cache semantics reflect new policy.
	if old.Health.ProbeTTL != snap.Health.ProbeTTL {
		// Rebuild health evaluator with existing probes referencing dynamic policy via e.Policy()
		if e.healthEval != nil {
			limiterProbe, resourceProbe, pipelineProbe := e.healthProbes()
			e.healthEval = telemetryhealth.NewEvaluator(snap.Health.ProbeTTL, limiterProbe, resourceProbe, pipelineProbe)
		}
	}
}

// healthProbes returns fresh probe funcs referencing current engine state & dynamic policy.
func (e *Engine) healthProbes() (telemetryhealth.Probe, telemetryhealth.Probe, telemetryhealth.Probe) {
	limiterProbe := telemetryhealth.ProbeFunc(func(ctx context.Context) telemetryhealth.ProbeResult {
		if e.limiter == nil {
			return telemetryhealth.Healthy("rate_limiter")
		}
		s := e.limiter.Snapshot()
		if s.OpenCircuits == 0 {
			return telemetryhealth.Healthy("rate_limiter")
		}
		if s.OpenCircuits > 0 && s.OpenCircuits < int64(len(s.Domains))/2+1 {
			return telemetryhealth.Degraded("rate_limiter", "some open circuits")
		}
		return telemetryhealth.Unhealthy("rate_limiter", "many open circuits")
	})
	resourceProbe := telemetryhealth.ProbeFunc(func(ctx context.Context) telemetryhealth.ProbeResult {
		if e.rm == nil {
			return telemetryhealth.Healthy("resources")
		}
		st := e.rm.Stats()
		pol := e.Policy()
		if st.CheckpointQueued >= pol.Health.ResourceUnhealthyCheckpoint {
			return telemetryhealth.Unhealthy("resources", "checkpoint backlog severe")
		}
		if st.CheckpointQueued >= pol.Health.ResourceDegradedCheckpoint {
			return telemetryhealth.Degraded("resources", "checkpoint backlog")
		}
		return telemetryhealth.Healthy("resources")
	})
	pipelineProbe := telemetryhealth.ProbeFunc(func(ctx context.Context) telemetryhealth.ProbeResult {
		if e.pl == nil {
			return telemetryhealth.Unknown("pipeline", "not initialized")
		}
		m := e.pl.Metrics()
		if m == nil {
			return telemetryhealth.Unknown("pipeline", "metrics nil")
		}
		processed := m.TotalProcessed
		failed := m.TotalFailed
		pol := e.Policy()
		if processed < pol.Health.PipelineMinSamples {
			return telemetryhealth.Healthy("pipeline")
		}
		ratio := float64(failed) / float64(processed)
		if ratio >= pol.Health.PipelineUnhealthyRatio {
			return telemetryhealth.Unhealthy("pipeline", "failure ratio severe")
		}
		if ratio >= pol.Health.PipelineDegradedRatio {
			return telemetryhealth.Degraded("pipeline", "failure ratio elevated")
		}
		return telemetryhealth.Healthy("pipeline")
	})
	return limiterProbe, resourceProbe, pipelineProbe
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

	// Phase 5E Iteration 1: initialize metrics provider if enabled (non-invasive wiring)
	if cfg.MetricsEnabled {
		// For now always use Prometheus provider. Future: allow injection / selection.
		p := telemetrymetrics.NewPrometheusProvider(telemetrymetrics.PrometheusProviderOptions{})
		e.metricsProvider = p
		// NOTE: Exposing handler or starting HTTP server is responsibility of caller to avoid
		// unilaterally opening ports. If PrometheusListenAddr is set future iteration may spawn server.
	}

	// Phase 5E Iteration 2: initialize event bus (metrics provider may be nil; bus tolerates nil -> noop metrics)
	e.eventBus = telemEvents.NewBus(e.metricsProvider)

	// Phase 5E Iteration 3: initialize tracer (enabled by default this iteration; future flag)
	e.tracer = telemetrytracing.NewTracer(true)

	// Phase 5E Iteration 4: initialize health evaluator with basic subsystem probes.
	// TTL seeded from default policy; may be overridden later via UpdateTelemetryPolicy.
	initialPolicy := telempolicy.Default()
	e.telemetryPolicy.Store(&initialPolicy)
	limiterProbe, resourceProbe, pipelineProbe := e.healthProbes()
	e.healthEval = telemetryhealth.NewEvaluator(initialPolicy.Health.ProbeTTL, limiterProbe, resourceProbe, pipelineProbe)
	// Create health status gauge if metrics enabled
	if e.metricsProvider != nil {
		g := e.metricsProvider.NewGauge(telemetrymetrics.GaugeOpts{CommonOpts: telemetrymetrics.CommonOpts{Namespace: "ariadne", Subsystem: "health", Name: "status", Help: "Engine overall health status (1=healthy,0.5=degraded,0=unhealthy,-1=unknown)"}})
		if g != nil {
			e.healthStatusGauge = g
			// initialize value to unknown until first snapshot read
			g.Set(-1)
		}
	}

	// Phase 5D Iteration 5: initialize asset strategy if enabled
	if cfg.AssetPolicy.Enabled {
		// For now always use DefaultAssetStrategy; future customization could be injected
		// via options or EngineStrategies.
		m := &AssetMetrics{}
		publisher := assetEventCollector{engine: e}
		as := NewDefaultAssetStrategy(m, publisher)
		if err := cfg.AssetPolicy.Validate(); err != nil {
			return nil, err
		}
		e.assetStrategy = as
		e.assetMetrics = m
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

// AssetMetricsSnapshot returns current aggregated counters (nil if strategy disabled)
func (e *Engine) AssetMetricsSnapshot() AssetMetricsSnapshot {
	if e.assetMetrics == nil {
		return AssetMetricsSnapshot{}
	}
	return e.assetMetrics.snapshot()
}

// assetEventCollector implements AssetEventPublisher capturing events in memory.
type assetEventCollector struct{ engine *Engine }

func (c assetEventCollector) Publish(ev AssetEvent) {
	if c.engine == nil {
		return
	}
	// Append with simple cap to prevent unbounded growth (keep last 1024)
	c.engine.assetEventsMu.Lock()
	c.engine.assetEvents = append(c.engine.assetEvents, ev)
	if len(c.engine.assetEvents) > 1024 {
		c.engine.assetEvents = c.engine.assetEvents[len(c.engine.assetEvents)-1024:]
	}
	c.engine.assetEventsMu.Unlock()
}

// AssetEvents returns a snapshot copy of collected events.
func (e *Engine) AssetEvents() []AssetEvent {
	e.assetEventsMu.Lock()
	out := make([]AssetEvent, len(e.assetEvents))
	copy(out, e.assetEvents)
	e.assetEventsMu.Unlock()
	return out
}

// HealthSnapshot evaluates (or returns cached) subsystem health. Zero-value if disabled.
func (e *Engine) HealthSnapshot(ctx context.Context) telemetryhealth.Snapshot {
	if e.healthEval == nil {
		return telemetryhealth.Snapshot{}
	}
	snap := e.healthEval.Evaluate(ctx)
	// Map status to numeric value
	var val float64
	switch snap.Overall {
	case telemetryhealth.StatusHealthy:
		val = 1
	case telemetryhealth.StatusDegraded:
		val = 0.5
	case telemetryhealth.StatusUnhealthy:
		val = 0
	default:
		val = -1
	}
	if e.healthStatusGauge != nil {
		e.healthStatusGauge.Set(val)
	}
	prevRaw := e.lastHealth.Load()
	prev := ""
	if prevRaw != nil {
		prev = prevRaw.(string)
	}
	cur := string(snap.Overall)
	if prev != "" && prev != cur && e.eventBus != nil {
		_ = e.eventBus.Publish(telemEvents.Event{Category: telemEvents.CategoryHealth, Type: "health_change", Severity: "info", Fields: map[string]interface{}{"previous": prev, "current": cur}})
	}
	e.lastHealth.Store(cur)
	return snap
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

// EventBus exposes the telemetry event bus (non-nil).
func (e *Engine) EventBus() telemEvents.Bus { return e.eventBus }

// Tracer returns the engine's tracer implementation.
func (e *Engine) Tracer() telemetrytracing.Tracer { return e.tracer }

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
