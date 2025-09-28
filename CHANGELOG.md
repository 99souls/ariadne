# Changelog

All notable changes to this project will be documented in this file. The format loosely follows Keep a Changelog and Semantic Versioning (pre-1.0 semantics: minor version may introduce breaking changes, signaled clearly).

## [Unreleased]

### Added

- engine: Introduced `strategies.go` consolidating `Fetcher`, `Processor`, `OutputSink`, and `AssetStrategy` interfaces with Experimental annotations (Wave 3).
- config: Added comprehensive Experimental annotations across `engine/config` (unified + runtime config, hot reload, versioning, AB testing) plus export allowlist guard test locking curated surface (Wave 3).
- engine: Added `engine_resources_snapshot_test.go` guard test ensuring `ResourceSnapshot` present only when resources subsystem configured (Wave 4 W4-04 follow-up).
- telemetry: Added export allowlist guard test across telemetry subpackages (events, metrics, tracing, policy, health, logging) locking current public surface (Wave 4 W4-07 governance).
- telemetry: Authored `md/telemetry-boundary.md` documenting current public telemetry surface, pruning candidates, and stability annotations (Wave 4 W4-05 partial).
- engine: Added helper `SelectMetricsProvider` centralizing backend selection (Prometheus, OTEL, noop) for potential reuse by adapters / future CLI telemetry wiring (Wave 4 W4-05).
- cli: Implemented provider-aware metrics adapter wiring (Prometheus handler exposure when selected) plus build info gauge registration; health endpoint retained without change (Wave 4 W4-05 adapter finalization).
- cli: Added integration test `TestCLIMetricsAndHealth` asserting startup log lines for metrics & health servers (replaced flaky HTTP polling approach) (Wave 4 W4-06 hardening).
- ci: Added CLI smoke workflow (`.github/workflows/cli-smoke.yml`) performing short crawl and probing metrics & health endpoints (Wave 4 W4-09 runtime validation).
- docs: Enhanced root `README.md` with embedding example and metrics/health quickstart; updated CLI README with metrics adapter notes (Wave 4 W4-08 docs pass).

### Changed

- engine: Marked `OutputSink` and `AssetStrategy` explicitly Experimental in pruning list (consolidated in strategies.go) (Wave 3).
- engine: Internalized former public resource manager implementation under `engine/internal/resources`; introduced public facade `ResourcesConfig` and preserved snapshot-only exposure (`ResourceSnapshot`) (Wave 4 W4-04).
- telemetry/metrics: Annotated all metrics interfaces (`Counter`, `Gauge`, `Histogram`, `Timer`, `Provider`) as Experimental and documented planned consolidation (Wave 4 W4-05).
- engine: `New` now delegates metrics backend initialization to `SelectMetricsProvider` reducing duplication and clarifying intended extension hook (Wave 4 W4-05).
- cli: Metrics endpoint transitioned from placeholder to provider-backed handler leveraging `engine.SelectMetricsProvider`; simplified verification strategy (log assertion) for deterministic tests (Wave 4 W4-05 / W4-06).
- config: Internalized former runtime configuration & A/B testing implementation (moved under `engine/internal/runtime`); left guarded stub `runtime.go` to prevent re-expansion (Wave 4 W4-03 partial completion).
- telemetry: Build info gauge intentionally emitted only via CLI metrics adapter to avoid expanding engine public surface (Wave 4 adapter scope decision).
- engine: Internalized former public `engine/monitoring` package (now `engine/internal/monitoring`); legacy metrics adapter updated to reference internal path (C2 pruning).
  _- engine: Internalized `engine/business/_`packages under`engine/internal/business/\*` (crawler, processor, output, policies) consolidating business rule system behind internal boundary (C2 pruning). No public re-export provided; future facade will expose only minimal policy tuning hooks if justified.
- engine: Snapshot now always includes a non-nil `Limiter` field; when rate limiting is disabled an empty `LimiterSnapshot` is returned (simplifies callers, part of C5 hard cut).
- policy: Adopted hard-cut removal approach pre-1.0 (no deprecation shims); plan & docs updated to reflect immediate removals with CHANGELOG notice only (applies retroactively to C5 and forward).
- engine: Telemetry policy package internalized (`engine/telemetry/policy` -> `engine/internal/telemetry/policy`); public access now via facade methods `Engine.Policy()`, `Engine.UpdateTelemetryPolicy()` and re-exported root types (`TelemetryPolicy`, `HealthPolicy`, `TracingPolicy`, `EventBusPolicy`) plus `DefaultTelemetryPolicy()` helper (C6 step 2b).

### Removed

- engine: Removed test-only method `(*Engine).HealthEvaluatorForTest` (Wave 3 API pruning). Tests should supply a `HealthSource` stub to HTTP health/readiness handlers instead.
- engine: Removed exported functional option type `Option`; constructor now uses only `Config` (Wave 3 API pruning).
- crawler: Removed deprecated alias `FetchedPage` in favor of `FetchResult` (Wave 3 pruning – breaking pre-v1 acceptable).
- engine: Deprecated & stubbed former `engine/resources` package (now empty, enforced by allowlist guard) after internalization (Wave 4 W4-04).
- engine: Removed `engine/adapters/telemetryhttp` (HTTP handlers now owned by CLI; shrink core surface) (C1 pruning).
- engine: Removed previously stubbed `engine/resources` package entirely (snapshot-only exposure via Engine remains) (C1 pruning).
- engine: Removed experimental `engine/strategies/` package (redundant; interfaces consolidated in root `strategies.go`) (C1 pruning).
- config: Removed vestigial `engine/config/runtime.go` stub (runtime system internalized; guard no longer required) (C1 pruning).
- engine: Removed public access to business implementation packages (`engine/business/*`) and monitoring metrics scaffolding (`engine/monitoring`) by internalization (C2 pruning – breaking pre-v1 acceptable).
- config: Removed experimental unified configuration layer (`UnifiedBusinessConfig`, advanced runtime layering & AB testing helpers) and associated tests; public `config` package now intentionally exposes no symbols (C3 pruning – simplifies facade and prevents re-expansion of config surface).
- engine: Internalized public crawler, processor, and output concrete implementation packages (`engine/crawler`, `engine/processor`, `engine/output` including sinks, assembly, enhancement, html, markdown, stdout) under `engine/internal/` (C4 pruning). Removed their public tests; updated imports; regenerated API report. Facade unchanged (interfaces `Fetcher`, `Processor`, `OutputSink` remain). Pre-v1 breaking change acceptable; all tests green.
- engine: Internalized adaptive rate limiter implementation (`engine/ratelimit`) under `engine/internal/ratelimit` (C5 pruning). Removed public `RateLimiter` interface & concrete types; facade now emits reduced diagnostic snapshot (`engine.LimiterSnapshot`) only. Pre-v1 breaking change acceptable.
- telemetry: Removed public `engine/telemetry/policy` package (C6 step 2b); replaced by root re-exports and facade methods. Pre-v1 breaking change acceptable.
  - governance: Dropped automated API report drift enforcement (pre-commit + CI) in favor of export allowlist guard tests; manual `make api-report` remains available for ad-hoc inspection.
- config: Deleted experimental `engine/configx` layered/dynamic configuration subsystem (C7 pruning). Rationale: avoid premature complexity; only static `engine.Config` supported pre-1.0 (see md/configx-internalization-analysis.md). Git history preserves implementation for future reconsideration.
- telemetry: Internalized and removed public `engine/telemetry/events` package (C8/C9). External consumers now observe telemetry via `Engine.RegisterEventObserver` and `TelemetryEvent` facade only. Event bus implementation is fully internal.
- telemetry: Completed removal of public tracing implementation (`engine/telemetry/tracing`) finalizing tracing internalization (C8).

### BREAKING (Telemetry Consolidation)

The public telemetry surface has been hard-cut and replaced by a narrow facade:

Removed:
  - `engine/telemetry/metrics` (all interfaces, constructors, option structs, adapters)
  - `engine/telemetry/events` (event bus types & constants)
  - `engine/telemetry/tracing` (direct tracer construction/export)
  - `engine/telemetry/policy` (replaced by root re-exported types + facade methods)

Replacement Facade & Configuration:
  - Backend selection & enablement now via `engine.Config{ MetricsEnabled, MetricsBackend }` (values: prom|otel|noop; empty defaults to prom when enabled).
  - Metrics exposition via `Engine.MetricsHandler()` (non-nil only when metrics enabled and Prometheus backend selected; OTEL/noop backends return nil handler by design).
  - Event observation via `Engine.RegisterEventObserver(func(TelemetryEvent))` receiving bridged `TelemetryEvent` values (currently health change events; future categories may be added behind the same facade).
  - Runtime policy updates: `Engine.UpdateTelemetryPolicy()` (with `TelemetryPolicy` and nested `HealthPolicy`, `TracingPolicy`, `EventBusPolicy` re-exported for configuration snapshots & mutation).

Migration Guide:
  - Replace direct provider construction (`metrics.NewPrometheusProvider()`, `metrics.NewOTelProvider()`, etc.) with `Config{ MetricsEnabled: true, MetricsBackend: "prom"|"otel" }`.
  - Remove any `EventBus()` / `Tracer()` usage; register observers instead and rely on future span helper APIs (to be added if needed pre v0.2.0).
  - Expose HTTP metrics endpoint only if `Engine.MetricsHandler()` returns non-nil; unchanged health endpoint wiring.

Rationale: Enables internal evolution of metrics/event/tracing implementations and policy logic without further public churn; reduces accidental dependency surface and simplifies operator story.

### Changed (Post Telemetry Consolidation)

- telemetryhttp: Handlers now accept `HealthHandlerOptions{Source: HealthSource}` instead of concrete engine pointer; simplifies tests and decouples facade.
- engine: Stability annotations applied to telemetry-related facade symbols (`TelemetryEvent`, `RegisterEventObserver`, `MetricsHandler`, policy re-exports).
- engine: Export allowlist guard tests updated to enforce removal of public telemetry implementation packages.
- strategies: Annotated entire package as Experimental with allowlist guard.
- crawler: Added Experimental annotations to `FetchResult`, `FetchPolicy`, and `FetcherStats`.

### Added

- Adaptive percentage-based tracer (policy-driven sample percent) replacing always-on tracer.
- Integrated workload benchmark (`BenchmarkIntegratedWorkload`) simulating page + asset telemetry mix.
- CLI module scaffold (`cli/`) with initial crawl command (Phase 5F Wave 2.5 initiation).
- API pruning candidate list (`engine/API_PRUNING_CANDIDATES.md`) drafted (Wave 3 preparation).
- Dedicated API report tooling module (`tools/apireport`) replacing former `cmd/apireport` path.
- `ROOT_LAYOUT.md` documenting the Atomic Root Layout invariant (no root module; curated directory whitelist).
- engine: Export allowlist guard for root facade (`TestEngineExportAllowlist`).
- strategies: Export allowlist guard and comprehensive Experimental doc comments.

### Breaking

- Removed legacy `packages/engine` tree (hard cut). Old import path `ariadne/packages/engine` no longer exists. Use `github.com/99souls/ariadne/engine`.
- Removed public `engine/pipeline` package; orchestration is now internal under `engine/internal/pipeline`.
- Removed root Go module (`go.mod` at repo root) – repository now operates purely as a `go.work` workspace of submodules (`engine`, `cli`, `tools/apireport`). Consumers must update any accidental root-module import assumptions.
- Root executable entrypoint removed; invoke CLI via `go run ./cli/cmd/ariadne` (or build the binary) instead of `go run .`.
- Relocated API report generator from `cmd/apireport` to standalone module `tools/apireport`; any scripts referencing old path must be updated.

### Changed

- Prometheus timer implementation pre-creates histogram (reduces per-timer allocations).
- Makefile & CI workflows now iterate over explicit module list (no implicit root build).
- Enforcement tests relocated to `cli/` to guard against importing `engine/internal/*` from the CLI surface.

### Deferred

- External trace exporter wiring (will introduce build tag & binary size note)
- Error/latency biased sampling boosts (policy fields scaffolded but logic deferred).
- Consolidation / possible internalization of `engine/monitoring` and business metrics constructs (pruning list v2 candidate).
- Potential shrink or removal of large Experimental `UnifiedBusinessConfig` in favor of narrower facade-level config (tracked for pruning list v2).

## [Phase 5E Completion] - 2025-09-27

Comprehensive Monitoring & Observability Expansion (Phase 5E) is complete.

### Added

- Metrics abstraction with selectable backend: Prometheus exporter (stable) & experimental OTEL provider.
- Event bus replacing legacy ring buffer with subscription API, backpressure & drop accounting.
- Tracing span hierarchy (crawl → page → stage → asset) with runtime-adjustable sampling.
- Structured logging enrichment (correlation fields, JSON mode) unified with tracing context.
- Health & readiness HTTP endpoints (`/healthz`, `/readyz`) plus metrics endpoint exposure.
- Runtime TelemetryPolicy enabling hot toggles (metrics, tracing, events, log level/format, sampling rate).
- OTEL cardinality guard rails & internal exceed counter `ariadne_internal_cardinality_exceeded_total` + one-time warning.
- Benchmark suite for metrics provider operations (counter, histogram, timer) across noop/prom/otel backends.
- Overhead report (`telemetry-overhead.md`) and SLO baseline document (`slo-baselines.md`).
- Documentation set: telemetry architecture, metrics reference, operator guide, overhead & SLO baselines.

### Changed

- Unified configuration path for selecting metrics backend via `metricsBackend` value.
- Logging/tracing integration standardized field taxonomy preventing key drift.

### Performance

- Single metric op overhead (counter/histogram) remains within low tens of ns (Prometheus ~57–65ns; OTEL ~4ns) vs noop ~1ns.
- Timer latency: Prometheus ~675ns, OTEL ~315ns, noop ~190ns; optimization opportunity identified (alloc reductions) without blocking release.

### Deferred

- Integrated end-to-end crawl workload overhead percentage validation.
- Prometheus timer allocation reduction (4 allocs/op) optimization.
- External trace exporter & build tag size audit.
- Adaptive/error-biased trace sampling prototype.
- Event schema unification & metrics push gateway exploration.

### Notes

Phase 5E establishes a stable foundation for advanced operator tooling and future distributed scaling phases. Deferred items are intentional and tracked for subsequent iterations.

## [v0.1.0] - 2025-09-26

### Added

- Engine facade (`packages/engine`) providing unified lifecycle: `New`, `Start`, `Stop`, `Snapshot`.
- Adaptive rate limiter with AIMD tuning, circuit breaker states, Retry-After handling, shard eviction.
- Multi-stage pipeline (discovery, extraction, processing, output) with metrics.
- Resource manager (LRU cache + disk spillover + checkpoint journal) and stats snapshot.
- Resume-from-checkpoint (seed filtering) + resume metrics in snapshot.
- Mock HTTP server test utility eliminating external network dependencies (fully offline test suite).
- Top-N domain limiter summaries (`LimiterSnapshot.Domains`).
- CLI migration to engine facade with flags: `-seeds`, `-seed-file`, `-resume`, `-checkpoint`, `-snapshot-interval`, `-version`.
- Enforcement test preventing `main.go` from importing `internal/*` packages.
- API stability guide and migration notes.

### Changed

- Deprecated (soft) direct use of `pipeline.NewPipeline` for production entrypoints (facade preferred).
- Updated README to emphasize engine-driven CLI (pending incremental refinements).

### Removed

- External dependency on httpbin.org in tests.

### Security

- N/A (initial baseline).

---

## Historical (Pre-v0.1.0)

Development phases (Phase 1–3) established core models, asset pipeline, and processing architecture before facade introduction.
