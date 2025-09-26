# Changelog

All notable changes to this project will be documented in this file. The format loosely follows Keep a Changelog and Semantic Versioning (pre-1.0 semantics: minor version may introduce breaking changes, signaled clearly).

## [Unreleased]

- Telemetry export (Prometheus integration) scaffolding

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
