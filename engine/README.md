# Engine Package

The `engine` package is the public, façade-oriented entry point for embedding Ariadne's crawling and processing capabilities. Implementation details (pipeline orchestration, rate limiting primitives, resource coordination, asset rewriting, telemetry internals) now live exclusively under `engine/internal/*` and are not part of the supported API surface.

## Current Architecture (Post Phase 5)

Public (importable) surface:

- `engine` (facade: construction, lifecycle, snapshotting, telemetry policy, health evaluation, asset strategy enablement)
- `engine/config` (static configuration structs & normalization helpers). Dynamic / layered configuration (former experimental `configx`) was removed (see ADR `md/decisions/2025-09-configx-removal.md`). Any future dynamic reload capability will arrive via a new proposal & facade method, not by re‑exposing internal layering primitives.
- `engine/models` (data structures: Page, CrawlResult, errors)
- `engine/resources` (resource manager configuration & high-level stats)

Removed (internalized) since C5:

- `engine/ratelimit` (adaptive limiter implementation + interfaces). A reduced diagnostic view is now exposed via `engine.LimiterSnapshot` fields on the facade `Snapshot()`. Existing consumers should remove imports of `engine/ratelimit`; no direct replacement API is required.

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

### URL Normalization & Variant De‑duplication

The crawler performs a normalization pass over each discovered URL before:

- Deciding de‑duplication / visited set membership
- Emitting a `CrawlResult` to downstream consumers

Current normalization rules (2025‑09‑29):

1. Strip URL fragment (`#...`).
2. Remove cosmetic / tracking query parameters:
   - `theme` (used for dark/light mode variants)
   - Any parameter whose key starts with `utm_` (e.g. `utm_source`, `utm_campaign`)
3. Preserve ordering of the remaining query parameters (standard `url.Values.Encode()` canonicalization).
4. Leave all other components (scheme, host, path) untouched.

Rationale:

- Prevents multiplication of logical pages due to presentational or marketing parameters.
- Ensures deterministic snapshot & metric cardinality (content de‑dup).
- Keeps genuinely content‑affecting parameters (e.g. `?page=2`, `?q=term`) intact until a future, explicit rule set expands or contracts the list.

Dark Mode Policy:

- The live test site exposes a dark mode variant via `?theme=dark`.
- After normalization only the canonical form (without the `theme` parameter) should surface in results; the variant must not appear as a separate page.
- Integration test `TestLiveSiteDarkModeDeDup` validates this behavior end‑to‑end.

Unit Test Coverage:

- `internal/crawler/normalize_test.go` (`TestNormalizeURLCosmeticParams`) locks the exact transformation table for representative inputs (theme param only, theme + utm combo, preserved functional params, fragment removal).

Extension Guidelines (Future):

- Additional cosmetic keys (e.g. `ref`, `fbclid`) should be added only after: (a) verifying they are non‑semantic for target domains, (b) updating the normalization unit test, and (c) documenting the change here.
- Consider promoting a configurable allow/deny list if use cases diverge (e.g., some users need to retain `utm_*`). That would live behind configuration while keeping the default stable.
- Any change that widens removal scope should include a quick scan for accidental collisions with known application routing patterns.

Stability:

- The current normalization rule set is Experimental; the facade does not yet surface customization. Once stabilized it will obtain a minor version guarantee and a public configuration hook.

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
