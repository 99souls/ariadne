# Architecture Appendix: Asset Strategy (Phase 5D)

Status: Draft (Iteration 8 Documentation Pass)
Date: 2025-09-27
Related Plan: `phase5d-plan.md`

---
## 1. Purpose
The Asset Strategy elevates asset handling (discovery → decision → execution → rewrite) into a policy-driven, observable subsystem of the Engine. It replaces the legacy synchronous logic that previously intertwined with content transformation.

## 2. Design Overview
Pipeline stages:
1. Discover: Parse HTML, extract candidate references.
2. Decide: Apply `AssetPolicy` (allow/block, limits, inline heuristic) producing `AssetAction`s.
3. Execute: Concurrently fetch + optional optimize (bounded worker pool). Track metrics & events.
4. Rewrite: Deterministically replace original references with hashed stable paths.

Determinism is ensured by hashing content (SHA-256) and sorting assets by hash prior to rewrite.

## 3. Key Data Types
```
type AssetRef struct { URL, Type, Attr, Original string }
type AssetAction struct { Ref AssetRef; Mode AssetMode }

type MaterializedAsset struct {
  Ref AssetRef; Bytes []byte; Hash, Path string; Size int; Optimizations []string
}

type AssetPolicy struct {
  Enabled bool; MaxBytes int64; MaxPerPage int; InlineMaxBytes int64; Optimize bool;
  RewritePrefix string; AllowTypes, BlockTypes []string; MaxConcurrent int
}
```

## 4. Concurrency Model
Iteration 7 introduced a bounded worker pool for Execute:
- Worker count = `MaxConcurrent` (0 => min(NumCPU, 8)).
- Cumulative size guard via `MaxBytes` (cap enforced before each fetch).
- Determinism preserved: rewrite order independent of fetch order.
- Atomic counters + mutex-protected event ring ensure race safety (validated by `go test -race`).

## 5. Metrics & Events
Metrics (atomic counters): discovered, selected, skipped, downloaded, failed, inlined, optimized, bytesIn, bytesOut, rewriteFailures.
Events:
- `asset_download` (includes optimization metadata if applied)
- `asset_stage_error`
- `asset_rewrite`
(Deprecated: standalone `asset_optimize` event removed to reduce volume.)

## 6. Extended Discovery Coverage
Implemented selectors:
- `img[src]`, `img[srcset]` (first candidate)
- `source[srcset]` (picture/media)
- `video source[src]`, `audio source[src]`
- `link[rel=stylesheet][href]`
- `link[rel=preload][as=image|script|style]`
- `script[src]`
- Document anchors (`a[href]` ending in .pdf/.doc/.docx/.ppt/.pptx/.xls/.xlsx)

Deduplication via a `(type|url)` map prevents double counting (e.g., overlapping src + srcset).

## 7. Optimization Strategy (Current Minimal)
Whitespace collapse for CSS/JS; placeholder "img_meta" tag for images (future: real compression/transcoding). Optimization occurs inline in workers; results embedded in download events.

## 8. Failure Semantics
- Individual fetch failures increment `Failed` metric + stage error event; do not abort page.
- Bytes for failed assets are not counted.
- Rewrite operates only on successfully materialized assets.

## 9. Deterministic Hash Path
`/assets/<first2>/<fullhash><ext>` ensures sharded directories (256 buckets) and stable caching key. Extension inferred from URL path if possible.

## 10. Policy Validation
Basic validation: rewrite prefix must start with '/'. Additional validation (e.g. bounds checking) can be extended without breaking consumers.

## 11. Testing Inventory
- Instrumentation & determinism tests.
- Concurrency test validating stable rewrite under variable latency.
- Failure test (failed download metrics invariant).
- Extended discovery test (preload, source[srcset], docs).
- Benchmark baseline (Iteration 7) for Execute throughput.

## 12. Future Enhancements (Backlog)
- Multi-variant srcset materialization (currently first candidate only).
- Streaming / chunked fetch with progressive hashing.
- Pluggable optimization pipeline (image transcoding, compression levels, SVG sanitization).
- Exporter interface for metrics/event streaming (Phase 5E candidate).
- Conditional inlining by actual byte size (HEAD or partial GET) rather than heuristic.
- Content-type sniffing for extension-less assets.

## 13. Migration Notes
Legacy `internal/assets` removed; consumers enable the subsystem via `AssetPolicy.Enabled`. No compatibility shim provided. Hash-based rewrite paths may require downstream static hosting or bundling adjustments (documented in migration guide TBD).

## 14. Rationale Summary
The separation provides clear business authority, improves observability, and lays groundwork for distributed or deferred asset processing without entangling core content transformation code.

---
End of Appendix.
