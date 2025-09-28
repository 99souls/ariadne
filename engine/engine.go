package engine

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	engpipeline "github.com/99souls/ariadne/engine/internal/pipeline"
	intrat "github.com/99souls/ariadne/engine/internal/ratelimit"
	intresources "github.com/99souls/ariadne/engine/internal/resources"
	telemEvents "github.com/99souls/ariadne/engine/internal/telemetry/events"
	intmetrics "github.com/99souls/ariadne/engine/internal/telemetry/metrics"
	inttelempolicy "github.com/99souls/ariadne/engine/internal/telemetry/policy"
	telemetrytracing "github.com/99souls/ariadne/engine/internal/telemetry/tracing"
	engmodels "github.com/99souls/ariadne/engine/models"
	telemetryhealth "github.com/99souls/ariadne/engine/telemetry/health"
)

// Snapshot is a unified view of engine state.
// Stable: Field additions are allowed; existing fields retain semantics.
type Snapshot struct {
	StartedAt time.Time                    `json:"started_at"`
	Uptime    time.Duration                `json:"uptime"`
	Pipeline  *engpipeline.PipelineMetrics `json:"pipeline,omitempty"`
	Limiter   *LimiterSnapshot             `json:"limiter,omitempty"`
	Resources *ResourceSnapshot            `json:"resources,omitempty"`
	Resume    *ResumeSnapshot              `json:"resume,omitempty"`
}

// TelemetryEvent is a reduced, stable event representation for external observers.
// Experimental: Field set may evolve (additive) pre-v1.0. Replaces direct access to
// internal event bus over time (Phase C6).
type TelemetryEvent struct {
	Time     time.Time              `json:"time"`
	Category string                 `json:"category"`
	Type     string                 `json:"type"`
	Severity string                 `json:"severity,omitempty"`
	TraceID  string                 `json:"trace_id,omitempty"`
	SpanID   string                 `json:"span_id,omitempty"`
	Labels   map[string]string      `json:"labels,omitempty"`
	Fields   map[string]interface{} `json:"fields,omitempty"`
}

// TelemetryOptions describes which telemetry subsystems are enabled plus tuning knobs.
// Experimental: Shape may change (e.g., embedded policy structs) before v1.0.
type TelemetryOptions struct {
	EnableMetrics   bool
	EnableTracing   bool
	EnableEvents    bool
	EnableHealth    bool
	MetricsBackend  string
	SamplingPercent float64
}

// EventObserver receives TelemetryEvent notifications.
// Experimental: May gain filtering or asynchronous delivery options.
type EventObserver func(ev TelemetryEvent)

// telemetryConfigFromLegacy maps legacy Config fields (pre-C6) onto TelemetryOptions.
// Temporary helper; will be removed once Config is refactored to embed TelemetryOptions directly.
func telemetryConfigFromLegacy(cfg Config) TelemetryOptions {
	return TelemetryOptions{
		EnableMetrics:   cfg.MetricsEnabled,
		EnableTracing:   true, // legacy behavior always created a tracer (adaptive) – keep until config evolves
		EnableEvents:    true, // legacy behavior always initialized bus
		EnableHealth:    true, // legacy behavior always initialized evaluator
		MetricsBackend:  cfg.MetricsBackend,
		SamplingPercent: 5, // mirrors previous default adaptive tracer percent (placeholder)
	}
}

// LimiterSnapshot is a public, reduced view of the internal adaptive rate limiter state.
// Experimental: Field set may shrink prior to v1.0; external consumers should treat as
// best-effort diagnostics (subject to consolidation under a future telemetry facade).
type LimiterSnapshot struct {
	TotalRequests    int64                `json:"total_requests"`
	Throttled        int64                `json:"throttled"`
	Denied           int64                `json:"denied"`
	OpenCircuits     int64                `json:"open_circuits"`
	HalfOpenCircuits int64                `json:"half_open_circuits"`
	Domains          []LimiterDomainState `json:"domains,omitempty"`
}

// LimiterDomainState summarizes recent domain-level adaptive state.
// Experimental: May be removed or replaced with aggregated counters only.
type LimiterDomainState struct {
	Domain       string    `json:"domain"`
	FillRate     float64   `json:"fill_rate"`
	CircuitState string    `json:"circuit_state"`
	LastActivity time.Time `json:"last_activity"`
}

// ResourceSnapshot summarizes resource manager internal counters.
// Experimental: Field set & naming may change pre-v1.0.
type ResourceSnapshot struct {
	CacheEntries     int `json:"cache_entries"`
	SpillFiles       int `json:"spill_files"`
	InFlight         int `json:"in_flight"`
	CheckpointQueued int `json:"checkpoint_queued"`
}

// ResumeSnapshot contains resume filter statistics.
// Experimental: Mechanism & counters may change; only present when resume enabled.
type ResumeSnapshot struct {
	SeedsBefore int   `json:"seeds_before"`
	Skipped     int64 `json:"skipped"`
}

// Engine composes all subsystems behind a single facade.
// Stable: Core lifecycle methods (Start, Stop, Snapshot, Policy, UpdateTelemetryPolicy) are
// committed to backwards compatible behavior after v1.0; until then only additive changes
// should occur.
type Engine struct {
	cfg           Config
	telemetry     TelemetryOptions
	pl            *engpipeline.Pipeline
	limiter       intrat.RateLimiter
	rm            *intresources.Manager
	started       atomic.Bool
	startedAt     time.Time
	resumeMetrics resumeState
	strategies    interface{} // Placeholder for broader strategy sets (future)
	assetStrategy AssetStrategy
	assetMetrics  *AssetMetrics
	assetEvents   []AssetEvent // simple in-memory buffer for now (Iteration 6 minimal impl)
	assetEventsMu sync.Mutex   // Iteration 7 part 2: protect slice under concurrency

	// Phase 5E: metrics provider (initially optional; nil if disabled)
	metricsProvider intmetrics.Provider
	// Internal telemetry implementations (C6 step2 will remove public accessors)
	eventBus telemEvents.Bus
	tracer   telemetrytracing.Tracer
	// Phase 5E Iteration 4: health evaluator
	healthEval *telemetryhealth.Evaluator
	// health status instrumentation
	healthStatusGauge intmetrics.Gauge
	lastHealth        atomic.Value // stores telemetryhealth.Status as string

	// Telemetry policy (atomic snapshot). Nil => use internal defaults from policy.Default().
	telemetryPolicy atomic.Pointer[inttelempolicy.TelemetryPolicy]

	// Phase C6: externally registered event observers (facade) fed from internal bus.
	eventObserversMu sync.RWMutex
	eventObservers   []EventObserver
}

// Policy returns the current telemetry policy snapshot.
// Experimental: Policy struct shape & semantics may evolve pre-v1.0. Never returns nil.
// Re-export telemetry policy types (C6 step 2b): stable facade surface while implementation internal.
type TelemetryPolicy = inttelempolicy.TelemetryPolicy
type HealthPolicy = inttelempolicy.HealthPolicy
type TracingPolicy = inttelempolicy.TracingPolicy
type EventBusPolicy = inttelempolicy.EventBusPolicy

// DefaultTelemetryPolicy returns the default normalized telemetry policy (wrapper around internal).
func DefaultTelemetryPolicy() TelemetryPolicy { return inttelempolicy.Default() }

func (e *Engine) Policy() TelemetryPolicy {
	if p := e.telemetryPolicy.Load(); p != nil {
		return *p
	}
	def := inttelempolicy.Default()
	return def
}

// MetricsHandler returns the HTTP handler for metrics exposition (Prometheus backend only).
// Returns nil if metrics disabled or backend does not provide an HTTP handler.
func (e *Engine) MetricsHandler() http.Handler {
	if e == nil || e.metricsProvider == nil {
		return nil
	}
	if hp, ok := e.metricsProvider.(interface{ MetricsHandler() http.Handler }); ok {
		return hp.MetricsHandler()
	}
	return nil
}

// UpdateTelemetryPolicy atomically swaps the active policy. Nil input resets to defaults.
// Experimental: May relocate behind a dedicated telemetry subpackage pre-v1.0.
// Safe for concurrent use; probes pick up new thresholds on next evaluation cycle.
func (e *Engine) UpdateTelemetryPolicy(p *TelemetryPolicy) {
	if e == nil {
		return
	}
	var snap inttelempolicy.TelemetryPolicy
	if p == nil {
		snap = inttelempolicy.Default()
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

// optionFn is an internal functional option for future extension.
// (Wave 3) Previously exported as Option; internalized to shrink public surface.
type optionFn func(*Config)

// New constructs an Engine with supplied configuration. Functional options were
// removed (previously ...Option) during Wave 3 pruning; callers now configure
// exclusively via the Config struct.
func New(cfg Config, opts ...optionFn) (*Engine, error) {
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}

	// Build resource manager if configured
	var rm *intresources.Manager
	if cfg.Resources.CacheCapacity > 0 || cfg.Resources.MaxInFlight > 0 || cfg.Resources.CheckpointPath != "" {
		manager, err := intresources.NewManager(cfg.Resources.toInternal())
		if err != nil {
			return nil, err
		}
		rm = manager
	}

	// Build rate limiter
	var limiter intrat.RateLimiter
	if cfg.RateLimit.Enabled {
		limiter = intrat.NewAdaptiveRateLimiter(cfg.RateLimit)
	}

	// Override checkpoint path if provided directly on facade config
	if cfg.CheckpointPath != "" {
		cfg.Resources.CheckpointPath = cfg.CheckpointPath
	}

	pc := (&cfg).toPipelineConfig(engineOptions{limiter: limiter, resourceManager: rm})
	pl := engpipeline.NewPipeline(pc)

	telemOpts := telemetryConfigFromLegacy(cfg)
	e := &Engine{cfg: cfg, telemetry: telemOpts, pl: pl, limiter: limiter, rm: rm, startedAt: time.Now()}

	// Initialize metrics provider (Wave 4 W4-05: delegated to helper for reuse & clarity)
	e.metricsProvider = selectMetricsProvider(cfg)
	// NOTE: Exposing HTTP handler / endpoint binding remains caller responsibility (CLI or embedding app).

	// Phase 5E Iteration 2 / C6 start: initialize event bus only if events enabled
	if telemOpts.EnableEvents {
		e.eventBus = telemEvents.NewBus(e.metricsProvider)
	}

	// Phase 5E Iteration 3 (C6 adaptation): tracer only if enabled
	if telemOpts.EnableTracing {
		e.tracer = telemetrytracing.NewAdaptiveTracer(func() float64 {
			// If policy sampling set use it, else fallback to TelemetryOptions SamplingPercent
			pct := e.Policy().Tracing.SamplePercent
			if pct <= 0 {
				return telemOpts.SamplingPercent
			}
			return pct
		})
	}

	// Phase 5E Iteration 4 (C6 adaptation): health only if enabled
	initialPolicy := inttelempolicy.Default()
	e.telemetryPolicy.Store(&initialPolicy)
	if telemOpts.EnableHealth {
		limiterProbe, resourceProbe, pipelineProbe := e.healthProbes()
		e.healthEval = telemetryhealth.NewEvaluator(initialPolicy.Health.ProbeTTL, limiterProbe, resourceProbe, pipelineProbe)
		// Create health status gauge if metrics enabled
		if e.metricsProvider != nil {
			g := e.metricsProvider.NewGauge(intmetrics.GaugeOpts{CommonOpts: intmetrics.CommonOpts{Namespace: "ariadne", Subsystem: "health", Name: "status", Help: "Engine overall health status (1=healthy,0.5=degraded,0=unhealthy,-1=unknown)"}})
			if g != nil {
				e.healthStatusGauge = g
				g.Set(-1) // initialize unknown
			}
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

// selectMetricsProvider returns a metrics.Provider based on telemetry fields in Config.
// NOTE: This helper was intentionally kept unexported after C9 to avoid
// prematurely codifying an extension point. Embedders configure telemetry
// exclusively via Config{ MetricsEnabled, MetricsBackend }.
// Experimental: Helper may relocate behind a telemetry facade or be internalized if
// embedding approach changes prior to v1.0. Exposed to reduce duplication across
// potential CLI / adapter wiring and to make backend selection auditable in one place.
// (duplicate doc block retained during refactors) Internal metrics provider selection.
// Experimental: Helper may relocate behind a telemetry facade in the future.
func selectMetricsProvider(cfg Config) intmetrics.Provider {
	if !cfg.MetricsEnabled {
		return nil
	}
	switch strings.ToLower(cfg.MetricsBackend) {
	case "", "prom", "prometheus":
		return intmetrics.NewPrometheusProvider(intmetrics.PrometheusProviderOptions{})
	case "otel", "opentelemetry":
		return intmetrics.NewOTelProvider(intmetrics.OTelProviderOptions{})
	case "noop":
		return intmetrics.NewNoopProvider()
	default:
		return intmetrics.NewPrometheusProvider(intmetrics.PrometheusProviderOptions{})
	}
}

// AssetMetricsSnapshot returns current aggregated counters (zero-value if strategy disabled).
// Experimental: Asset subsystem instrumentation may change drastically pre-v1.0.
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

// AssetEvents returns a snapshot copy of collected asset events.
// Experimental: Event model & buffering policy may change or become streaming.
func (e *Engine) AssetEvents() []AssetEvent {
	e.assetEventsMu.Lock()
	out := make([]AssetEvent, len(e.assetEvents))
	copy(out, e.assetEvents)
	e.assetEventsMu.Unlock()
	return out
}

// HealthSnapshot evaluates (or returns cached) subsystem health. Zero-value if disabled.
// Experimental: Health snapshot structure & evaluation cadence may change.
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
		iev := telemEvents.Event{Category: telemEvents.CategoryHealth, Type: "health_change", Severity: "info", Fields: map[string]interface{}{"previous": prev, "current": cur}}
		_ = e.eventBus.Publish(iev)
		// Bridge to facade observers
		e.dispatchEvent(iev)
	}
	e.lastHealth.Store(cur)
	return snap
}

// RegisterEventObserver adds an observer invoked synchronously for each internal telemetry
// event. Safe for concurrent use. No-op if nil provided.
// Experimental: May gain filtering / async delivery options pre-v1.0.
func (e *Engine) RegisterEventObserver(obs EventObserver) {
	if e == nil || obs == nil {
		return
	}
	e.eventObserversMu.Lock()
	e.eventObservers = append(e.eventObservers, obs)
	e.eventObserversMu.Unlock()
}

// dispatchEvent maps an internal bus event to the public TelemetryEvent and notifies observers.
// Called where we publish to the internal bus (limited injection points initially: health changes).
func (e *Engine) dispatchEvent(ev telemEvents.Event) {
	e.eventObserversMu.RLock()
	if len(e.eventObservers) == 0 {
		e.eventObserversMu.RUnlock()
		return
	}
	observers := append([]EventObserver(nil), e.eventObservers...)
	e.eventObserversMu.RUnlock()
	pub := TelemetryEvent{Time: ev.Time, Category: ev.Category, Type: ev.Type, Severity: ev.Severity, TraceID: ev.TraceID, SpanID: ev.SpanID, Labels: ev.Labels, Fields: ev.Fields}
	for _, o := range observers { // synchronous; observers must be fast
		func() { defer func() { _ = recover() }(); o(pub) }()
	}
}

// Start begins processing of the provided seed URLs and returns a read-only results channel.
// Stable: Contract (non-nil channel on success, error on invalid state) will hold after v1.0.
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
// Stable: Idempotent; safe to call multiple times after v1.0.
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
// Stable: See Snapshot field stability guarantees.
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
		is := e.limiter.Snapshot()
		pub := LimiterSnapshot{TotalRequests: is.TotalRequests, Throttled: is.Throttled, Denied: is.Denied, OpenCircuits: is.OpenCircuits, HalfOpenCircuits: is.HalfOpenCircuits}
		if len(is.Domains) > 0 {
			pub.Domains = make([]LimiterDomainState, 0, len(is.Domains))
			for _, d := range is.Domains {
				pub.Domains = append(pub.Domains, LimiterDomainState{Domain: d.Domain, FillRate: d.FillRate, CircuitState: d.CircuitState, LastActivity: d.LastActivity})
			}
		}
		snap.Limiter = &pub
	} else {
		// Provide an empty snapshot rather than nil for simpler external handling.
		empty := LimiterSnapshot{}
		snap.Limiter = &empty
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
// Experimental: Direct bus exposure may be replaced by subscription APIs.
// (C6) Public EventBus() and Tracer() accessors removed; external observers should use
// RegisterEventObserver and future span helper (if introduced). Intentionally no replacement exported now.

// EngineStrategies defines business logic components for dependency injection.
// Experimental: Placeholder for future strategy extension wiring; not yet integrated.
type EngineStrategies struct {
	Fetcher     interface{} // Placeholder for crawler.Fetcher interface
	Processors  interface{} // Placeholder for []processor.Processor slice
	OutputSinks interface{} // Placeholder for []output.OutputSink slice
}

// NewWithStrategies creates an engine with custom business logic strategies.
// Experimental: Construction path likely to change once strategy integration lands.
func NewWithStrategies(cfg Config, strategies EngineStrategies, opts ...optionFn) (*Engine, error) {
	// Build engine using existing constructor
	engine, err := New(cfg, opts...)
	if err != nil {
		return nil, err
	}

	// Store strategies for future use in pipeline integration
	engine.strategies = strategies
	return engine, nil
}
