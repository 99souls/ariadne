# Telemetry Architecture Overview (Phase 5E)

Status: Draft
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
         metrics -->|   |   |   |--> events bus           | runtime toggles
               tracing ->|   |                             |
                      logging -> health ------------------+
```

## 3. Components
- Metrics Provider: Abstract factory for instruments (counters, gauges, histograms) with Prometheus backend and optional OTEL bridge.
- Event Bus: Pub/sub dispatcher with per-subscriber ring buffers & drop accounting.
- Tracer: Context span manager (session → page → stage → asset) with sampling & attribute injection.
- Logger: Structured log wrapper adding correlation fields (trace_id, domain, component, page_url).
- Health Evaluator: Aggregates component probes (rate limiter, resources, asset strategy, config runtime) to readiness/liveness states.

## 4. Data Flow
1. Business operations emit internal signals via lightweight inline functions (no mutex on hot paths; atomic counters + lock-free publish attempts).
2. Telemetry Facade translates signals to exporter-specific state only if enabled (fast no-op path otherwise).
3. Event Bus fan-outs structured events to subscribers (logger, future external forwarders). Slow subscribers incur drops counted in metrics.
4. Tracer attaches span context to events/log entries for correlation.
5. Health endpoints read aggregated views + last evaluation timestamp (avoid active scanning on every request — cached with TTL).

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
| Aspect            | Target                               |
| ----------------- | ------------------------------------- |
| Disabled overhead | < 50ns per signal                     |
| Metrics increment | < 200ns (hot counters)                |
| Span creation     | < 2µs (sampled)                       |
| Event publish     | < 1µs (no contention)                 |
| Allocation budget | No heap alloc on disabled paths       |

## 10. Open Technical Decisions
1. Exact histogram buckets for rate limiting latency & page processing.
2. Choice of time source abstraction (monotonic clock wrapper) for deterministic tests.
3. Build tag naming for optional OTEL integration (e.g. `otel`).
4. Fallback strategy if exporter HTTP handler fails (self-heal vs. error surfacing).

## 11. Future Evolution
- Remote subscriber bridging (gRPC / WebSocket) for distributed crawlers.
- Adaptive sampling (error & latency biased) feeding dynamic config updates.
- Multi-tenant namespace segregation (metric prefix partitioning).

## 12. Status
Draft – pending interface stabilization during Iteration 1 & 2 of Phase 5E.
