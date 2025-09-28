# Telemetry Boundary (Wave 4 – Draft)

Status: Draft (Wave 4) – Establishing which telemetry symbols remain intentionally public prior to Wave 5 version baseline.

## Goals

1. Minimize long‑term commitment surface for observability.
2. Allow external adapters (metrics exporter, custom tracing) without exposing internal engine pipeline details.
3. Enable future internalization (or facade relocation) of legacy compatibility shims.

## Current Public Telemetry Packages

| Package | Purpose | Intentional Public Symbols | Notes / Candidate Changes |
|---------|---------|----------------------------|---------------------------|
| `telemetry` (root) | Aggregated governance test location | (No additional exports beyond guards) | Keep empty facade; may add high-level adapter registration later. |
| `telemetry/events` | Lightweight event bus for internal & adapter consumption | `Event`, `Subscription`, `Bus`, `BusStats`, `NewBus`, category constants | Potential future facade to hide `Bus` behind Engine-managed channel; keep for now. |
| `telemetry/metrics` | Abstraction over concrete metrics providers | Interfaces: `Provider`, `Counter`, `Gauge`, `Histogram`, `Timer`; Opt structs; Constructors: `NewPrometheusProvider`, `NewOTelProvider`, `NewNoopProvider`; Provider option types; Legacy adapter: `BusinessCollectorAdapter` | Candidate: internalize `BusinessCollectorAdapter` after replacing legacy `monitoring` dependency or move under `internal/telemetryadapter`. |
| `telemetry/tracing` | Basic span/tracer abstraction + adaptive sampling | `Tracer`, `Span`, `SpanContext`, constructors (`NewTracer`, `NewAdaptiveTracer`), helpers (`SpanFromContext`, `ExtractIDs`) | Keep minimal; evaluate splitting adaptive policy into policy package. |
| `telemetry/policy` | Aggregated runtime policy knobs | `TelemetryPolicy`, `HealthPolicy`, `TracingPolicy`, `EventBusPolicy`, `Default` | Long-term: relocate to `engine/config` or expose snapshot only. |
| `telemetry/health` | Health snapshot + evaluator helpers | `Snapshot`, `ProbeResult`, `Status`, `Probe`, `ProbeFunc`, `Evaluator`, `NewEvaluator`, status helpers (`Healthy`, etc.), status constants | Consider shrinking: export only `Snapshot` + constructor & status constants if evaluators are always engine-owned. |
| `telemetry/logging` | Logging facade thin layer | `Logger`, `NewLogger`, `New` | Potential collapse into single `NewLogger` and deprecate `New`. |

## Allowlist Guard Alignment

`engine/telemetry/telemetry_allowlist_guard_test.go` enumerates exactly the above. Any removal requires updating the allowlist deliberately.

## Proposed Wave 5 Pruning Candidates

1. `metrics.BusinessCollectorAdapter` (remove or move internal; replace with Engine-provided translation path).
2. `telemetry/policy.TelemetryPolicy` sub-structs – replace with high-level config struct consumed by Engine; expose read-only snapshot.
3. `telemetry/health` constructors: collapse helpers into unexported functions; keep only `NewEvaluator`, `Status*` constants, `Snapshot`.
4. `telemetry/logging.New` alias – keep only `NewLogger`.
5. Potential facade method on `Engine` for `SetMetricsProvider(p Provider)` to hide direct use of `metrics` constructors externally.

## Annotations Plan

| Symbol Category | Stability Tag (Target) | Rationale |
|-----------------|------------------------|-----------|
| Core interfaces (`metrics.Provider`, `tracing.Tracer`) | Experimental (promote later) | Need usage validation externally. |
| Legacy adapter (`BusinessCollectorAdapter`) | Deprecated (pre-removal) | Transitional shim; discourage adoption. |
| Policy structs | Experimental (possible redesign) | Likely consolidated into config. |
| Health evaluator pieces | Experimental (shrink surface) | Evaluate simpler API. |
| Logging constructors | Experimental → Stable (once narrowed) | Low risk stable front door. |

## Migration Considerations

- External adopters should prefer engine-level configuration once provided (e.g., `engine.Config.Telemetry.Provider = metrics.NewPrometheusProvider(...)`).
- Direct new provider wiring remains allowed short-term; tagging baseline will document stability tier.

## Next Steps (Actionable)

1. Add Deprecated notice to `BusinessCollectorAdapter` + changelog entry (Wave 4 follow-up).
2. Add Experimental doc comments to core interfaces lacking explicit stability tags.
3. Prepare pruning list v2 referencing this document (W4-10).
4. Evaluate adding `Engine.WithMetricsProvider(p metrics.Provider)` helper hiding constructor details.

## Decision Log

| Date | Decision | Outcome |
|------|----------|---------|
| 2025-09-28 | Lock initial telemetry allowlist (W4-07) | Guard test merged; boundary doc drafted. |

---
Draft; will iterate as pruning v2 list is formalized.
