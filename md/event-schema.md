# Event Categories & Schema (Phase 5E)

Status: Draft
Date: 2025-09-27
Related: `phase5e-plan.md`, `telemetry-architecture.md`

---

## 1. Purpose
Define canonical event categories, payload schema, evolution rules, and subscriber/backpressure semantics for the Phase 5E event bus.

## 2. Categories (Initial Set)
| Category       | Description                                      | Example Producers             |
| -------------- | ------------------------------------------------ | ----------------------------- |
| asset          | Asset discovery, download, optimization, rewrite | AssetStrategy                 |
| pipeline       | Page lifecycle milestones                        | Pipeline coordinator          |
| rate_limit     | Token decisions, circuit events                  | RateLimiter                   |
| resources      | Cache operations, eviction, spill                | ResourceManager               |
| config_change  | Dynamic config reload / apply                    | Config runtime                |
| error          | High-level recoverable errors (classified)       | Various (wrapped emission)    |

## 3. Envelope
```json
{
  "ts": "2025-09-27T12:34:56.789Z",
  "category": "asset",
  "type": "download",        // category-specific subtype
  "level": "info",           // info|warn|error|debug
  "trace_id": "abc123",      // optional (if tracing enabled)
  "span_id": "def456",       // optional
  "seq": 1024,                // monotonically increasing per process (uint64 wrap ok)
  "payload": { /* type-specific object */ }
}
```

## 4. Payload Schemas (Draft)

### asset.download
```json
{
  "url_hash": "e3b0c4",          // truncated hash of original URL
  "type": "image",               // normalized asset type
  "bytes_in": 12345,
  "bytes_out": 12001,
  "duration_ms": 4.2,
  "optimizations": ["whitespace"],
  "status": "success"            // success|failed
}
```

### asset.stage_error
```json
{
  "stage": "download",          // download|optimize|rewrite
  "error_class": "timeout"      // timeout|dns|http_4xx|http_5xx|other
}
```

### asset.rewrite
```json
{ "count": 12 }
```

### pipeline.page
```json
{
  "url_hash": "9aa1ff",
  "domain_hash": "1f2a3b",
  "status": "success",          // success|error
  "duration_ms": 142.7,
  "assets_selected": 8,
  "assets_failed": 1
}
```

### rate_limit.decision
```json
{
  "domain_hash": "1f2a3b",
  "result": "allow",             // allow|deny|backoff
  "tokens": 4,
  "latency_us": 317
}
```

### resources.cache
```json
{
  "op": "hit",                  // hit|miss|evict|spill
  "tier": "memory",             // memory|disk
  "duration_us": 12
}
```

### config_change.apply
```json
{
  "version": 17,
  "status": "success",          // success|failed
  "changed_keys": ["telemetry.TraceSamplePercent"],
  "duration_ms": 3.1
}
```

### error.general
```json
{
  "component": "pipeline",
  "error_class": "panic_recovered",
  "message": "recovered from panic in stage X"
}
```

## 5. Evolution Rules
- Additive only during Phase 5E (no field removal / renaming).
- New categories require doc + tests + version entry in CHANGELOG.
- Field Stability Levels:
  - core: guaranteed after 5E completion (url_hash, category, type, ts)
  - extended: subject to modification (optimizations list, internal counters)

## 6. Backpressure & Delivery Semantics
- Delivery: at-most-once to each subscriber (no retries; best-effort).
- Bounded per-subscriber ring buffer; on overflow oldest dropped; increments `ariadne_events_dropped_total{subscriber}`.
- Sequence `seq` monotonic per process; gaps indicate dropped events.

## 7. Security & Privacy
- Raw URLs never emitted (hash only) to avoid leaking crawl targets.
- No user content excerpts beyond metadata sizes.
- Config diffs limited to key names (values not logged unless explicitly safe & non-sensitive).

## 8. Testing Strategy
- Schema validation tests (JSON struct marshaling, required fields non-empty where applicable).
- Drop scenario tests (forced overflow) verifying counter increment + seq gap.
- Hash collision improbability not tested (cryptographic hash assumption); ensure truncation length sufficient (6–8 hex chars) -> collision probability negligible for operational horizon.

## 9. Open Questions
1. Include per-asset HTTP status? (Potential — watch cardinality.)
2. Expose full domain instead of hash in trusted deployments? (Maybe config flag later.)
3. Should we compress event streams for external forwarding? (Future distributed phase.)

## 10. Status
Draft – refine after EventBus prototype (Iteration 2).
