# Changelog

All notable changes to this project will be documented in this file. The format loosely follows Keep a Changelog and Semantic Versioning (pre-1.0 semantics: minor version may introduce breaking changes, signaled clearly).

## [Unreleased]

### Added

- engine: Introduced `strategies.go` consolidating `Fetcher`, `Processor`, `OutputSink`, and `AssetStrategy` interfaces with Experimental annotations (Wave 3).

### Changed

- engine: Marked `OutputSink` and `AssetStrategy` explicitly Experimental in pruning list (consolidated in strategies.go) (Wave 3).

### Removed

- engine: Removed test-only method `(*Engine).HealthEvaluatorForTest` (Wave 3 API pruning). Tests should supply a `HealthSource` stub to HTTP health/readiness handlers instead.
- engine: Removed exported functional option type `Option`; constructor now uses only `Config` (Wave 3 API pruning).
- crawler: Removed deprecated alias `FetchedPage` in favor of `FetchResult` (Wave 3 pruning – breaking pre-v1 acceptable).

### Changed

- telemetryhttp: Handlers now accept `HealthHandlerOptions{Source: HealthSource}` instead of a concrete `*engine.Engine` field. The `HealthSource` interface requires only `HealthSnapshot(context.Context)`, enabling simpler test doubles and eliminating the need for a mutable public hook on `Engine`.
- engine: Added stability annotations (Experimental / Stable) to `Config` fields, `Engine`, snapshots, telemetry methods, and asset subsystem exports (Wave 3).
- engine: Introduced export allowlist guard test (`engine_allowlist_guard_test.go`) locking current curated root package surface (Wave 3).
- strategies: Annotated entire package as Experimental and added export allowlist guard (`strategies_allowlist_test.go`).
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
