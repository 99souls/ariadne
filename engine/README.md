# Engine Package

The `engine` package is the public, façade-oriented entry point for embedding Ariadne's crawling and processing capabilities. Implementation details (pipeline orchestration, rate limiting primitives, resource coordination, asset rewriting, telemetry internals) now live exclusively under `engine/internal/*` and are not part of the supported API surface.

## Current Architecture (Post Phase 5)

Public (importable) surface:

- `engine` (facade: construction, lifecycle, snapshotting, telemetry policy, health evaluation, asset strategy enablement)
- `engine/config` (configuration structs & normalization helpers)
- `engine/models` (data structures: Page, CrawlResult, errors)
- `engine/ratelimit` (adaptive limiter interfaces & snapshots)
- `engine/resources` (resource manager configuration & high-level stats)

Internal-only (subject to change without notice):

- `engine/internal/pipeline` (multi-stage orchestration, retries, backpressure)
- `engine/internal/*` (crawler, processor, downloader/assets, telemetry subsystem wiring, test utilities)

The former public `engine/pipeline` package has been fully removed. All orchestration now occurs behind the facade; direct pipeline construction and tests were migrated internally to preserve behavior and coverage.

## Stability Policy

See `API_STABILITY.md` for detailed stability tiers. In summary:

- Facade lifecycle (`New`, `Start`, `Stop`, `Snapshot`) is Stable.
- Core worker sizing & rate/resource toggle fields in `Config` are Stable.
- Resume, asset policy, metrics backend knobs are Experimental (shape may evolve).
- Internal packages provide no compatibility guarantees.

## Testing Strategy

Behavioral and stress tests for backpressure, graceful shutdown, metrics aggregation, rate limiting feedback, and asset strategy integration reside under `engine/internal/pipeline/*_test.go` to validate invariants while keeping implementation private. Facade integration tests (e.g. `engine_integration_test.go`, `resume_integration_test.go`) ensure public contract correctness.

## Telemetry & Observability

The engine wires an adaptive tracer, metrics provider (Prometheus or OpenTelemetry), event bus, and health evaluator. Policy-driven thresholds (failure ratios, probe TTLs, resource backlog) are configurable via `UpdateTelemetryPolicy` and reflected in `HealthSnapshot` plus metrics gauges.

## Rationale for Internalization

Eliminating the public pipeline entry:

1. Prevents accidental tight coupling to orchestration internals.
2. Enables iterative evolution (stage composition, concurrency control, retry semantics) without breaking downstream users.
3. Simplifies API surface and documentation for the first tagged release (`v0.1.0`).

## Regenerating API Report

Run `make api-report` to rebuild `API_REPORT.md` (uses the `tools/apireport` module) enumerating exported symbols by stability tier. Internal packages (`engine/internal/*`) are excluded.

---

This README reflects the post-internalization architecture and will evolve ahead of the `v0.1.0` tag.
