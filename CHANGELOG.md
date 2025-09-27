# Changelog

All notable changes to this project will be documented in this file. The format loosely follows Keep a Changelog and Semantic Versioning (pre-1.0 semantics: minor version may introduce breaking changes, signaled clearly).

## [Unreleased]

### Added
- (Planned) Integrated workload benchmark for telemetry overhead (deferred from Phase 5E)
- (Planned) Adaptive/error-biased trace sampling prototype

### Changed
- (Planned) Prometheus timer allocation optimization

### Deferred
- External trace exporter wiring (will introduce build tag & binary size note)

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
