# SLO & Performance Baselines (Phase 5E)

Status: Draft
Date: 2025-09-27
Related: `phase5e-plan.md`, `metrics-reference.md`

---

## 1. Purpose
Codify initial Service Level Objectives (SLOs) and measurement methodologies for Ariadne’s core operational dimensions prior to implementing Phase 5E instrumentation.

## 2. Baseline Sources
- Phase 5D benchmark (AssetExecute) for per-asset execution cost.
- Existing pipeline test timings (fixtures) for page processing latency.
- Resource manager cache tests for hit/miss ratios under synthetic load.

## 3. Initial SLO Targets
| Domain              | Objective                                   | Target / Window             | Current Confidence |
| ------------------- | ------------------------------------------- | --------------------------- | ------------------ |
| Page Success Rate   | Non-error pages                              | ≥ 99% / rolling 24h         | Medium             |
| Asset Failure Rate  | Failed / selected                            | ≤ 2% / rolling 1h           | Medium             |
| Processing Latency  | Page p95 (standard fixture corpus)           | < 500ms / rolling 1h        | Medium             |
| Rate Limit Latency  | Decision p95                                 | < 2ms / rolling 1h          | Low (pre-metrics)  |
| Event Drop Rate     | Dropped / published                          | < 0.1% / rolling 1h         | Low (not built)    |
| Telemetry Overhead  | CPU + memory delta vs baseline               | <5% CPU, <10% memory        | Low (pre-feature)  |
| Cache Hit Ratio     | (hits / (hits+misses))                       | ≥ 0.7 / rolling 6h          | Medium             |

## 4. Measurement Mapping (Planned)
| SLO Domain          | Metric / Source Example                                           |
| ------------------- | ----------------------------------------------------------------- |
| Page Success Rate   | increase(ariadne_pipeline_pages_processed_total{outcome="success"}[24h]) / total processed |
| Asset Failure Rate  | increase(ariadne_assets_failed_total[1h]) / increase(ariadne_assets_selected_total[1h]) |
| Processing Latency  | histogram_quantile(0.95, rate(ariadne_pipeline_page_duration_seconds_bucket[1h])) |
| Rate Limit Latency  | histogram_quantile(0.95, rate(ariadne_rate_limit_decision_duration_seconds_bucket[1h])) |
| Event Drop Rate     | increase(ariadne_events_dropped_total[1h]) / published (derived)  |
| Telemetry Overhead  | Benchmark diff: telemetry full vs disabled scenario               |
| Cache Hit Ratio     | hits / (hits + misses)                                            |

## 5. Benchmark Plan
Run benchmarks in four modes (Iterations 5):
1. Disabled (baseline)
2. Metrics only
3. Metrics + events
4. Full (metrics + events + tracing @ 20%)

Collect: ns/op, B/op, allocs/op, CPU utilization sample.

## 6. Reliability Considerations
- Use p95 vs p99 initially to reduce noise; revisit after volume scaling.
- Apply 3-point moving average to smooth transient spikes before alerting.
- Document known fixture bias (synthetic HTML complexity lower than prod pages).

## 7. Adjustments Policy
- 3 consecutive breaches → investigation + potential capacity adjustment.
- Persistent over-performance (≥30% margin) → consider tightening target for next release.

## 8. Risks
| Risk                           | Impact | Mitigation |
| ------------------------------ | ------ | ---------- |
| Incomplete instrumentation     | SLO unmeasurable | Phase gating (no sign-off until measurable) |
| Sampling bias (tracing)        | Latency blind spots | Error-biased sampling + metrics correlation |
| Cardinality explosion risk     | Metric cost inflation | Strict label policy review |

## 9. Future Enhancements
- Adaptive SLO targets based on corpus size tiers.
- Automatic anomaly detection vs static thresholds.
- Multi-tenant SLO segmentation.

## 10. Status
Draft – finalize after instrumentation groundwork (Iterations 1–3) and overhead measurements (Iteration 5).
