# Operator Telemetry Guide (Phase 5E)

Status: Draft
Date: 2025-09-27
Related: `phase5e-plan.md`, `metrics-reference.md`, `event-schema.md`, `tracing-model.md`

---

## 1. Purpose

Provide operational runbook for enabling, configuring, and interpreting Ariadne telemetry signals (metrics, events, traces, logs, health) in production environments.

## 2. Quick Start

1. Enable metrics & event bus in config:

```
telemetry:
  metricsEnabled: true
  eventBusEnabled: true
  prometheusEndpoint: ":9090"
```

2. (Optional) Enable tracing (20% sampling):

```
telemetry:
  tracingEnabled: true
  traceSamplePercent: 20
```

3. Start engine and run the telemetry HTTP adapter (or integrated main) to expose `/metrics`, `/healthz`, `/readyz`.

## 3. Key Dashboards (Recommended Panels)

- Crawl Throughput: `rate(ariadne_pipeline_pages_processed_total[5m])`
- Error Budget: `sum(increase(ariadne_pipeline_errors_total[1h])) / sum(increase(ariadne_pipeline_pages_processed_total[1h]))`
- Asset Failure Ratio: `increase(ariadne_assets_failed_total[10m]) / increase(ariadne_assets_selected_total[10m])`
- Rate Limit Latency p95: `histogram_quantile(0.95, sum(rate(ariadne_rate_limit_decision_duration_seconds_bucket[5m])) by (le))`
- Event Drop %: `increase(ariadne_events_dropped_total[5m]) / (increase(ariadne_events_dropped_total[5m]) + increase(ariadne_pipeline_pages_processed_total[5m]))`
- Resource Cache Hit Rate: `increase(ariadne_resources_cache_hits_total[5m]) / (increase(ariadne_resources_cache_hits_total[5m]) + increase(ariadne_resources_cache_misses_total[5m]))`
- Engine Health: `ariadne_health_status` (target = 1; alert if < 1 for 5m, critical if 0)

## 4. Interpreting Signals

| Signal                        | Symptom                               | Likely Cause                          | Action                                                |
| ----------------------------- | ------------------------------------- | ------------------------------------- | ----------------------------------------------------- |
| rising asset_failed_total     | External site instability             | Transient network / blocking          | Monitor; consider retry policy tuning                 |
| high event drop %             | Slow subscriber                       | Downstream processing or small buffer | Increase buffer size; optimize consumer               |
| elevated rate_limit latency   | Domain burst or token under-provision | Adaptive algorithm adjusting          | Observe; if persists, bump initial tokens             |
| low cache hit rate            | Cold start or eviction                | Working set exceeds memory            | Increase memory budget or tune eviction               |
| processing latency p95 breach | Heavy pages or asset spikes           | Large assets or complex DOM           | Introduce adaptive sampling / optimize asset pipeline |

## 5. Runtime Adjustments

Adjust telemetry parameters via dynamic config (example patch):

```
telemetry:
  traceSamplePercent: 5   # reduce sampling
  logLevel: "warn"        # suppress debug/info
  maxSubscriberBuffer: 2048
```

Changes apply atomically; verify via config_change.apply events.

## 6. Event Consumption Patterns

- Logger Subscriber: Converts event envelopes to structured logs (debug mode).
- Metrics Augmentor: (Future) Derive secondary metrics from event stream.
- External Forwarder: (Future) Streams events to queue / broker.

## 7. Health & Readiness

Endpoints are provided by an HTTP adapter layer (not the engine itself) to keep core business logic transport‑agnostic.

- `/healthz`: Always 200 if process responsive; returns JSON including overall and component statuses.
- `/readyz`: 200 if overall status is healthy OR degraded (no components unhealthy); 503 if unhealthy or unknown.
- Numeric gauge `ariadne_health_status` mapping: -1 unknown, 0 unhealthy, 0.5 degraded, 1 healthy.
- Transition events (`category=health type=health_change`) emitted on status changes (initial state suppressed) – useful for alert correlation / annotation.
- Readiness treats degraded as acceptable to avoid premature restarts during transient dips.

## 8. Alerting Suggestions

| Alert                  | Expression                                                                                                   | Threshold            |
| ---------------------- | ------------------------------------------------------------------------------------------------------------ | -------------------- |
| High Asset Failure     | increase(ariadne_assets_failed_total[15m]) / increase(ariadne_assets_selected_total[15m]) > 0.05             | Investigate upstream |
| Event Drops            | increase(ariadne_events_dropped_total[5m]) > 0                                                               | Inspect subscribers  |
| High RL Latency        | histogram_quantile(0.95, sum(rate(ariadne_rate_limit_decision_duration_seconds_bucket[10m])) by (le)) > 0.01 | Performance tune     |
| Processing Latency p95 | page processing p95 > SLO target                                                                             | Optimize or scale    |
| Low Cache Hit Rate     | hit rate < 0.6 for 30m                                                                                       | Capacity review      |
| Health Degraded        | ariadne_health_status == 0.5 for 5m                                                                          | Investigate probe    |
| Health Unhealthy       | ariadne_health_status == 0 for 1m                                                                            | Immediate action     |

## 9. Tracing Usage

- Filter spans by `crawl_id` to isolate a single crawl session.
- Investigate slow pages: look for `asset.execute` span dominance.
- Correlate asset failures: sample forced on error_class → analyze cluster.

## 10. Logging Strategy

- Default level `info`; elevate to `debug` only for short diagnostic windows.
- Ensure log aggregation system preserves JSON structure when enabled.

## 11. Capacity Planning

| Metric                           | Scaling Indicator               | Action                                            |
| -------------------------------- | ------------------------------- | ------------------------------------------------- |
| events_dropped_total             | sustained growth                | Increase buffer / add consumer                    |
| rate_limit_decision_duration p95 | rising                          | Optimize limiter or add concurrency               |
| assets_inflight gauge            | near MaxConcurrent consistently | Evaluate raising concurrency or optimizing assets |

## 12. Security & Privacy

- URLs hashed; raw not logged by default.
- Configuration values not logged; only key names in events.
- Ensure Prometheus endpoint protected if running in multi-tenant cluster.

## 13. Troubleshooting Checklist

1. Event drop anomaly → Inspect subscriber goroutines (pprof) & buffer size.
2. Elevated latency → Compare spans vs metrics to isolate pipeline stage.
3. Missing metrics → Verify metricsEnabled flag & /metrics endpoint status.
4. Trace gaps → Confirm sampling rate & error bias logic engaged.

## 14. Roadmap Hooks

- Adaptive sampling (error/latency triggered).
- Multi-tenant metric partitioning.
- External event forwarder plugin architecture.

## 15. Status

Updated after Iterations 1–4. Pending: TelemetryPolicy & HTTP adapter implementation (Iteration 5), then OTEL bridge & SLO baseline.
