# Metrics Reference (Phase 5E)

Status: Complete (Phase 5E) – hardening finished; adaptive sampling & workload benchmark added post-completion
Date: 2025-09-27
Related: `phase5e-plan.md`, `telemetry-architecture.md`

---

## 1. Naming Convention

`ariadne_<domain>_<subject>_<unit>` (snake case). Domains:

- assets
- pipeline
- rate_limit
- resources
- crawler
- config
- events (meta counters)

Units / suffixes:

- `_total` (counters)
- `_bytes_total`
- `_seconds` (gauges / histograms base unit)
- `_duration_seconds` (histogram)
- `_inflight` (gauge)

## 2. Current Counters (Pre-Exporter Internal Names)

(Will be bound to public metric names during Iteration 1)

| Internal Field               | Proposed Metric Name                         | Type      | Labels                  | Description                                                      |
| ---------------------------- | -------------------------------------------- | --------- | ----------------------- | ---------------------------------------------------------------- |
| asset.discovered             | ariadne_assets_discovered_total              | counter   | component="asset"       | Candidate asset refs found                                       |
| asset.selected               | ariadne_assets_selected_total                | counter   | component="asset"       | Refs selected after policy                                       |
| asset.skipped                | ariadne_assets_skipped_total                 | counter   | reason                  | Skipped per policy (limit/type)                                  |
| asset.downloaded             | ariadne_assets_downloaded_total              | counter   | status="success"        | Successful downloads                                             |
| asset.failed                 | ariadne_assets_failed_total                  | counter   | error_class             | Failed download attempts                                         |
| asset.inlined                | ariadne_assets_inlined_total                 | counter   | -                       | Inlined small assets                                             |
| asset.optimized              | ariadne_assets_optimized_total               | counter   | optimization            | Assets where optimization occurred                               |
| asset.bytesIn                | ariadne_assets_bytes_in_total                | counter   | -                       | Raw bytes downloaded                                             |
| asset.bytesOut               | ariadne_assets_bytes_out_total               | counter   | -                       | Bytes after optimization                                         |
| asset.rewriteFailures        | ariadne_assets_rewrite_failures_total        | counter   | -                       | Rewrite stage failures                                           |
| pipeline.pages               | ariadne_pipeline_pages_processed_total       | counter   | outcome (success/error) | Pages fully processed                                            |
| pipeline.errors              | ariadne_pipeline_errors_total                | counter   | stage                   | Processing errors                                                |
| rate_limit.decisions         | ariadne_rate_limit_decisions_total           | counter   | result (allow/deny)     | Rate limit decisions                                             |
| rate_limit.latency           | ariadne_rate_limit_decision_duration_seconds | histogram | -                       | Decision latency distribution                                    |
| resources.cacheHits          | ariadne_resources_cache_hits_total           | counter   | tier (memory/disk)      | Cache hits                                                       |
| resources.cacheMisses        | ariadne_resources_cache_misses_total         | counter   | tier                    | Cache misses                                                     |
| config.reloads               | ariadne_config_reloads_total                 | counter   | status (success/fail)   | Dynamic config reload attempts                                   |
| events.dropped               | ariadne_events_dropped_total                 | counter   | subscriber              | Dropped events per subscriber                                    |
| events.published             | ariadne_events_published_total               | counter   | -                       | Total events published (fan-out)                                 |
| internal.cardinalityExceeded | ariadne_internal_cardinality_exceeded_total  | counter   | metric                  | Metrics whose label cardinality exceeded limit (once per metric) |

## 3. Histograms (Initial Buckets Draft)

### Rate Limit Decision Latency

Buckets (seconds): 0.0005, 0.001, 0.002, 0.005, 0.01

### Page Processing Duration (Fixture Scale)

Buckets (seconds): 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5

### Asset Execute Duration (Per Asset)

Buckets (seconds): 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05

## 4. Gauges

| Proposed Name                       | Source                       | Description                                                              |
| ----------------------------------- | ---------------------------- | ------------------------------------------------------------------------ |
| ariadne_resources_memory_inflight   | runtime stats / manager      | Bytes currently held in memory cache                                     |
| ariadne_resources_assets_inflight   | asset strategy (in-progress) | Concurrent active asset executions                                       |
| ariadne_pipeline_pages_inflight     | pipeline scheduler           | Pages concurrently being processed                                       |
| ariadne_rate_limit_tokens_available | rate limiter state           | Current available tokens (aggregate)                                     |
| ariadne_health_status               | engine health evaluator      | Overall engine health (-1 unknown, 0 unhealthy, 0.5 degraded, 1 healthy) |

## 5. Label Cardinality Constraints

- `domain`: optional hashed form only (e.g. fnv32) to limit cardinality
- `page_url`: NEVER allowed as label
- `asset_url`: NEVER allowed as label
- `error_class`: derived classification (timeout, dns, http_4xx, http_5xx, other)
- `optimization`: limited set (whitespace, none, basic)
- `reason`: policy reason categories (limit_exceeded, type_blocked)

## 6. Metric Lifecycle & Stability

- Phase 5E: Metrics in `experimental` stability tier until end-of-phase; changes logged in CHANGELOG.
- Post 5E completion: Promote core metrics (pipeline, rate_limit, resources, assets) to `stable`.

## 7. Validation Tests

- Counter monotonicity checks
- Histogram bucket coverage (p95 within defined buckets)
- Label cardinality enforcement (fail test if dynamic explosion detected)

## 8. Adapter & Sync Semantics (Business Metrics)

Legacy business metrics (rules, strategies, outcomes) are exposed via the `BusinessCollectorAdapter`:

- Snapshot Model: `SyncOnce()` reads cumulative counts from `BusinessMetricsCollector` and adds them to Prometheus counters. Repeated calls would over-count (delta optimization deferred).
- Recommended Usage: Invoke exactly once before first scrape OR refactor to periodic delta sync in future iteration (marked for potential hardening in Iteration 6).
- Future Enhancement: Maintain previous snapshot and compute deltas to permit periodic invocation without duplication.

Implication: Current adapter is a transition mechanism; final design will migrate business logic to emit directly through the metrics provider.

## 9. Open Questions

1. Do we expose per-asset-type download latency histogram or aggregate only? (Start aggregate.)
2. Should cache hit/miss be split by resource class beyond tier? (Defer.)
3. Do we add gauge for backlog depth (queued pages)? (Potential.)

## 10. Roadmap Hooks

Future phases may add: distributed shard metrics (shard_id label), exporter health metrics, adaptive sampling exposure.

## 11. Status

Complete – Providers (noop, prom, otel), event bus, tracing/logging, health evaluator, cardinality guard (prom & otel), overhead benchmarks (micro + integrated workload), adaptive sampling tracer hook.

### Health Metric Rationale

The single numeric health gauge provides low-cardinality readiness insight for dashboards / alerting without a multi-series join:

| Value | Meaning   | Emission Condition                                  |
| ----- | --------- | --------------------------------------------------- |
| -1    | unknown   | No probes registered / evaluator not yet sampled    |
| 0     | unhealthy | Any probe returns unhealthy OR severe threshold met |
| 0.5   | degraded  | Any probe degraded (none unhealthy)                 |
| 1     | healthy   | All probes healthy and sample count thresholds met  |

Pipeline failure ratio thresholds: degraded >=50%, unhealthy >=80% (min 10 samples) to dampen early noise. Resource checkpoint backlog thresholds: degraded >=256, unhealthy >=512 queued entries.
