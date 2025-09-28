# API Report

Signature: 3d4adcfd79c93f4dd324e6ba2f73161abe2888f46e284e539e5e802bcd618247

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
Engine.HealthSnapshot | method | Experimental | HealthSnapshot evaluates (or returns cached) subsystem health. Zero-value if disabled.
Engine.MetricsHandler | method |  | MetricsHandler returns the HTTP handler for metrics exposition (Prometheus backend only).
Engine.Policy | method |  | 
Engine.RegisterEventObserver | method | Experimental | RegisterEventObserver adds an observer invoked synchronously for each internal telemetry
Engine.Snapshot | method | Stable | Snapshot returns a unified state view.
Engine.Start | method | Stable | Start begins processing of the provided seed URLs and returns a read-only results channel.
Engine.Stop | method | Stable | Stop gracefully stops the engine and underlying components.
Engine.UpdateTelemetryPolicy | method | Experimental | UpdateTelemetryPolicy atomically swaps the active policy. Nil input resets to defaults.
EngineStrategies | type | Experimental | EngineStrategies defines business logic components for dependency injection.
EventBusPolicy | type |  | 
EventObserver | type | Experimental | EventObserver receives TelemetryEvent notifications.
Fetcher | type | Experimental | Fetcher defines how pages are fetched.
HealthPolicy | type |  | 
LimiterDomainState | type | Experimental | LimiterDomainState summarizes recent domain-level adaptive state.
LimiterSnapshot | type | Experimental | LimiterSnapshot is a public, reduced view of the internal adaptive rate limiter state.
MaterializedAsset | type |  | MaterializedAsset represents an asset after execution (download / inline / optimization).
OutputSink | type | Experimental | OutputSink consumes processed pages.
Processor | type | Experimental | Processor transforms a fetched page into enriched content.
ResourceSnapshot | type | Experimental | ResourceSnapshot summarizes resource manager internal counters.
ResourcesConfig | type | Experimental | ResourcesConfig is the public facade configuration for resource management.
ResumeSnapshot | type | Experimental | ResumeSnapshot contains resume filter statistics.
Snapshot | type | Stable | Snapshot is a unified view of engine state.
TelemetryEvent | type | Experimental | TelemetryEvent is a reduced, stable event representation for external observers.
TelemetryOptions | type | Experimental | TelemetryOptions describes which telemetry subsystems are enabled plus tuning knobs.
TelemetryPolicy | type | Experimental | Policy returns the current telemetry policy snapshot.
TracingPolicy | type |  | 

## Package `config`

Package config provides higher-level composition helpers for engine component
policies plus runtime configuration facilities.

Experimental: This package's exported surface is still being refined prior to
v1.0. Types and functions here may be renamed, relocated (some content may
move under internal/), or significantly reduced in scope. Consumers should
treat all identifiers as Experimental unless/until explicitly promoted to
Stable in documentation.

(no exported symbols)

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

