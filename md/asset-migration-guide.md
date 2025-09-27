# Asset Subsystem Migration Guide (Phase 5D → 5E Readiness)

Status: Draft (Iteration 8)

## 1. Audience

Projects upgrading from pre-Phase 5D (legacy synchronous asset handling under `internal/assets`) to the new policy-driven strategy in `packages/engine`.

## 2. Summary of Breaking Changes

| Aspect              | Old                          | New                                                   |
| ------------------- | ---------------------------- | ----------------------------------------------------- |
| Location            | `internal/assets` (implicit) | `packages/engine` strategy (explicit)                 |
| Activation          | Always on (implicit)         | `Config.AssetPolicy.Enabled = true`                   |
| Configuration       | Scattered / limited          | Centralized `AssetPolicy` struct                      |
| Concurrency         | Serial                       | Bounded worker pool (`MaxConcurrent`)                 |
| Optimization Events | Separate optimize event      | Coalesced into `asset_download` (Optimizations field) |
| Failure Metrics     | Not explicit                 | `Failed` counter + stage error events                 |
| Discovery Scope     | Basic img/script/stylesheet  | Extended (srcset, preload, media, docs)               |

## 3. Enabling the New System

```go
cfg := engine.Defaults()
cfg.AssetPolicy.Enabled = true
cfg.AssetPolicy.AllowTypes = []string{"img","script","stylesheet"}
eng, err := engine.New(cfg)
```

## 4. Key Policy Fields

| Field          | Purpose                             | Notes                                             |
| -------------- | ----------------------------------- | ------------------------------------------------- |
| Enabled        | Master switch                       | Disabled = bypass all asset stages                |
| MaxBytes       | Per-page cumulative fetch cap       | 0 = unlimited                                     |
| MaxPerPage     | Max selected assets                 | Early exit in Decide stage                        |
| InlineMaxBytes | Threshold for potential inline mode | Current heuristic only (size not probed)          |
| Optimize       | Enable lightweight optimization     | CSS/JS whitespace collapse, simple image meta tag |
| RewritePrefix  | Output path prefix                  | Must start with '/'                               |
| AllowTypes     | Whitelist of asset types            | Empty = allow all not blocked                     |
| BlockTypes     | Explicit block list                 | Evaluated before allow list                       |
| MaxConcurrent  | Worker pool size                    | 0 => auto (capped NumCPU, max 8)                  |

## 5. Metrics Mapping

| New Metric      | Description              | Replacement For             |
| --------------- | ------------------------ | --------------------------- |
| Discovered      | Candidates found         | (n/a)                       |
| Selected        | After policy filter      | (n/a)                       |
| Skipped         | Policy skipped           | (n/a)                       |
| Downloaded      | Successful fetches       | Old implicit success counts |
| Failed          | Fetch errors             | (new)                       |
| Inlined         | Inline mode chosen       | (new)                       |
| Optimized       | Any optimization applied | (new)                       |
| BytesIn         | Pre-optimization bytes   | Old total bytes (approx)    |
| BytesOut        | Post-optimization bytes  | (new distinction)           |
| RewriteFailures | Rewrite stage errors     | (new)                       |

## 6. Events

Current: `asset_download`, `asset_stage_error`, `asset_rewrite`.
Deprecated: `asset_optimize` (merged into download event).

## 7. Deterministic Paths

`/assets/<first2>/<fullhash><ext>` — consumers must ensure static file serving or bundling expects hashed filenames.

## 8. Migration Steps

1. Remove references to legacy internal asset functions (deleted).
2. Enable policy; tune `AllowTypes` and caps.
3. Update any log/event consumers to read optimization data from `asset_download.Optimizations`.
4. If you previously relied on inline heuristic, validate actual size budgets (future enhancement may change heuristic).
5. Add monitoring dashboards for new metrics (naming TBD in exporter integration phase).

## 9. Validation Checklist

- [ ] Tests pass with `AssetPolicy.Enabled=true`.
- [ ] Rewritten HTML references hashed assets.
- [ ] Asset metrics snapshot shows non-zero Discovered/Selected/Downloaded for pages with assets.
- [ ] No unexpected growth in page processing latency (>5%).

## 10. Troubleshooting

| Symptom             | Possible Cause                               | Resolution                                                    |
| ------------------- | -------------------------------------------- | ------------------------------------------------------------- |
| No assets rewritten | Policy disabled or AllowTypes excludes types | Enable policy / adjust AllowTypes                             |
| High Failed count   | Remote errors / network issues               | Inspect `asset_stage_error` events; consider retries (future) |
| BytesIn >> BytesOut | Optimization collapsed whitespace            | Expected if Optimize=true                                     |
| Duplicate assets    | Input HTML duplicates suppressed?            | Dedup keyed by type+url; check differing attributes           |

## 11. Future Safe Changes

The following can evolve without breaking consumers:

- Additional asset types (e.g., fonts, video poster).
- Smarter inline heuristics.
- Additional optimization transforms.

## 12. Deferred / Out of Scope

- Persistent caching layer.
- Distributed execution.
- Advanced image transcoding.

---

End of Migration Guide.
