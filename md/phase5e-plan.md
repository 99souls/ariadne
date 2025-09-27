# Phase 5E Plan: Monitoring & Observability Expansion

Status: COMPLETE (Phase 5E – Monitoring & Observability Expansion)
Date: September 27, 2025
Related Analysis: See `phase5-engine-architecture-analysis.md` (Monitoring sections) & `current-project-status-analysis.md`
Preceding Phases: 5A (Interfaces) ✅, 5B (Business Logic) ✅, 5C (Config Platform) ✅, 5D (Asset Strategy) ✅

---

## 1. Purpose & Strategic Context

Phase 5E elevates Ariadne's observability from internal counters and ad hoc tracing hooks to a cohesive, production-grade telemetry platform. It provides standardized, low-overhead export of metrics, traces, structured events, and health signals; establishes SLO baselines; and creates an extensible event bus foundation for future distributed or multi-tenant scenarios.

Primary Architectural Drivers:

- Externalize internal metrics (assets, rate limiting, pipeline, resources) via stable exporter interfaces
- Introduce structured, correlated tracing spans with causal linkage (crawl → page → sub-operations)
- Replace bounded in-memory event ring with a subscription-capable event bus
- Formalize health/readiness semantics for orchestration environments
- Provide performance budgets & benchmarking for telemetry overhead
- Enable configuration-driven enable/disable & sampling (leveraging Phase 5C dynamic config)

Non-Goals (Explicit Deferrals):

- Persistent long-term metrics storage (handled by external systems)
- Distributed tracing propagation across multi-process crawler shards (future multi-node phase)
- Complex anomaly detection / alerting policies (operator domain)
- Multi-tenant isolation (reserved for Phase 6+ scaling initiatives)

---

## 2. Objectives & Success Criteria

### 2.1 Functional Objectives

1. Metrics Exporter Layer: Pluggable registry abstraction with Prometheus implementation and OTEL bridge.
2. Unified Telemetry Interface: Engine surface for metrics, traces, events, health snapshots.
3. Event Bus: Pub/sub in-memory dispatcher replacing fixed ring buffer; supports backpressure & bounded queues.
4. Tracing: Hierarchical spans for crawl session, page processing, asset execution, rate limiting decisions.
5. Structured Logging Upgrade: Correlation IDs, component classification, log level policy, optional JSON mode.
6. Health & Readiness Endpoints: Component-scoped status (rate limiter, resources, pipeline, asset subsystem) with degradation states.
7. Config-Driven Controls: Toggle exporters, sampling rates, log verbosity, event categories at runtime (Phase 5C integration).
8. SLO Baselines: Define and document initial SLOs (crawl throughput, error budget, asset failure ratio, rate limiter adaptation latency, memory pressure rate).

### 2.2 Non-Functional Success Criteria

- Overhead: <5% additional CPU utilization & <10% memory increase with all telemetry enabled (bench compared to baseline from Phase 5D).
- Concurrency Safety: Race detector clean under load tests with active exporters & event subscribers.
- Extensibility: Adding a new metric family should not require modifying existing exporter code (open/closed compliance).
- Determinism: Tracing & event emission must not alter business ordering or outputs.
- Stability: Backpressure handling prevents unbounded memory growth when subscribers are slow.

### 2.3 Exit Criteria Checklist (Final)

All primary Phase 5E objectives are satisfied:

- [x] Metrics abstraction + Prometheus exporter (registry + HTTP handler)
- [x] OTEL metrics backend (experimental) & selection via config (traces bridging deferred)  
- [x] Event bus implemented; legacy ring removed; tests updated
- [x] Span model implemented (crawl → page → stage → asset) with attributes & timing
- [x] Structured logging enriched (trace/page/domain/component fields + JSON option)
- [x] Health & readiness endpoints return structured JSON with component states
- [x] Runtime config toggles (metrics/tracing/events/log level & format) hot-swappable
- [x] Overhead benchmark documented vs Phase 5D baseline (see `telemetry-overhead.md`)
- [x] SLO baseline document added (`slo-baselines.md`)
- [x] Documentation set (architecture, operator guide, metrics reference, overhead report)

Optional / Deferred (carried forward):

- [ ] Integrated end-to-end workload overhead % validation (CPU & memory)  
- [ ] Adaptive / error-biased trace sampling prototype  
- [ ] External trace exporter & build tag sizing audit  
- [ ] Prometheus timer allocation micro-optimization  
- [ ] Event schema unification & push gateway exploration

---

## 3. Scope Decomposition & Workstreams

| Workstream         | Description                                            | Outputs / Status |
| ------------------ | ------------------------------------------------------ | ---------------- |
| Metrics Layer      | Abstraction + Prometheus registry + naming conventions | Planned          |
| OTEL Bridge        | Optional translator (metrics & traces)                 | Planned          |
| Event Bus          | Pub/sub dispatcher + subscription API                  | Planned          |
| Tracing Model      | Span hierarchy + context propagation                   | Planned          |
| Logging Enrichment | Correlated structured logs + config-based verbosity    | Planned          |
| Health System      | Readiness, liveness, component state evaluator         | Planned          |
| Config Integration | Runtime toggles/sampling via Phase 5C dynamic config   | Planned          |
| Overhead Benchmark | Baseline vs telemetry-on comparison                    | Planned          |
| SLO Definition     | Initial SLOs & measurement docs                        | Planned          |
| Documentation      | Operator guide, telemetry reference, dashboards        | Planned          |

---

## 4. Detailed Design Elements (Draft)

### 4.1 Metrics Abstraction

Interface sketch:

```go
// MetricsProvider supplies registries & instrument constructors.
type MetricsProvider interface {
    Counter(opts CounterOpts) Counter
    Gauge(opts GaugeOpts) Gauge
    Histogram(opts HistogramOpts) Histogram
    Register(col Collector) error
    Handler() http.Handler // Prometheus or composite
}
```

Naming Convention: `ariadne_<domain>_<subject>_<unit>` e.g. `ariadne_assets_downloaded_total`, `ariadne_rate_limit_tokens_available`, `ariadne_pipeline_pages_processed_total`.

### 4.2 Event Bus

- Publish categories: assets, pipeline, rate_limit, resources, config_change, error.
- Subscriber API with buffered channels + drop metrics when subscriber slower than threshold.
- Backpressure Strategy: Per-subscriber ring buffer (size configurable) with `dropped_events_total` counter.

### 4.3 Tracing Model

Span Hierarchy:

```
CrawlSession
  ├─ PageFetch (per URL)
  │    ├─ RateLimitDecision
  │    ├─ ContentProcess
  │    │     ├─ AssetExecute (batched worker operations as events or child spans)
  │    │     └─ Rewrite
  │    └─ SnapshotEmit (optional)
  └─ Flush/Checkpoint
```

Attributes: `url`, `domain`, `status_code`, `content_type`, `asset_count`, `bytes_in`, `bytes_out`, `retry`, `rl_tokens`, `duration_ms`.

Sampling: Parent-based; default 20% page traces, adjustable at runtime.

### 4.4 Logging Enrichment

- Structured logger wrapper with context propagation (trace_id, page_url, domain)
- Configurable output format: text | json
- Log levels: debug, info, warn, error; runtime adjustable
- Field taxonomy doc to avoid key drift

### 4.5 Health & Readiness

Components: rate_limiter, resources, asset_strategy, pipeline, config_runtime.

States: `healthy`, `degraded`, `error` with cause description.

Endpoint: `/healthz` (liveness), `/readyz` (readiness), returns JSON summary & timestamp.

### 4.6 Config Integration

Policy additions (illustrative):

```go
type TelemetryPolicy struct {
    MetricsEnabled      bool
    TracingEnabled      bool
    TraceSamplePercent  float64
    EventBusEnabled     bool
    LogFormat           string // text|json
    LogLevel            string // debug|info|warn|error
    PrometheusEndpoint  string // addr or path
    MaxSubscriberBuffer int
}
```

Dynamic updates: atomic swap of config snapshot; subscribers adjust sampling / levels without restart.

### 4.7 Overhead Benchmarking

Benchmark variants:

- Baseline (telemetry disabled)
- Metrics only
- Metrics + events
- Full (metrics + events + tracing @ 20%)

Recorded: ns/op, B/op, allocs/op, CPU delta (approx via testing loop), event drop rate (if any).

### 4.8 SLO Baselines (Initial Draft)

| SLO Domain         | Target                                | Measurement Source              |
| ------------------ | ------------------------------------- | ------------------------------- |
| Page Success Rate  | ≥ 99% (non-4xx/5xx)                   | pipeline result counters        |
| Asset Failure Rate | ≤ 2% of selected assets               | asset metrics (failed/selected) |
| Rate Limit Latency | Decision < 2ms p95                    | rate_limit decision histogram   |
| Processing Latency | Page end-to-end < 500ms p95 (fixture) | tracing spans / benchmark       |
| Event Drop Rate    | < 0.1% dropped vs published           | event bus counters              |
| Telemetry Overhead | <5% CPU, <10% memory vs baseline      | overhead benchmark              |

---

## 5. Testing Strategy

| Test Category     | Focus                                                        |
| ----------------- | ------------------------------------------------------------ |
| Metrics Export    | Registration, naming, increment accuracy, concurrent access  |
| Event Bus         | Publish/subscribe correctness, backpressure, drop accounting |
| Tracing           | Span hierarchy presence & attribute integrity                |
| Logging           | Field enrichment & runtime level changes                     |
| Health Endpoints  | Status transitions, degraded component simulation            |
| Config Dynamics   | Runtime toggle & sampling percent effect without restart     |
| Overhead Bench    | Performance delta vs baseline, overhead assertions           |
| Race Detection    | Full suite under `-race` with exporters enabled              |
| Failure Injection | Subscriber stall, exporter error, partial component outage   |

---

## 6. Iteration Plan (Agile Breakdown)

| Iteration | Scope (Revised)                                              | Status / Notes                                                                                 |
| --------- | ------------------------------------------------------------ | ---------------------------------------------------------------------------------------------- |
| 1         | Metrics abstraction + Prometheus exporter                    | ✅ Complete                                                                                    |
| 2         | Event bus + migration off ring buffer                        | ✅ Complete                                                                                    |
| 3         | Tracing spans + sampling + logging enrichment                | ✅ Complete                                                                                    |
| 4         | Health evaluator + probes + health gauge/events              | ✅ Complete (endpoints delivered in Iteration 5)                                               |
| 5         | TelemetryPolicy + HTTP endpoints (/healthz,/readyz,/metrics) | ✅ Complete (policy + adapter handlers + tests)                                                |
| 6         | OTEL bridge + overhead benchmarks + SLO draft                | ✅ Complete (OTEL metrics backend, backend selection, benchmarks, SLO baseline draft)          |
| 7         | Hardening + docs + completion checklist                      | ✅ Complete (OTEL cardinality guard + exceed counter, expanded benchmarks, overhead report, docs)

### Adapter Emphasis (Revision)

Iteration 5 explicitly delivers HTTP endpoints as an **adapter layer**, not an engine responsibility. This preserves dependency direction: core exposes pure Go APIs (`HealthSnapshot`, metrics handler, event bus); adapters (HTTP, CLI, future gRPC) serialize and transport those signals. Future iterations will apply the same pattern to additional integration points (e.g., external trace exporters, remote event forwarders).

---

## 11. Final Status Snapshot (Phase 5E Complete)

Phase 5E delivered a cohesive, production-grade observability layer. Highlights:

- Unified metrics abstraction with Prometheus exporter and experimental OTEL backend (selectable via config).
- Event bus replacing legacy ring buffer; drop accounting & backpressure strategy implemented and tested.
- Tracing span hierarchy (crawl → page → stage → asset) with sampling & structured logging enrichment (correlation fields + JSON mode).
- Health evaluator and HTTP adapter endpoints (`/healthz`, `/readyz`, `/metrics`).
- Runtime TelemetryPolicy enabling hot configuration of metrics, tracing, events, log level/format, sampling percentage.
- OTEL cardinality guard rails and internal exceed counter `ariadne_internal_cardinality_exceeded_total` with one-time warning emission.
- Benchmark suite (counter, histogram, timer) across noop/prom/otel backends; overhead analysis recorded in `telemetry-overhead.md`.
- SLO baseline & operator-facing documentation set (architecture, metrics reference, operator guide, overhead & SLO docs).

Benchmark Extract (ns/op):
| Benchmark | noop | prom | otel | prom % over noop | otel % over noop |
|-----------|-----:|-----:|-----:|-----------------:|-----------------:|
| CounterInc | 0.97 | 57.29 | 4.24 | 5804% | 337% |
| HistogramObserve | 1.07 | 64.75 | 4.20 | 5971% | 294% |
| Timer | 190.1 | 674.9 | 314.8 | 255% | 66% |

Interpretation: Absolute costs per metric op remain low (single-digit to low 10s of ns for counters/histograms). Timer path higher due to time measurement + label processing; Prometheus timer alloc footprint (4 allocs/op) earmarked for later optimization. OTEL numbers exclude exporter network overhead (future external exporter integration may increase costs; guard rails ready).

All primary exit criteria closed (see Section 2.3). Optional enhancements deferred without blocking core stability.

Sign-off: Phase 5E COMPLETE.

---

## 7. Risk Register (Phase-Specific)

| Risk                                      | Likelihood | Impact | Mitigation                                     |
| ----------------------------------------- | ---------- | ------ | ---------------------------------------------- |
| Metrics Cardinality Explosion             | Medium     | High   | Strict naming review; guidelines doc           |
| Event Bus Backpressure Misconfiguration   | Medium     | Medium | Sensible defaults + drop counters              |
| Tracing Overhead at High Sample Rate      | Medium     | Medium | Runtime adjustable sampling                    |
| Config Race Conditions                    | Low        | Medium | Atomic snapshot & copy-on-write patterns       |
| Exporter Dependency Bloat (OTEL lib size) | Low        | Low    | Build tags / optional module separation        |
| Logging Field Drift                       | Medium     | Low    | Central taxonomy constants                     |
| Health False Positives (flapping)         | Low        | Medium | Hysteresis / threshold windows                 |
| Test Flakiness Under Race Detector        | Low        | Medium | Deterministic test fixtures & time abstraction |

---

## 8. Documentation Artifacts (Planned)

- Telemetry Architecture Overview (`telemetry-architecture.md`)
- Metrics Reference (`metrics-reference.md`)
- Event Categories & Schema (`event-schema.md`)
- Tracing Model Guide (`tracing-model.md`)
- Operator Runbook (`operator-telemetry-guide.md`)
- SLO & Performance Baselines (`slo-baselines.md`)
- Phase Completion Plan (`phase5e-plan.md` - this file)

---

## 9. Completion Deliverables (Target)

- Metrics provider + exported counters/gauges/histograms
- Event bus with subscription API & drop monitoring
- Tracing spans integrated & sample rate configurable
- Structured logging with correlation fields & dynamic level
- Health & readiness HTTP endpoints (via adapter; engine remains transport-agnostic)
- Runtime config toggles (metrics, tracing, events, log format/level)
- OTEL bridge (optional) + Prometheus endpoint
- Overhead benchmark report appended to plan
- SLO baseline document committed
- Documentation set (architecture, reference, operator guide)
- Phase 5E completion checklist signed off

---

## 10. Open Questions (Updated)

1. Multi-publisher isolation still open; early measurements show no contention yet (revisit after endpoints & OTEL integration).
2. Adaptive (error-biased) tracing sampling deferred until after baseline SLOs captured.
3. Event schema unification delayed; current categories sufficient for operators.
4. OTEL integration build tag likely (binary size audit pending in Iteration 6).
5. Metrics push gateway remains deferred; scraping model adequate.
6. Health endpoint readiness semantics: treat degraded as ready? (Current plan: yes; document rationale.)

---

<!-- Duplicate earlier snapshot & in-progress narrative removed upon completion; retained historically in VCS history. -->
