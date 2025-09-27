## Public API Report (Draft)

Generated snapshot of exported identifiers under `engine` (excluding `engine/internal`). Used for pruning & stability annotations.

### engine (facade)
- Types: Snapshot, ResourceSnapshot, ResumeSnapshot, Engine, AssetRef, AssetMode, AssetAction, MaterializedAsset, AssetStrategy (interface), AssetEvent, AssetEventPublisher (interface), AssetMetrics, AssetMetricsSnapshot, AssetPolicy (if defined elsewhere), AssetFilter (if present)
- Functions / Methods (excerpt): New, (*Engine).Policy, (*Engine).MetricsProvider, (*Engine).HealthEvaluatorForTest, (*Engine).UpdateTelemetryPolicy, (*Engine).HealthSnapshot, (*Engine).Start, (*Engine).Stop, (*Engine).Snapshot, (*Engine).AssetMetrics, (*Engine).PublishAssetEvent, (*Engine).SetAssetStrategy, (*Engine).SetAssetPolicy (future), etc.
- Constants: AssetModeDownload, AssetModeSkip, AssetModeInline, AssetModeRewrite

### engine/models
(Enumerate core data structures: Page, CrawlResult, PageMeta, OpenGraphMeta, ScraperConfig, etc.)

### engine/pipeline
Exported: NewPipeline (candidate for deprecation), PipelineMetrics, etc.

### engine/ratelimit
Exported: NewAdaptiveRateLimiter, RateLimiter interface, RateLimitConfig, LimiterSnapshot

### engine/resources
Exported: Manager, NewManager, ResourceConfig, ResourceStats

### engine/adapters/telemetryhttp
Exported: NewHealthHandler, NewReadinessHandler, NewMetricsHandler, HealthHandlerOptions

### Candidates for Internalization / Pruning
1. Direct construction helpers (NewPipeline) – move behind Engine or internal.
2. Asset strategy fine-grained types not needed by most consumers.
3. Test-only methods (HealthEvaluatorForTest) – keep but mark as unstable.

### Proposed Stability Tiers
- Stable: engine.Engine, engine.Config, engine/models types
- Experimental: pipeline construction, resource manager direct usage
- Internal: asset strategy extensibility points (later once plugin story clarified)

---
This file is maintained manually during Phase 5F; later we can automate via `go list -json` tooling.
