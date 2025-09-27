# Phase 5D Plan: Asset Strategy Integration

Status: Draft (Pending Approval)
Date: September 27, 2025
Related Analysis: See `phase5-engine-architecture-analysis.md` (Section Phase 5D)
Preceding Phase: 5C (Processor Migration & Config Platform Enhancements) — COMPLETE

---

## 1. Purpose & Strategic Context

Phase 5D operationalizes assets as a first-class, configurable business strategy within the Engine core. It extracts and decouples the current synchronous, tightly bound asset handling logic from `internal/assets` and the monolithic processor chain, turning it into an injectable and policy-driven subsystem. This extends the Engine’s business authority (per architecture analysis) and sets foundations for later multi-module separation.

Primary Architectural Drivers:

- Eliminate hard coupling between content transformation and asset I/O
- Establish policy-surface for download / skip / optimize / rewrite behaviors
- Provide deterministic, observable asset pipeline lifecycle and metrics
- Enable future async or distributed asset execution models (deferred to later phases)

Non-Goals (Explicit Deferrals):

- CDN / external cache integration
- Distributed asset processing workers
- Advanced image transcoding / media optimization heuristics
- Multi-module extraction (Phase 5G trigger conditions not yet evaluated)

---

## 2. Objectives & Success Criteria

### 2.1 Functional Objectives

1. Introduce `AssetStrategy` (or `AssetHandler`) interface with clear contract for discovery → selection → retrieval → optimization → rewrite.
2. Migrate existing asset logic from `internal/assets` into an Engine-owned default implementation without regressions.
3. Expose asset behavior through `engine.Config` (unified configuration surface) with validation and sensible defaults.
4. Integrate metrics & events (downloads attempted, succeeded, skipped, optimized, bytes in/out, rewrite failures).
5. Ensure processor pipeline no longer performs direct asset downloading or rewriting logic (inversion via strategy).
6. Provide deterministic rewrite semantics (stable URL mapping / hash naming) to support future caching.
7. Maintain or improve current performance baseline (<5% increase in end-to-end crawl time on existing test corpus).

### 2.2 Non-Functional Success Criteria

- Test Coverage: ≥ 85% for new asset subsystem (unit + integration) / overall engine package coverage not reduced below prior baseline.
- Concurrency Safety: Race detector clean across modified packages.
- Observability: Metrics + events available and validated via tests.
- Backwards Compatibility: No changes required in CLI or existing engine public APIs beyond additive config fields.
- Determinism: Same input corpus yields identical asset rewrite outputs across runs (hash naming + ordering tests).

### 2.3 Exit Criteria Checklist

- [ ] Interface + default implementation merged
- [ ] Config surface wired + validated
- [ ] Processor refactored to delegate asset path
- [ ] Tests (unit, integration, determinism) green
- [ ] Metrics & events instrumentation verified
- [ ] Performance sanity benchmark recorded
- [ ] Documentation updated (API, operations, architecture progress)
- [ ] Phase 5D completion note committed

---

## 3. Scope Decomposition & Workstreams

| Workstream            | Description                                                   | Outputs                                       |
| --------------------- | ------------------------------------------------------------- | --------------------------------------------- |
| Interface Design      | Define `AssetStrategy` contract + data models                 | `asset_strategy.go`, docs section             |
| Extraction & Shim     | Move logic from `internal/assets`; add shim adapter layer     | `default_strategy.go`, transitional adapter   |
| Config Integration    | Extend `engine.Config` with `AssetPolicy` struct + validation | `config.go` changes, `validation` tests       |
| Processor Refactor    | Remove embedded asset logic; inject strategy                  | Modified `processor` package + tests          |
| Observability         | Metrics counters + events types                               | `metrics.go` additions, `events.go` additions |
| Determinism & Hashing | Stable naming + mapping tables                                | Hashing helper, determinism tests             |
| Testing Matrix        | Unit, integration, concurrency, rollback scenarios            | New `_test.go` files                          |
| Documentation         | API doc, operations guide, progress log updates               | `phase5d-progress.md`, API & ops diffs        |
| Benchmark             | Baseline before/after micro-benchmark script                  | `asset_benchmark_test.go`                     |

---

## 4. Detailed Design Elements

### 4.1 Interface Contract (Initial Draft)

```go
// AssetStrategy defines the pluggable asset handling pipeline.
type AssetStrategy interface {
    // Discover parses processed (or raw) content and returns candidate asset references.
    Discover(ctx context.Context, page *models.Page) ([]AssetRef, error)
    // Decide evaluates policy and returns selected subset + actions (download, skip, inline, rewrite-only).
    Decide(ctx context.Context, refs []AssetRef, policy AssetPolicy) ([]AssetAction, error)
    // Execute performs retrieval + optional optimization, returning materialized assets.
    Execute(ctx context.Context, actions []AssetAction) ([]MaterializedAsset, error)
    // Rewrite mutates page content (or returns a transformed copy) with updated asset links.
    Rewrite(ctx context.Context, page *models.Page, assets []MaterializedAsset) (*models.Page, error)
}
```

### 4.2 Data Structures (Draft)

```go
type AssetRef struct { URL string; Type string; Attr string; Original string }

type AssetAction struct { Ref AssetRef; Mode AssetMode }

type AssetMode int
const (
    AssetModeDownload AssetMode = iota
    AssetModeSkip
    AssetModeInline  // small SVG etc
    AssetModeRewrite // no fetch, path mapping only
)

type MaterializedAsset struct {
    Ref    AssetRef
    Bytes  []byte
    Hash   string // sha256 hex
    Path   string // stable relative output path
    Size   int
    Optimizations []string
}

// Policy available via engine.Config
type AssetPolicy struct {
    Enabled          bool
    MaxBytes         int64
    MaxPerPage       int
    InlineMaxBytes   int64
    Optimize         bool
    RewritePrefix    string // e.g. /assets/
    AllowTypes       []string
    BlockTypes       []string
}
```

### 4.3 Deterministic Naming

`Path` = `<rewritePrefix><first2>/<hash>.<ext>` to avoid directory bloat, support future CDN mapping.

### 4.4 Concurrency Model

- Page-level handling executed per processed page, but per-page strategy ensures internal ordering.
- Asset downloads performed with bounded worker pool (policy-driven: default = min(4, GOMAXPROCS)).
- Avoid global shared mutable state beyond metrics counters.

### 4.5 Metrics (Additions)

| Metric                       | Type    | Description                   |
| ---------------------------- | ------- | ----------------------------- |
| assets_discovered_total      | counter | Raw candidates found          |
| assets_selected_total        | counter | Selected after decision stage |
| assets_downloaded_total      | counter | Successfully retrieved        |
| assets_skipped_total         | counter | Skipped by policy             |
| assets_inlined_total         | counter | Inlined (below threshold)     |
| assets_optimized_total       | counter | Optimization performed        |
| asset_bytes_in_total         | counter | Bytes downloaded              |
| asset_bytes_out_total        | counter | Bytes after optimization      |
| asset_rewrite_failures_total | counter | Rewrite errors                |

### 4.6 Events (Additions)

- `asset_stage_error` (stage, error)
- `asset_download` (url, bytes, duration)
- `asset_optimize` (url, saved_bytes)
- `asset_rewrite` (count)

### 4.7 Failure & Recovery Semantics

- Single asset failure does not abort page unless policy has `FailFast=true` (future extension; not in initial scope).
- Partial success: rewrite only successful assets; emit event for failures.

### 4.8 Backwards Compatibility Strategy

- Processor will invoke strategy if `policy.Enabled` else skip, preserving legacy behavior (no asset transformation).
- Legacy fields in internal processor deprecated with TODO markers -> removed in Phase 5E or earlier cleanup.

---

## 5. Testing Strategy

| Test Category    | Focus                                                                 |
| ---------------- | --------------------------------------------------------------------- |
| Unit             | Policy decision edge cases, hash stability, path generation           |
| Integration      | Full page crawl with assets (HTML fixture referencing css/js/img/svg) |
| Determinism      | Repeated run diff of rewritten HTML + asset paths                     |
| Concurrency      | Race detector on parallel page processing                             |
| Limits           | MaxBytes, MaxPerPage enforcement                                      |
| Inline           | Inline threshold behavior                                             |
| Optimization     | When Optimize=true; ensure marker recorded                            |
| Metrics & Events | Emission counts, failure scenarios                                    |

Synthetic fixtures placed under `packages/engine/testdata/assets/`.

---

## 6. Iteration Plan (Agile Breakdown)

| Iteration | Scope                                                      | Deliverables                                                |
| --------- | ---------------------------------------------------------- | ----------------------------------------------------------- |
| 1         | Interface + policy structs + docs stub                     | `asset_strategy.go`, config additions, skeletal tests       |
| 2         | Extraction of discovery + basic download (no optimization) | Default strategy (discover, decide=all download) tests pass |
| 3         | Policy decision matrix (allow/block, limits, inline)       | Decision tests, updated integration test                    |
| 4         | Optimization hook + hashing + deterministic path           | Hashing helper, path tests, optimization stub               |
| 5         | Rewrite stage + processor delegation removal               | Processor refactor, determinism tests pass                  |
| 6         | Metrics + events instrumentation                           | Observability tests                                         |
| 7         | Edge cases + concurrency + performance baseline            | Bench + race detector clean                                 |
| 8         | Documentation + polish + deprecation markers               | Updated docs, progress log, completion checklist            |

---

## 7. Risk Register (Phase-Specific)

| Risk                                         | Likelihood | Impact | Mitigation                                              |
| -------------------------------------------- | ---------- | ------ | ------------------------------------------------------- |
| Hidden coupling in processor to asset fields | Medium     | High   | Incremental refactor w/ adapter shim first              |
| Performance regression from hashing / I/O    | Medium     | Medium | Batch hashing, streaming download, benchmark early      |
| Configuration overload                       | Low        | Medium | Provide defaults; only expose critical fields initially |
| Determinism flakiness (timestamp, ordering)  | Low        | Medium | Avoid time-based names, stable sort inputs              |

---

## 8. Documentation Artifacts

- `phase5d-progress.md` (iteration log similar to 5C)
- Add section to: `config-api.md`, `config-operations-guide.md`, architecture analysis appendix update describing asset strategy shift
- Update README feature matrix

---

## 9. Completion Deliverables

- Merged code (interfaces + default strategy + refactored processor)
- Added + updated tests (unit, integration, determinism, concurrency)
- Updated docs (architecture, API, ops, progress)
- Performance benchmark delta documented (before/after table)
- Phase 5D completion note appended to progress log

---

## 10. Post-Phase Evaluation Inputs

Inputs for Phase 5E readiness:

- Asset subsystem stable (interface churn <2 changes over last 2 iterations)
- No open P0 bugs in asset rewrite
- Bench: ≤5% slowdown vs pre-Phase 5D baseline or justified with optimization backlog item

---

## 11. Open Questions (Tracked)

1. Should optimization be synchronous or future offloaded? (Defer: future phase / feature flag)
2. Is inline threshold dynamic (e.g., percentile-based)? (Defer)
3. Do we need content-type sniffing beyond extension? (Add follow-up ticket if false positives observed)

---

## 12. Approval

(Sign-off to be recorded upon checklist completion.)
