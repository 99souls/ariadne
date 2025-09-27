# Phase 5E Progress Log

Status: INITIATED (Iteration 1: Metrics Abstraction / Exporter Scaffold)

## Iteration 1 Scope

- Define minimal metrics abstraction (Counter, Gauge, Histogram, Timer helper) with pluggable backend.
- Provide no-op implementation (zero allocations hot path).
- Provide Prometheus-backed implementation (wrapper over client_golang) with safe registration and idempotent lookups.
- Bridge existing business metrics (rule/strategy/outcome) via adapter without double counting.
- Introduce engine-level initialization wiring (select provider via config/env; default: prometheus if available, else noop).
- Add unit tests for abstraction (concurrency, label cardinality guard, no-op invariants).

## Design Decisions (Running)

1. Compatibility First: Existing `BusinessMetricsCollector` remains; new provider will inject a `MetricFactory` that the collector can optionally use in future refactor. For now we expose an adapter that surfaces existing counters through the new interface names so dashboards remain stable.
2. Allocation Budget: Interface methods avoid building label maps each call. We expose With(labels...) pattern returning a bound metric handle where possible (phase 2 enhancement). Iteration 1: simple direct Record/Inc with vararg label values (slice reuse internal via sync.Pool in Prometheus impl if needed—deferred unless profiler shows pressure).
3. Error Handling: Registration failures (duplicate name) are surfaced once at creation time; subsequent Get/Inc calls are safe (no panics). We log (debug) and fall back to a no-op instrument if registration fails.
4. Metrics Naming: Canonical names come from `metrics-reference.md`. Provider enforces snake_case; rejects invalid characters.
5. Label Cardinality Guard: Basic guard warns (once per metric) if label cardinality grows beyond threshold (default 100 distinct combinations) – instrumentation for later adaptive sampling.

## Checklist

- [x] Interfaces & option structs committed
- [x] No-op provider + tests
- [x] Prometheus provider skeleton (registry, creation, handlers)
- [ ] Adapter for existing business metrics (phase 1: read-only exposure)
- [x] Engine wiring (config flag + default)
- [x] Basic tests pass (CI green)
- [ ] Documentation update linking abstraction to existing `telemetry-architecture.md`

## Risks / Watch

- Potential duplication of registrations if both legacy exporter and new provider register same metric names (mitigate by deferring migration until adapter complete and disabling duplicate registration path).
- Latency overhead of label slice allocation (measure after initial commit).
- Metric cardinality explosion (guard instrumentation early).

## Next Action

Implement adapter mapping existing BusinessMetricsCollector counters into the new Provider interface without double registration; then update telemetry architecture doc to reference the new metrics package.
