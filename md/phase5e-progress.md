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
- [x] Adapter for existing business metrics (phase 1: read-only exposure)
- [x] Engine wiring (config flag + default)
- [x] Basic tests pass (CI green)
- [x] Documentation update linking abstraction to existing `telemetry-architecture.md`

## Risks / Watch

- Potential duplication of registrations if both legacy exporter and new provider register same metric names (mitigate by deferring migration until adapter complete and disabling duplicate registration path).
- Latency overhead of label slice allocation (measure after initial commit).
- Metric cardinality explosion (guard instrumentation early).

## Iteration 1 Status

Completed. All planned scope delivered (adapter snapshot caveat documented). Deltas (possible future hardening): adapter delta mode, cardinality warning emission metric.

## Next Action (Iteration 2 Kickoff)

Scaffold Event Bus:

- Define interfaces (Publish, Subscribe, Unsubscribe, Stats)
- Implement bounded per-subscriber ring buffer with drop counter
- Integrate minimal metrics (events published, dropped) via provider
- Add basic tests (single subscriber, multiple subscribers, slow subscriber drop)

---

## Iteration 2: Event Bus & Event Telemetry (Complete ✅)

### Scope

1. In-memory event bus (fan-out) with bounded subscriber queues (prevent unbounded memory growth)
2. Backpressure handling via non-blocking publish + per-subscriber drop accounting
3. Metrics instrumentation: `events_published_total`, `events_dropped_total{subscriber="<id>"}`
4. Subscription lifecycle management (unsubscribe closes channel)
5. Stats surface for debugging / future operator endpoints
6. (Planned later in iteration) Engine integration point + documentation + tracing correlation fields

### Checklist

- [x] Interfaces & core bus struct
- [x] Publish logic with drop handling
- [x] Metrics counters wired (noop + provider-safe)
- [x] Basic tests: delivery, multi-subscriber fan-out, drop behavior
- [x] Engine wiring (bus initialized regardless of metrics enabled)
- [x] Documentation updates (telemetry architecture + metrics reference)
- [ ] Race detector validation (scheduled early Iteration 3 hardening pass)
- [ ] Optional: per-category counters (deferred)

### Notes

- Current implementation copies subscriber slice under read lock then publishes outside lock to minimize contention.
- Subscriber ID label kept numeric to control cardinality; consider hashing stable component names later once stable subscription registry exists.
- Drop strategy: silent (no error) + metrics increment — intentional to avoid cascading failures in hot paths.
- Further hardening: expose `SubscribeWithID(name string, ...)` to allow semantic labels without cardinality risk (deferred).

### Completion Summary

Event Bus implemented and integrated: engine now exposes `EventBus()` returning a live bus instance. Metrics counters for published and dropped events are registered when metrics provider is active (noop otherwise). Tests confirm publish/subscribe semantics, multi-subscriber fan-out, and drop accounting. Documentation updated to move Event Bus from planned to implemented; metrics reference includes new counters. Outstanding validation (race + micro-bench) rolled forward to Iteration 3 hardening.

### Deferred / Follow-Ups

- Race detector suite expansion (`go test -race ./...`) after tracing scaffolding merges to avoid duplicate churn.
- Per-category counters once event category taxonomy stabilizes.
- Possible `SubscribeWithID` for semantic subscriber labels with controlled cardinality.

---

## Iteration 3: Tracing & Correlated Logging (Complete ✅)

### Scope

1. Minimal tracing API (Tracer, Span, SpanContext) with noop + simple in-process implementation.
2. Hierarchical spans (root → child) with parent linkage & start/end timing.
3. Context propagation helpers + extraction of TraceID/SpanID for correlation (events/logging in later sub-step).
4. Engine wiring: always initialize tracer (enabled); future config flag for disabling.
5. Tests covering span lifecycle, parent/child relationships, timing order, noop behavior.
6. Update architecture doc to reflect tracing scaffold.

### Checklist

- [x] Tracing interfaces & types
- [x] No-op tracer
- [x] Simple tracer implementation (random IDs, parent linkage)
- [x] Basic tests (hierarchy, end, attributes, timing)
- [x] Engine wiring + accessor `Tracer()`
- [x] Architecture doc updated
- [x] Event correlation (PublishCtx with automatic TraceID/SpanID enrichment)
- [x] Logging adapter scaffolding (slog wrapper with trace/span fields)
- [x] Race detector run (preliminary manual trigger pending broader suite) _defer full suite to Iteration 4 health checks_
- [x] Micro-benchmark (publish with & without active span)

### Notes

- ID generation uses crypto/rand hex; could swap to faster source if profiling shows overhead.
- Attribute storage kept internal; future iteration may expose snapshot or exporter bridge.
- Tracer currently always enabled; config flag will be added once policy object is introduced.

### Completion Summary

Tracing scaffold, event correlation (`PublishCtx`), correlated logging adapter, and micro benchmark implemented. Engine exposes tracer; logs and events now carry trace/span IDs when context has an active span. Race detector full-suite pass deferred to Iteration 4 when health probes are added to avoid duplicated work; spot checks executed locally. Iteration 3 scope closed.

---

## Iteration 4: Health Evaluator & Subsystem Probes (Complete ✅)

### Scope

1. Health evaluator with TTL-cached snapshot aggregating probe results.
2. Initial probes: rate limiter, resource manager, pipeline processing.
3. Heuristics:
   - Rate limiter: open circuit count -> degraded/unhealthy thresholds.
   - Resources: checkpoint backlog threshold (initial static heuristic).
   - Pipeline: failure ratio vs successes (>50% = degraded).
4. Engine wiring + accessor (`HealthSnapshot(context.Context)`).
5. Unit tests for evaluator (caching, rollup precedence) – done.
6. Documentation updates (architecture doc: add Health section + status table refresh).
7. Progress log update (this section) & iterate heuristics if needed.
8. Race detector full suite pass including new package.

### Checklist

- [x] Evaluator implementation (probe interface, TTL caching, rollup logic)
- [x] Evaluator tests (cache reuse, degraded/unhealthy precedence)
- [x] Engine probes (limiter/resource/pipeline) wired
- [x] Pipeline probe compile fix (use `TotalFailed` not non-existent `TotalErrors`)
- [x] Accessor method `HealthSnapshot(ctx)` on Engine
- [x] Architecture doc updated (health implemented, tracer/logging status refreshed)
- [x] Race detector run across repo (`go test -race ./...`) and note results (full suite clean)
- [x] Heuristic tuning pass (document threshold rationale)
- [x] Optional: emit health change events / metrics gauge (implemented: gauge `ariadne_health_status`, event category `health` with `health_change` type)

### Completion Summary

Core evaluator + three probes implemented (rate limiter, resources, pipeline) with tiered thresholds:

- Pipeline failure ratio: degraded >=50%, unhealthy >=80% (min 10 samples to suppress startup noise).
- Resource checkpoint backlog: degraded >=256 queued, unhealthy >=512 (initial heuristic scaled to default buffer expectations).
- Rate limiter: existing circuit heuristic retained (some open => degraded; majority open => unhealthy).

Health instrumentation additions:

- Numeric gauge `ariadne_health_status` (-1 unknown, 0 unhealthy, 0.5 degraded, 1 healthy).
- Transition events (`category=health type=health_change`) emitted only on state changes (first snapshot suppressed).
- Unit test `TestHealthChangeEvent` validates emission logic and TTL-based recomputation.

Docs updated (architecture, metrics reference, operator guide) with mapping + alert suggestions. Full race detector run clean; no contention surfaced. Iteration 4 closed.

### Deferred / Follow-Ups

- Configurable TelemetryPolicy (thresholds, TTL tuning, enable/disable probes)
- Optional one-hot state gauge variant (labels) if dashboards need discrete series
- Additional probes: asset backlog saturation, config reload failures, external sink availability, span exporter health (future tracing evolution)

---

## Iteration 5: TelemetryPolicy & HTTP Health/Readiness Endpoints (In Progress 🚧)

### Rationale

Before expanding outward (OTEL bridge, SLO doc), we need a configuration surface to externalize currently hard‑coded telemetry heuristics and expose health + metrics via stable HTTP endpoints. This iteration pulls forward the "Config Integration" and part of the original Iteration 4/5 plan to reduce future refactors.

### Scope

1. Define `TelemetryPolicy` struct (dynamic thresholds, probe TTL, enable flags, sampling %, subscriber buffer size).
2. Add atomic policy holder + `UpdateTelemetryPolicy(policy TelemetryPolicy)` with safe fan‑out (metrics/tracer/event bus adjustments where applicable).
3. Extract health heuristic constants in `engine.go` to policy (pipeline failure ratio thresholds, resource backlog tiers, limiter open ratios, min sample window).
4. Introduce lightweight HTTP server (separate goroutine) wiring:
   - `/healthz` (liveness + overall + per component rollup simplified)
   - `/readyz` (readiness: overall must be healthy OR degraded without any unhealthy components; include last change timestamp)
   - `/metrics` (delegate to provider handler when Prometheus enabled)
5. JSON schema: `{ overall:"healthy|degraded|unhealthy|unknown", components:[{name,status,details}], checked_at, previous, changed_at }` (ready endpoint identical plus readiness boolean field).
6. Add tests: policy update adjusts thresholds (simulate transition); health/ready endpoint handlers return valid JSON & correct HTTP codes (200 healthy/degraded, 503 unhealthy for readiness).
7. Documentation updates: plan & progress (this file), operator guide (reference HTTP endpoints), metrics reference (no changes except maybe note for health gauge already present).

### Checklist

- [x] TelemetryPolicy struct + defaults
- [x] Engine integration (store + accessor + update method)
- [x] Heuristics moved from constants to policy
- [ ] HTTP server + handlers (`/healthz`, `/readyz`, `/metrics` passthrough)
- [ ] JSON schema & status code semantics
- [x] Unit tests (policy update) _(endpoints test pending)_
- [x] Docs updated (operator guide endpoints section, plan adjustments)
- [x] Progress log updated (this section)

### Notes

- Keep server optional: start only if any endpoint enabled or metrics provider is Prometheus.
- Avoid bringing in full router dependency; use stdlib `http` only to minimize footprint.
- Policy updates should not block hot paths; use atomic pointer swap pattern.

### Deferred (Post Iteration 5)

- OTEL bridge (metrics/traces) & build tag
- Overhead benchmark harness & SLO baseline doc
- Additional probes & one-hot health gauge variant

---
