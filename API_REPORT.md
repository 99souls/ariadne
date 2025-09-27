# API Report

Generated: 2025-09-27T22:37:56+01:00

## Package `models`

Name | Kind | Stability | Summary
-----|------|-----------|--------
CrawlError | type |  | 
CrawlError.Error | method |  | 
CrawlError.Unwrap | method |  | 
CrawlResult | type |  | CrawlResult represents the result of processing a single URL through the pipeline
CrawlStats | type |  | 
ErrAssetDownloadFailed | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrContentNotFound | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrFileWriteFailed | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrHTMLParsingFailed | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrHTTPError | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrInvalidMaxDepth | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrMarkdownConversion | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrMaxDepthExceeded | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrMaxPagesExceeded | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrMissingAllowedDomains | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrMissingStartURL | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrOutputDirCreation | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrTemplateExecution | var |  | Domain-specific errors (copied for locality; keep values identical)
ErrURLNotAllowed | var |  | Domain-specific errors (copied for locality; keep values identical)
OpenGraphMeta | type |  | 
Page | type |  | Page represents a single scraped web page with its content and metadata
PageMeta | type |  | 
RateLimitConfig | type |  | RateLimitConfig defines adaptive per-domain rate limiting behavior
ScraperConfig | type |  | ScraperConfig holds crawler configuration formerly defined in legacy pkg/models.
ScraperConfig.Validate | method |  | Validate performs basic sanity checks on the configuration.

## Package `ratelimit`

Name | Kind | Stability | Summary
-----|------|-----------|--------
AdaptiveRateLimiter | type |  | 
AdaptiveRateLimiter.Acquire | method |  | 
AdaptiveRateLimiter.Close | method |  | 
AdaptiveRateLimiter.Feedback | method |  | 
AdaptiveRateLimiter.Snapshot | method |  | 
AdaptiveRateLimiter.WithClock | method |  | 
Clock | type |  | Clock abstracts time operations for deterministic testing.
DomainSummary | type |  | 
ErrCircuitOpen | var |  | 
Feedback | type |  | 
LimiterSnapshot | type |  | 
Permit | type |  | 
RateLimiter | type |  | 

## Package `resources`

Name | Kind | Stability | Summary
-----|------|-----------|--------
Config | type |  | Config controls resource management features such as caching, spillover, and checkpoints.
Manager | type |  | Manager coordinates resource usage across the pipeline.
Manager.Acquire | method |  | Acquire reserves an in-flight slot; blocks when capacity reached.
Manager.Checkpoint | method |  | Checkpoint records completion.
Manager.Close | method |  | Close flushes and stops background goroutines.
Manager.GetPage | method |  | GetPage retrieves from cache or spill.
Manager.Release | method |  | Release frees an in-flight slot.
Manager.Stats | method |  | 
Manager.StorePage | method |  | StorePage caches a page, evicting oldest to spill if needed.
Stats | type |  | Stats provides lightweight insight into current resource manager state.

## Package `config`

Name | Kind | Stability | Summary
-----|------|-----------|--------
ABTest | type |  | ABTest represents an A/B test configuration
ABTestResult | type |  | ABTestResult represents results from an A/B test
ABTestingFramework | type |  | ABTestingFramework manages A/B testing for configuration changes
ABTestingFramework.AnalyzeTestResults | method |  | AnalyzeTestResults analyzes A/B test results
ABTestingFramework.CreateABTest | method |  | CreateABTest creates a new A/B test
ABTestingFramework.GetConfigForUser | method |  | GetConfigForUser returns the appropriate configuration for a user based on A/B test
ABTestingFramework.RecordTestResult | method |  | RecordTestResult records a result from an A/B test
ConfigChange | type |  | ConfigChange represents a detected configuration change
ConfigValidator | type |  | ConfigValidator validates configuration before applying updates
ConfigVersion | type |  | ConfigVersion represents a stored configuration version
ConfigVersionManager | type |  | ConfigVersionManager manages configuration version history and rollbacks
ConfigVersionManager.GetVersionHistory | method |  | GetVersionHistory returns the version history
ConfigVersionManager.RollbackToVersion | method |  | RollbackToVersion rolls back to a specific version
ConfigVersionManager.SaveVersion | method |  | SaveVersion saves a configuration version with description
GlobalSettings | type |  | GlobalSettings contains cross-cutting configuration
HotReloadSystem | type |  | HotReloadSystem manages file system watching and configuration hot-reloading
HotReloadSystem.DetectChanges | method |  | DetectChanges compares two configurations and returns true if they differ
HotReloadSystem.StopWatching | method |  | StopWatching stops the file system watcher
HotReloadSystem.WatchConfigChanges | method |  | WatchConfigChanges starts watching for configuration file changes
IntegratedRuntimeSystem | type |  | IntegratedRuntimeSystem combines all runtime configuration management components
IntegratedRuntimeSystem.DeployConfiguration | method |  | DeployConfiguration deploys a new configuration with versioning
IntegratedRuntimeSystem.GetCurrentConfiguration | method |  | GetCurrentConfiguration returns the current configuration
IntegratedRuntimeSystem.RollbackToVersion | method |  | RollbackToVersion rolls back to a specific configuration version
RuntimeBusinessConfig | type |  | RuntimeBusinessConfig represents a complete runtime configuration
RuntimeConfigManager | type |  | RuntimeConfigManager manages runtime configuration updates
RuntimeConfigManager.AddValidator | method |  | AddValidator adds a configuration validator
RuntimeConfigManager.GetCurrentConfig | method |  | GetCurrentConfig returns the current configuration (read-only copy)
RuntimeConfigManager.LoadConfiguration | method |  | LoadConfiguration loads configuration from file
RuntimeConfigManager.UpdateConfiguration | method |  | UpdateConfiguration updates the current configuration
RuntimeConfigManager.ValidateConfiguration | method |  | ValidateConfiguration validates a configuration without applying it
TestResultRecord | type |  | TestResultRecord represents a single test result record
UnifiedBusinessConfig | type |  | UnifiedBusinessConfig provides a unified configuration for all engine components
UnifiedBusinessConfig.ApplyDefaults | method |  | ApplyDefaults applies default values to all components
UnifiedBusinessConfig.ApplyFetchDefaults | method |  | ApplyFetchDefaults applies fetch policy defaults
UnifiedBusinessConfig.ApplyGlobalDefaults | method |  | ApplyGlobalDefaults applies global settings defaults
UnifiedBusinessConfig.ApplyProcessDefaults | method |  | ApplyProcessDefaults applies process policy defaults
UnifiedBusinessConfig.ApplySinkDefaults | method |  | ApplySinkDefaults applies sink policy defaults
UnifiedBusinessConfig.ExtractFetchPolicy | method |  | ExtractFetchPolicy returns a copy of the fetch policy
UnifiedBusinessConfig.ExtractProcessPolicy | method |  | ExtractProcessPolicy returns a copy of the process policy
UnifiedBusinessConfig.ExtractSinkPolicy | method |  | ExtractSinkPolicy returns a copy of the sink policy
UnifiedBusinessConfig.Validate | method |  | Validate performs comprehensive validation of the unified configuration
VariantResult | type |  | VariantResult represents results for a specific variant

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
Config | type |  | Config is the public configuration surface for the Engine facade. It intentionally
DefaultAssetStrategy | type |  | DefaultAssetStrategy implements AssetStrategy with instrumentation hooks.
DefaultAssetStrategy.Decide | method |  | 
DefaultAssetStrategy.Discover | method |  | Discover parses the HTML and extracts candidate asset references.
DefaultAssetStrategy.Execute | method |  | 
DefaultAssetStrategy.Name | method |  | 
DefaultAssetStrategy.Rewrite | method |  | 
Engine | type | Stable | Engine composes the pipeline, limiter, and resource manager under a single facade.
Engine.AssetEvents | method |  | AssetEvents returns a snapshot copy of collected events.
Engine.AssetMetricsSnapshot | method |  | AssetMetricsSnapshot returns current aggregated counters (nil if strategy disabled)
Engine.EventBus | method |  | EventBus exposes the telemetry event bus (non-nil).
Engine.HealthEvaluatorForTest | method |  | HealthEvaluatorForTest allows tests to replace the evaluator (not concurrency-safe for production use).
Engine.HealthSnapshot | method |  | HealthSnapshot evaluates (or returns cached) subsystem health. Zero-value if disabled.
Engine.MetricsProvider | method |  | MetricsProvider returns the active metrics provider (may be nil if disabled).
Engine.Policy | method |  | Policy returns the current telemetry policy snapshot (never nil; returns default if unset)
Engine.Snapshot | method |  | Snapshot returns a unified state view.
Engine.Start | method |  | Start begins processing of the provided seed URLs. It returns a read-only results channel.
Engine.Stop | method |  | Stop gracefully stops the engine and underlying components.
Engine.Tracer | method |  | Tracer returns the engine's tracer implementation.
Engine.UpdateTelemetryPolicy | method |  | UpdateTelemetryPolicy atomically swaps the active policy. Nil input resets to defaults.
EngineStrategies | type |  | EngineStrategies defines business logic components for dependency injection
MaterializedAsset | type |  | MaterializedAsset represents an asset after execution (download / inline / optimization).
Option | type |  | Option functional option for customization.
ResourceSnapshot | type | Experimental | ResourceSnapshot surfaces basic cache / spill / checkpoint telemetry.
ResumeSnapshot | type |  | ResumeSnapshot exposes resume filtering counters.
Snapshot | type | Stable | Snapshot is a unified view of engine state (initial minimal subset).

