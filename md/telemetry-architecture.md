# Telemetry Architecture Overview (Phase 5E)

Status: Complete (Phase 5E) – Hardening + benchmarks finalized
Date: 2025-09-27
Related Plan: `phase5e-plan.md`

---

## 1. Purpose

Provide an architectural blueprint for Ariadne's Phase 5E observability layer: metrics, tracing, events, logging, and health signals—ensuring low overhead, extensibility, and deterministic business behavior.

## 2. Conceptual Model

```
                +-----------------+          +--------------------+
                |  Engine Core    |          | Dynamic Config     |
                | (Pipeline etc.) |          | (Phase 5C)         |
                +--------+--------+          +---------+----------+
                            |                             |
                            | instrumentation             | watches / updates
                            v                             v
                +-----------------+        +---------------------------+
                | Telemetry Facade|<------>| TelemetryPolicy Snapshot  |
                +--+---+---+---+--+        +-------------+-------------+
                    |   |   |   |                         |
        metrics ->|   |   |   |-> events bus            | runtime toggles
             tracing ->|   |                             |
                 logging -> health -----------------------+
                            |
                            | (pure in-process API: HealthSnapshot, metrics handler)
                            v
                +----------------------------+
                |  Adapters (HTTP / CLI /    |
                |  External Export Bridges)  |
                +----------------------------+
```

## 3. Components

- Metrics Provider (Implemented Iteration 1): Abstract factory for instruments (counters, gauges, histograms, timers). Current backends: No-op (zero overhead) and Prometheus (registry + HTTP handler). OTEL bridge deferred.
- Metrics Adapter (Implemented Iteration 1): Bridges legacy `BusinessMetricsCollector` into new provider without double registration (snapshot-based additive sync; future delta optimization).
- Event Bus (Implemented Iteration 2): Pub/sub dispatcher with bounded per-subscriber queues & drop accounting (metrics: published_total, dropped_total{subscriber}).
- Tracer (Implemented Iteration 3): Simple in-process tracer (TraceID/SpanID, parent linkage, attributes). Sampling & external export deferred.
- Logger (Implemented Iteration 3): Structured log wrapper adding correlation fields (trace_id, span_id) with optional domain/component enrichment.
- Health Evaluator (Implemented Iteration 4): Aggregates subsystem probes (rate limiter, resources, pipeline) into cached snapshot (TTL). Thresholds currently heuristic; moving to TelemetryPolicy (Iteration 5).
- HTTP / Endpoint Adapter (Implemented Iteration 5): Serves `/healthz`, `/readyz`, `/metrics` outside core engine; consumes `HealthSnapshot` & provider handler. Not embedded in engine to preserve dependency direction.

## 4. Data Flow

1. Business operations emit internal signals via lightweight inline functions (no mutex on hot paths; atomic counters + lock-free publish attempts).
2. Telemetry Facade translates signals to exporter-specific state only if enabled (fast no-op path otherwise).
3. Event Bus fan-outs structured events to subscribers (logger, future external forwarders). Slow subscribers incur drops counted in metrics.
4. Tracer attaches span context to events/log entries for correlation.
5. Health adapters (when enabled) read aggregated snapshot + last evaluation timestamp from engine (cached with TTL) and serialize over chosen transport (HTTP initially). Core engine never depends on `net/http`.

## 5. Design Principles

- Zero-Allocation Fast Path: Disabled telemetry incurs minimal overhead (branch + atomic read).
- Opt-In Cardinality: Metric labels (tags) strictly limited; default set: {component, outcome, domain(optional hash), asset_type}. High-cardinality labels (URL) prohibited.
- Determinism Preservation: Telemetry must never influence scheduling or business ordering.
- Backpressure Safety: Event bus bounded per subscriber; drops observable & alertable.
- Progressive Adoption: Each facility (metrics, tracing…) independently enable/disable.

## 6. Interfaces (Illustrative)

```go
// Telemetry encapsulates all subsystems.
type Telemetry interface {
    Metrics() MetricsProvider
    Events() EventBus
    Tracer() Tracer
    Logger() Logger
    Health() HealthProbe
    Configure(TelemetryPolicy) error
}
```

## 7. Configuration Mapping

Key fields from `TelemetryPolicy` → subsystem switches (see `phase5e-plan.md`). Runtime application uses atomic pointer swap pattern; subscribers fetch new snapshot on next emission.

## 8. Extensibility Hooks

- MetricsProvider.Register: Custom collectors (e.g. GC stats) may be injected.
- EventBus.Subscribe: External consumer (future WebSocket / remote forwarder).
- Tracer.WithSpan: Wrapper to integrate external context propagation later.

## 9. Performance Targets

| Aspect            | Target                                                                    |
| ----------------- | ------------------------------------------------------------------------- |
| Disabled overhead | < 50ns per signal (validated for no-op path by design; benchmark pending) |
| Metrics increment | < 200ns (hot counters)                                                    |
| Span creation     | < 2µs (sampled)                                                           |
| Event publish     | < 1µs (no contention)                                                     |
| Allocation budget | No heap alloc on disabled paths                                           |

## 10. Open Technical Decisions

1. Exact histogram buckets for rate limiting latency & page processing.
2. Choice of time source abstraction (monotonic clock wrapper) for deterministic tests.
3. Build tag naming for optional OTEL integration (e.g. `otel`).
4. Fallback strategy if exporter HTTP adapter handler fails (self-heal vs. error surfacing).
5. Exact readiness semantics (treat degraded as ready) – proposed: YES; unhealthy / unknown => 503.

## 11. Future Evolution

- Remote subscriber bridging (gRPC / WebSocket) for distributed crawlers.
- Adaptive sampling (error & latency biased) feeding dynamic config updates.
- Multi-tenant namespace segregation (metric prefix partitioning).
- Additional adapters: fetcher variants (browser, API), output sink plugins, external event forwarder.

## 12. Status

Implemented: Metrics Provider, Metrics Adapter, Event Bus, Tracer (adaptive sampler), Logger, Health Evaluator (+ health status gauge & change events), TelemetryPolicy, HTTP Adapter (/healthz,/readyz,/metrics), OTEL Metrics Bridge (label & cardinality guard), Benchmarks (counter/histogram/timer + integrated workload), SLO baseline & overhead report.
Deferred: Remote subscribers, error/latency biased sampling boosts, external trace exporter.

### 13. Adapter Wiring Example

The engine intentionally does not start HTTP servers. A minimal integration:

```go
eng, _ := engine.New(engine.Defaults())
// Optionally adjust telemetry policy at runtime
pol := engine.TelemetryPolicy() // or build custom & eng.UpdateTelemetryPolicy(&custom)

mux := http.NewServeMux()
mux.Handle("/healthz", telemetryhttp.NewHealthHandler(telemetryhttp.HealthHandlerOptions{Engine: eng, IncludeProbes: true}))
mux.Handle("/readyz", telemetryhttp.NewReadinessHandler(telemetryhttp.HealthHandlerOptions{Engine: eng}))
if mp := eng.MetricsProvider(); mp != nil { // expose metrics only if enabled
// Metrics exposition now retrieved via eng.MetricsHandler() facade (Prometheus backend only).
// The engine intentionally does not start HTTP servers. A minimal integration:
```

go
eng, \_ := engine.New(engine.Defaults())
// Optionally adjust telemetry policy at runtime
pol := engine.TelemetryPolicy() // or build custom & eng.UpdateTelemetryPolicy(&custom)

```
http.ListenAndServe(":8080", mux)
```

This separation keeps transport concerns out of core logic and supports alternative adapters (CLI, gRPC) later.
