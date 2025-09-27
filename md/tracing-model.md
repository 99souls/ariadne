# Tracing Model Guide (Phase 5E)

Status: Draft
Date: 2025-09-27
Related: `phase5e-plan.md`, `telemetry-architecture.md`

---

## 1. Purpose
Define canonical span hierarchy, naming, attribute schema, and sampling strategy for Ariadne's tracing subsystem.

## 2. Span Hierarchy
```
CrawlSession (root)
  PageFetch (per URL)
    RateLimitDecision (optional child or event)
    ContentProcess
      AssetExecute (may aggregate multiple downloads as events)
      Rewrite
    SnapshotEmit (optional)
```

## 3. Span Names (Verb-Noun Convention)
| Span                | Name               | Rationale |
| ------------------- | ------------------ | --------- |
| CrawlSession        | crawl.session      | Root context for a logical crawl run |
| PageFetch           | page.fetch         | Network retrieval & response decode |
| RateLimitDecision   | rate_limit.decision| Decision sub-operation (fast) |
| ContentProcess      | page.process       | Transformation & extraction |
| AssetExecute        | asset.execute      | Concurrent asset retrieval/optimization |
| Rewrite             | page.rewrite       | DOM / reference rewrite |
| SnapshotEmit        | snapshot.emit      | Periodic telemetry snapshot emission |

## 4. Core Attributes
| Attribute          | Applies To                    | Example             |
| ------------------ | ----------------------------- | ------------------- |
| crawl_id           | all spans                     | uuid                |
| page_url_hash      | PageFetch+ descendants        | 9aa1ff              |
| domain_hash        | PageFetch+ descendants        | 1f2a3b              |
| status_code        | PageFetch                     | 200                 |
| content_type       | PageFetch                     | text/html           |
| attempt            | PageFetch                     | 1                   |
| retry              | PageFetch                     | false               |
| asset_count        | ContentProcess / AssetExecute | 8                   |
| asset_failed       | ContentProcess / AssetExecute | 1                   |
| bytes_in           | AssetExecute / PageFetch      | 20480               |
| bytes_out          | AssetExecute                  | 20010               |
| rl_result          | RateLimitDecision             | allow               |
| rl_latency_us      | RateLimitDecision             | 314                 |
| optimization_modes | AssetExecute                  | ["whitespace"]      |
| processing_ms      | PageFetch / ContentProcess    | 142.7               |
| error_class        | any (on failure)              | timeout             |

## 5. Events vs Child Spans
- `RateLimitDecision`: Only a child span if latency sampling or detailed analysis enabled; otherwise structured event on PageFetch.
- `AssetExecute`: Either one span per page with aggregated metrics OR per-asset child spans when debug sampling toggled.
- Decision controlled via config flags (e.g. `TracingDetailLevel=basic|detailed`).

## 6. Sampling Strategy
- Baseline: Parent-based, 20% of PageFetch spans sampled.
- Error Bias: If PageFetch results in failure/error_class set, force sample regardless of rate.
- Dynamic Update: Sampling percent & detail level adjustable at runtime via config.

## 7. Context Propagation
- Internal only (no cross-process headers) in Phase 5E.
- Potential future HTTP header injection for distributed crawling agents.
- Trace/Span IDs stored in log fields when tracing enabled.

## 8. Performance Considerations
- Avoid attribute maps on hot path: predeclare attribute structs.
- Reuse buffers for converting ints to strings when needed (or use OTEL attribute types directly).
- Ensure unsampled path avoids heap allocations (guard early and return).

## 9. Testing Strategy
- Span presence tests: ensure hierarchy appears when sampled.
- Attribute integrity: verify required attributes populated, no unexpected empties.
- Sampling toggle test: runtime change affects subsequent spans only.
- Error bias test: forced sampling on failure regardless of base rate.
- Overhead test: benchmark with tracing disabled vs enabled (basic) vs detailed.

## 10. Open Questions
1. Do we include DOM transformation duration separately from total page.process? (Potential sub-span.)
2. Should we capture rate limiter token bucket depth as attribute? (Maybe gauge metric only.)
3. Are per-asset spans valuable for production or only debug? (Likely debug-only to contain cardinality.)

## 11. Status
Draft – refine post prototype (Iteration 3).
