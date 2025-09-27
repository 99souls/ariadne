# Phase 5D Plan: Asset Strategy Integration

Status: In Progress (Iterations 1–6 complete; Iteration 7 ACTIVE)
Date: September 27, 2025 (Updated: Iteration 7 concurrency + extended discovery scaffold started)
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
- Observability: Metrics + events available and validated via tests. (Partially Complete: basic counters + in-memory events implemented & tested.)
- Explicit Breaking Change: Legacy `internal/assets` subsystem removed. Consumers must adopt the new policy-driven strategy (major version shift expectation).
- Determinism: Same input corpus yields identical asset rewrite outputs across runs (hash naming + ordering tests) — Validated by integration test.

### 2.3 Backward Compatibility Stance (Directive)

Phase 5D introduces a deliberate, non-backward-compatible replacement of the legacy asset pipeline:

- The previous `internal/assets` package was removed in Iteration 5 (no soft deprecation window on this feature branch).
- Upgrade Path: Consumers enable `AssetPolicy.Enabled` and rely on deterministic hashed paths + rewrite semantics; no legacy shim will be provided.
- Rationale: Legacy implementation was incomplete, not directionally aligned, and created maintenance drag; early removal reduces migration cost over time.
- Communication: Mark release notes prominently as "Breaking: Asset subsystem replaced"; provide before/after config mapping in docs (to be added in Iteration 8 docs pass).

### 2.4 Exit Criteria Checklist (Progress)

- [x] Interface + default implementation merged (Iterations 1–4)
- [x] Config surface wired + validated (Iteration 1 + validation tests)
- [x] Processor/pipeline refactored to delegate asset path (Iteration 5 hook integration)
- [x] Tests (unit, integration, determinism) green (Deterministic rewrite + instrumentation test added)
- [x] Metrics & events instrumentation (baseline counters + events & tests)
- [ ] Performance sanity benchmark recorded (Planned Iteration 7)
- [ ] Race detector & concurrency validation (Planned Iteration 7)
- [ ] Documentation updated (API, operations, architecture progress) (Planned Iteration 8)
- [ ] Phase 5D completion note committed (Iteration 8)

---

## 3. Scope Decomposition & Workstreams

| Workstream            | Description                                            | Outputs / Status                                       |
| --------------------- | ------------------------------------------------------ | ------------------------------------------------------ |
| Interface Design      | Define `AssetStrategy` contract + data models          | Implemented (`asset_strategy.go`)                      |
| Extraction (Direct)   | Build discovery/decision/execute without legacy shim   | Implemented (legacy code removed; no adapter retained) |
| Config Integration    | Extend `engine.Config` with `AssetPolicy` + validation | Implemented & tested                                   |
| Pipeline Refactor     | Introduce hook & remove legacy pipeline                | Implemented (Iteration 5)                              |
| Observability         | Metrics counters + events types + tests                | Baseline implemented (Iteration 6); exporter TBD       |
| Determinism & Hashing | Stable naming + path scheme + tests                    | Implemented & validated                                |
| Concurrency & Perf    | Parallel Execute, worker pool, baseline benchmark      | Pending (Iteration 7)                                  |
| Extended Discovery    | srcset, media, docs (parity backlog)                   | Pending (Iteration 7+)                                 |
| Documentation         | API doc, operations guide, migration & progress log    | Pending (Iteration 8)                                  |
| Benchmark             | Before/after micro-benchmark script                    | Pending (Iteration 7)                                  |

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

Current (Iterations 1–6):

- Execute stage runs serially per page (simplifies determinism + early correctness).

Planned (Iteration 7):

- Introduce bounded worker pool (default size: min(4, GOMAXPROCS); configurable future `AssetPolicy.MaxConcurrent`).
- Maintain deterministic final rewrite ordering by sorting materialized assets by hash (already in place).
- Aggregate errors; non-fatal failures produce events and skip specific assets only.
- Metrics updated atomically for parallel downloads (may refactor counters to atomic operations if contention observed).

### 4.5 Metrics (Implemented Baseline / Additions)

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

Implementation Notes:

- Current counters stored in `AssetMetrics`; snapshot exposed via `Engine.AssetMetricsSnapshot()`.
- Events stored in bounded in-memory ring (cap 1024) via `Engine.AssetEvents()` — future exporter hook slated for Phase 5E or monitoring layer.
- Optimization currently reports an event per optimized asset (css/js whitespace collapse, svg meta tag placeholder).

### 4.6 Events (Implemented)

- `asset_stage_error` (stage, error)
- `asset_download` (url, bytes, duration)
- `asset_optimize` (url, saved_bytes)
- `asset_rewrite` (count)

### 4.7 Failure & Recovery Semantics

- Single asset failure does not abort page unless policy has `FailFast=true` (future extension; not in initial scope).
- Partial success: rewrite only successful assets; emit event for failures.

### 4.8 Backwards Compatibility Strategy (Superseded)

Original strategy (graceful coexistence) has been replaced by a hard cut-over:

- Legacy `internal/assets` package deleted in Iteration 5.
- No transitional adapter: existing users must enable `AssetPolicy` or accept no asset rewriting (if disabled).
- Major version release will highlight this; semantic version bump required.
- Future compatibility mitigation: Provide migration doc mapping old conceptual stages (discovery → download → optimize → rewrite) to new API calls & metrics.

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

| Iteration | Scope                                                   | Deliverables (Status)                               |
| --------- | ------------------------------------------------------- | --------------------------------------------------- |
| 1         | Interface + policy structs + docs stub                  | Strategy + config defaults (Complete)               |
| 2         | Discovery + basic execute                               | Serial discovery/execute tests (Complete)           |
| 3         | Policy decision matrix (allow/block, limits, inline)    | Decision + limit tests (Complete)                   |
| 4         | Optimization hook + hashing + deterministic path        | Path + optimization tests (Complete)                |
| 5         | Rewrite stage + processor delegation removal            | Hook integration, legacy removal (Complete)         |
| 6         | Metrics + events instrumentation                        | Counters + events + tests (Complete)                |
| 7         | Concurrency + extended discovery + performance baseline | Worker pool (in progress), srcset/media (partial), benchmark (baseline test stub) |
| 8         | Documentation + polish + release checklist              | Docs, migration guide, completion note (Pending)    |

---

## 7. Risk Register (Phase-Specific)

| Risk                                         | Likelihood | Impact | Mitigation                                                |
| -------------------------------------------- | ---------- | ------ | --------------------------------------------------------- |
| Hidden coupling in processor to asset fields | Medium     | High   | Early integration hook + full legacy removal (resolved)   |
| Performance regression from hashing / I/O    | Medium     | Medium | Benchmark in Iteration 7; introduce concurrency & pooling |
| Configuration overload                       | Low        | Medium | Minimal policy fields; defer advanced knobs               |
| Determinism flakiness (timestamp, ordering)  | Low        | Medium | Stable hash ordering & deterministic tests (validated)    |
| Event buffer growth / memory pressure        | Low        | Low    | Bounded event slice (cap 1024); future exporter offload   |
| Parallel metrics race (future concurrency)   | Medium     | Medium | Adopt atomic increments; add race tests in Iteration 7    |

---

## 8. Documentation Artifacts

- `phase5d-progress.md` (iteration log similar to 5C)
- Add section to: `config-api.md`, `config-operations-guide.md`, architecture analysis appendix update describing asset strategy shift
- Update README feature matrix

---

## 9. Completion Deliverables (Updated)

- Merged code (interfaces + default strategy + refactored pipeline) – Done
- Added + updated tests (unit, integration, determinism, instrumentation) – Done (concurrency pending)
- Extended discovery parity (srcset/media/doc assets) – Pending
- Performance benchmark delta documented (before/after table) – Pending
- Race detector clean report – Pending
- Updated docs (architecture, API, migration, ops, progress) – Pending
- Phase 5D completion note appended to progress log – Pending

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
