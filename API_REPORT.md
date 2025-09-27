# API Report

Generated: 2025-09-27T23:32:28+01:00

## Package `models`

Name | Kind | Stability | Summary
-----|------|-----------|--------
CrawlError | type | Experimental | CrawlError wraps a stage-specific error with page context.
CrawlError.Error | method |  | 
CrawlError.Unwrap | method |  | 
CrawlResult | type | Experimental | CrawlResult represents the result of processing a single URL through the pipeline.
CrawlStats | type | Experimental | CrawlStats aggregates crawl progress metrics.
ErrAssetDownloadFailed | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrContentNotFound | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrFileWriteFailed | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrHTMLParsingFailed | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrHTTPError | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrInvalidMaxDepth | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrMarkdownConversion | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrMaxDepthExceeded | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrMaxPagesExceeded | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrMissingAllowedDomains | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrMissingStartURL | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrOutputDirCreation | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrTemplateExecution | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
ErrURLNotAllowed | var | Experimental | Domain-specific errors (copied for locality; keep values identical)
OpenGraphMeta | type | Experimental | OpenGraphMeta captures a subset of Open Graph tags.
Page | type | Stable | Page represents a single scraped web page with its content and metadata.
PageMeta | type | Experimental | PageMeta contains structured metadata extracted from the page.
RateLimitConfig | type | Experimental | RateLimitConfig defines adaptive per-domain rate limiting behavior.
ScraperConfig | type | Experimental | ScraperConfig holds crawler configuration formerly defined in legacy pkg/models.
ScraperConfig.Validate | method | Experimental | Validate performs basic sanity checks on the configuration.

## Package `ratelimit`

Name | Kind | Stability | Summary
-----|------|-----------|--------
AdaptiveRateLimiter | type | Experimental | AdaptiveRateLimiter implements RateLimiter using AIMD + circuit breaking.
AdaptiveRateLimiter.Acquire | method |  | 
AdaptiveRateLimiter.Close | method |  | 
AdaptiveRateLimiter.Feedback | method |  | 
AdaptiveRateLimiter.Snapshot | method |  | 
AdaptiveRateLimiter.WithClock | method |  | 
Clock | type | Experimental | Clock abstracts time operations for deterministic testing.
DomainSummary | type | Experimental | DomainSummary reports per-domain adaptive state.
ErrCircuitOpen | var | Experimental | ErrCircuitOpen signals requests are temporarily denied due to breaker state.
Feedback | type | Experimental | Feedback supplies outcome metrics from completed requests.
LimiterSnapshot | type | Experimental | LimiterSnapshot aggregates limiter-level counters.
Permit | type |  | Permit represents an acquired capacity token.
RateLimiter | type | Experimental | RateLimiter is the adaptive per-domain limiter interface.

## Package `resources`

Name | Kind | Stability | Summary
-----|------|-----------|--------
Config | type | Experimental | Config controls resource management features such as caching, spillover, and checkpoints.
Manager | type | Experimental | Manager coordinates resource usage across the pipeline.
Manager.Acquire | method |  | Acquire reserves an in-flight slot; blocks when capacity reached.
Manager.Checkpoint | method |  | Checkpoint records completion.
Manager.Close | method |  | Close flushes and stops background goroutines.
Manager.GetPage | method |  | GetPage retrieves from cache or spill.
Manager.Release | method |  | Release frees an in-flight slot.
Manager.Stats | method |  | 
Manager.StorePage | method |  | StorePage caches a page, evicting oldest to spill if needed.
Stats | type | Experimental | Stats provides lightweight insight into current resource manager state.

## Package `config`

Package config provides higher-level composition helpers for engine component
policies plus runtime configuration facilities.

Experimental: This package's exported surface is still being refined prior to
v1.0. Types and functions here may be renamed, relocated (some content may
move under internal/), or significantly reduced in scope. Consumers should
treat all identifiers as Experimental unless/until explicitly promoted to
Stable in documentation.

Name | Kind | Stability | Summary
-----|------|-----------|--------
ABTest | type | Experimental | Experimental: ABTest definition; fields and naming may change or move.
ABTestResult | type | Experimental | Experimental: ABTestResult aggregates computed statistics; subject to change.
ABTestingFramework | type | Experimental | Experimental: ABTestingFramework provides a simplistic in-process A/B test
ABTestingFramework.AnalyzeTestResults | method | Experimental | Experimental: AnalyzeTestResults computes basic stats; methodology will evolve.
ABTestingFramework.CreateABTest | method | Experimental | Experimental: CreateABTest persists a new test; persistence model may
ABTestingFramework.GetConfigForUser | method | Experimental | Experimental: GetConfigForUser distributes users across variants; hashing
ABTestingFramework.RecordTestResult | method | Experimental | Experimental: RecordTestResult appends result; format & durability may change.
ConfigChange | type | Experimental | Experimental: ConfigChange describes a detected configuration update. Shape
ConfigValidator | type | Experimental | Experimental: ConfigValidator allows custom validation hooks. Interface may
ConfigVersion | type | Experimental | Experimental: ConfigVersion captures stored configuration metadata; shape
ConfigVersionManager | type | Experimental | Experimental: ConfigVersionManager persists configuration versions. May move
ConfigVersionManager.GetVersionHistory | method | Experimental | Experimental: GetVersionHistory enumerates on-disk versions. Ordering &
ConfigVersionManager.RollbackToVersion | method | Experimental | Experimental: RollbackToVersion applies a previous version. Side-effects may
ConfigVersionManager.SaveVersion | method | Experimental | Experimental: SaveVersion persists version metadata. On-disk format not
GlobalSettings | type | Experimental | Experimental: GlobalSettings houses cross-cutting knobs. Field names and
HotReloadSystem | type | Experimental | Experimental: HotReloadSystem watches a config file and produces change
HotReloadSystem.DetectChanges | method | Experimental | Experimental: DetectChanges performs a naive comparison; strategy may change
HotReloadSystem.StopWatching | method | Experimental | Experimental: StopWatching halts watching. May become idempotent error.
HotReloadSystem.WatchConfigChanges | method | Experimental | Experimental: WatchConfigChanges emits change events. Channel protocol and
IntegratedRuntimeSystem | type | Experimental | Experimental: IntegratedRuntimeSystem is a convenience aggregate; may be
IntegratedRuntimeSystem.DeployConfiguration | method | Experimental | Experimental: DeployConfiguration saves & applies config; transactional
IntegratedRuntimeSystem.GetCurrentConfiguration | method | Experimental | Experimental: GetCurrentConfiguration returns current config; may become a
IntegratedRuntimeSystem.RollbackToVersion | method | Experimental | Experimental: RollbackToVersion performs rollback; side-effects may change.
RuntimeBusinessConfig | type | Experimental | Experimental: RuntimeBusinessConfig is a higher-level runtime representation.
RuntimeConfigManager | type | Experimental | Experimental: RuntimeConfigManager orchestrates loading and applying runtime
RuntimeConfigManager.AddValidator | method | Experimental | Experimental: AddValidator registers a validation hook. May shift to
RuntimeConfigManager.GetCurrentConfig | method | Experimental | Experimental: GetCurrentConfig returns a shallow copy; deeper copies may be
RuntimeConfigManager.LoadConfiguration | method | Experimental | Experimental: LoadConfiguration loads config from disk. Error handling and
RuntimeConfigManager.UpdateConfiguration | method | Experimental | Experimental: UpdateConfiguration applies a new configuration. Concurrency
RuntimeConfigManager.ValidateConfiguration | method | Experimental | Experimental: ValidateConfiguration may merge with UpdateConfiguration or be
TestResultRecord | type | Experimental | Experimental: TestResultRecord storage model may change; persistence
UnifiedBusinessConfig | type | Experimental | Experimental: UnifiedBusinessConfig aggregates per-component policies and
UnifiedBusinessConfig.ApplyDefaults | method | Experimental | Experimental: ApplyDefaults sets unset fields to opinionated defaults. The
UnifiedBusinessConfig.ApplyFetchDefaults | method | Experimental | Experimental: ApplyFetchDefaults sets defaults for FetchPolicy. May become
UnifiedBusinessConfig.ApplyGlobalDefaults | method | Experimental | Experimental: ApplyGlobalDefaults sets defaults for GlobalSettings. May
UnifiedBusinessConfig.ApplyProcessDefaults | method | Experimental | Experimental: ApplyProcessDefaults sets defaults for ProcessPolicy. May
UnifiedBusinessConfig.ApplySinkDefaults | method | Experimental | Experimental: ApplySinkDefaults sets defaults for SinkPolicy. May become
UnifiedBusinessConfig.ExtractFetchPolicy | method | Experimental | Experimental: ExtractFetchPolicy returns a defensive copy of FetchPolicy.
UnifiedBusinessConfig.ExtractProcessPolicy | method | Experimental | Experimental: ExtractProcessPolicy returns a defensive copy.
UnifiedBusinessConfig.ExtractSinkPolicy | method | Experimental | Experimental: ExtractSinkPolicy returns a defensive copy.
UnifiedBusinessConfig.Validate | method | Experimental | Experimental: Validate checks internal consistency. Error messages and
VariantResult | type | Experimental | Experimental: VariantResult metrics may change; do not rely on ordering.

## Package `engine`

Name | Kind | Stability | Summary
-----|------|-----------|--------
AssetAction | type |  | AssetAction couples a reference with a decided handling mode.
AssetEvent | type |  | AssetEvent represents a lifecycle occurrence for observability.
AssetEventPublisher | type |  | AssetEventPublisher publishes events (non-blocking behavior recommended).
AssetMetrics | type |  | AssetMetrics holds counters for asset processing lifecycle.
AssetMetricsSnapshot | type |  | Snapshot returns immutable view for assertions / reporting.
AssetMode | type |  | AssetMode describes the handling decision for an asset.
AssetPolicy | type |  | AssetPolicy configures the asset subsystem when enabled. Iteration 1 surface; enforcement &
AssetPolicy.Validate | method |  | Validation placeholder: ensure rewrite prefix has leading & trailing slash semantics.
AssetRef | type |  | AssetRef represents a discovered asset reference inside a page.
AssetStrategy | type |  | AssetStrategy defines the pluggable asset handling pipeline lifecycle.
Config | type | Experimental | Config is the public configuration surface for the Engine facade.
DefaultAssetStrategy | type |  | DefaultAssetStrategy implements AssetStrategy with instrumentation hooks.
DefaultAssetStrategy.Decide | method |  | 
DefaultAssetStrategy.Discover | method |  | Discover parses the HTML and extracts candidate asset references.
DefaultAssetStrategy.Execute | method |  | 
DefaultAssetStrategy.Name | method |  | 
DefaultAssetStrategy.Rewrite | method |  | 
Engine | type | Stable | Engine composes all subsystems behind a single facade.
Engine.AssetEvents | method | Experimental | AssetEvents returns a snapshot copy of collected asset events.
Engine.AssetMetricsSnapshot | method | Experimental | AssetMetricsSnapshot returns current aggregated counters (zero-value if strategy disabled).
Engine.EventBus | method | Experimental | EventBus exposes the telemetry event bus (non-nil).
Engine.HealthSnapshot | method | Experimental | HealthSnapshot evaluates (or returns cached) subsystem health. Zero-value if disabled.
Engine.MetricsProvider | method | Experimental | MetricsProvider returns the active metrics provider (may be nil if disabled).
Engine.Policy | method | Experimental | Policy returns the current telemetry policy snapshot.
Engine.Snapshot | method | Stable | Snapshot returns a unified state view.
Engine.Start | method | Stable | Start begins processing of the provided seed URLs and returns a read-only results channel.
Engine.Stop | method | Stable | Stop gracefully stops the engine and underlying components.
Engine.Tracer | method | Experimental | Tracer returns the engine's tracer implementation.
Engine.UpdateTelemetryPolicy | method | Experimental | UpdateTelemetryPolicy atomically swaps the active policy. Nil input resets to defaults.
EngineStrategies | type | Experimental | EngineStrategies defines business logic components for dependency injection.
Fetcher | type | Experimental | Fetcher defines how pages are fetched.
MaterializedAsset | type |  | MaterializedAsset represents an asset after execution (download / inline / optimization).
OutputSink | type | Experimental | OutputSink consumes processed pages.
Processor | type | Experimental | Processor transforms a fetched page into enriched content.
ResourceSnapshot | type | Experimental | ResourceSnapshot summarizes resource manager internal counters.
ResumeSnapshot | type | Experimental | ResumeSnapshot contains resume filter statistics.
Snapshot | type | Stable | Snapshot is a unified view of engine state.

