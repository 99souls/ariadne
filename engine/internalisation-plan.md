# Internalisation Plan

A structured, low-risk sequence to reduce the public API surface of the `engine` module while preserving user value and enabling faster internal iteration.

> Status: DRAFT (Phase5f branch)
> Target window: Pre–`v0.2.0` (still <1.0 so breaking changes allowed, but we stage deprecations for trust)

---

## Objectives

1. Encapsulate implementation details (telemetry, ratelimit, resources, asset execution) behind the facade.
2. Present a **small, coherent, documentable public surface**.
3. Maintain a _graceful deprecation experience_ (transitional aliases + docs) even pre-1.0.
4. Unlock future deep refactors (pipeline concurrency, alternative metrics backends, storage backends) without public breakage.

### Key Principles

- "Stable by default" – only types the embedder must configure or consume remain exported.
- "Snapshots not internals" – expose aggregate, immutable views (health, limiter, resources) instead of live objects.
- Phased small PRs (≤ ~400 LOC diff) to keep review focused and bisectable.
- Each phase: (a) introduce new façade, (b) migrate internal call sites/tests, (c) add deprecation shims, (d) remove shims next phase.

---

## Current Public Surface (Baseline Inventory)

| Category    | Package / Symbols                                                                                                       | Notes                                                                        |
| ----------- | ----------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| Facade      | `engine.Engine`, `Config`, `Defaults`, lifecycle methods                                                                | Keep (core)                                                                  |
| Assets      | `AssetStrategy`, `DefaultAssetStrategy`, `AssetRef`, `AssetAction`, `MaterializedAsset`, `AssetEvent*`, `AssetMetrics*` | Consider slimming (Phase 4)                                                  |
| Telemetry   | `telemetry/events`, `telemetry/health`, `telemetry/metrics`, `telemetry/tracing`, `telemetry/policy`                    | Internalise progressively                                                    |
| Infra       | `ratelimit/*`                                                                                                           | Internalise (Phase 3)                                                        |
| Infra       | `resources/*`                                                                                                           | Internalise (Phase 3)                                                        |
| Adapters    | `adapters/telemetryhttp`                                                                                                | Wrap via facade factories                                                    |
| Data        | `models/*`                                                                                                              | Retain core data types; possibly move `RateLimitConfig` under `engine` later |
| Placeholder | `EngineStrategies`, `NewWithStrategies`                                                                                 | Remove or internalise early                                                  |

---

## Phase Overview

| Phase        | Goal                                 | Deliverables                                                 | Removal / Deprecation Introduced                         |
| ------------ | ------------------------------------ | ------------------------------------------------------------ | -------------------------------------------------------- |
| 0            | Baseline & guard                     | API snapshot, allowlist script                               | None                                                     |
| 1            | Health & Events wrap                 | `engine.Health`, `SubscribeEvents()`; internal bus/evaluator | Mark old `EventBus()` & health snapshot types deprecated |
| 2            | Telemetry policy & metrics narrowing | New `TelemetryPolicy` subset; `MetricsHandler()` facade      | Deprecate direct provider access, tracing concrete types |
| 3            | Rate limit & resources internal      | Move packages under `internal/`; expose snapshots only       | Deprecate imports `engine/ratelimit`, `engine/resources` |
| 4            | Asset surface pruning                | Hide strategy internals; keep metrics snapshot + config      | Deprecate `AssetStrategy` (unless plugin roadmap locked) |
| 5            | Cleanup & enforcement                | Remove shims; enforce allowlist CI                           | Remove deprecated symbols                                |
| 6 (optional) | Extension plugin story               | Stable plugin interfaces if demanded                         | N/A                                                      |

---

## Phase 0 – Baseline & Tooling

**Objectives**: Freeze a snapshot of current exported symbols and create an allowlist for future CI enforcement.

**Tasks**:

- Generate `API_REPORT.md` (go/packages + AST walker).

# Engine Internalization Decision Record

Status: ACTIVE – decisive contraction of public surface (no deprecation staging; breaking changes acceptable pre‑1.0).

Audience: Core maintainers. Goal: Minimise exported API to what embedders must use. Everything else becomes internal implementation detail or is removed.

---

## 1. Objective (Single Sentence)

Expose only a slim lifecycle + configuration + snapshot + extension point interface set; internalize or remove all implementation, orchestration, telemetry plumbing, advanced config, and business/monitoring layers.

## 2. Non‑Goals

- Backwards compatibility shims.
- Transitional stubs for removed packages (`resources`, `runtime`, etc.).
- External plugin story (can be introduced later on top of a smaller surface).

## 3. Target Public Surface (Post-Prune)

Top-level package `engine`:

- Types: `Engine`, `Config`, `EngineSnapshot`, `LimiterSnapshot` (trimmed), strategy interfaces: `Fetcher`, `Processor`, `OutputSink`, `AssetStrategy` (OPTIONAL – remove if no near-term plugin need).
- Functions: `New(Config) (*Engine, error)`, `SelectMetricsProvider(...)` (may remain if genuinely useful), `Version()` (if present).
- Methods: `Start`, `Stop`, `Snapshot`, `HealthSnapshot` (or `Health`), `Policy` (if still required), minimal telemetry policy update if retained.
- Errors: only canonical sentinel errors actually used by callers (re-evaluate).

Additional packages retained:

- `engine/models`: Pure data structures (no behavioral factories beyond constructors).
- `engine/config`: Slim `Config` only; remove unified / business / layered constructs.

Everything else: internal or removed.

## 4. Package Action Matrix

| Package / Path                                    | Action                                           | Rationale                                                                          | Notes                                                                            |
| ------------------------------------------------- | ------------------------------------------------ | ---------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `adapters/telemetryhttp`                          | Remove (moved logic to CLI)                      | HTTP exposure belongs outside core                                                 | Delete tests; CLI already hosts handlers.                                        |
| `resources/` (stub)                               | Delete                                           | Dead namespace; snapshot exposure is via Engine                                    | Remove allowlist guard tied to it.                                               |
| `monitoring/`                                     | Delete OR internalize whole file                 | Monolithic mixed concerns; superseded by CLI adapter + snapshots                   | Prefer delete; reintroduce as external module if ever needed.                    |
| `business/*`                                      | Internalize or delete                            | Historical layering; not part of minimal embed surface                             | If unused in tests, drop entirely.                                               |
| `strategies/` dir                                 | Delete (interfaces already in `strategies.go`)   | Redundant; reduces cognitive load                                                  | Adjust tests to import root.                                                     |
| `config/unified_config.go`                        | Remove                                           | Bloated experimental config; keep only lean `Config` struct                        | Inline only fields actually consumed by `New`.                                   |
| `config/runtime.go` (stub)                        | Delete                                           | Vestigial placeholder                                                              | Drop commentary; record decision here.                                           |
| `configx/`                                        | Internalize OR extract to `x/configlayers` later | Advanced layering & simulation not core                                            | Move to `internal/configlayers` for now.                                         |
| `crawler/` impl                                   | Internalize                                      | Implementation detail; expose `Fetcher` interface only                             | Provide default fetcher internally.                                              |
| `processor/` impl                                 | Internalize                                      | Same argument as crawler                                                           | Keep interface.                                                                  |
| `output/` concrete sinks                          | Internalize all but maybe `stdout` example       | Trim surface; encourage custom implementations via interface                       | Option: internalize `stdout` too and show doc snippet instead.                   |
| `ratelimit/`                                      | Internalize                                      | Allows algorithm redesign without API churn                                        | Export only snapshot struct from root.                                           |
| `telemetry/*` (events, tracing, policy internals) | Internalize                                      | Users shouldn't assemble telemetry primitives                                      | Keep only provider selection or even internalize that and drive via Config enum. |
| `engine/SelectMetricsProvider`                    | Keep or internalize                              | Keep only if real external extension; else move to internal and expose simple enum | Decision: KEEP (for now) – documented as provisional.                            |

## 5. Rationale Highlights (Critical Lens)

- Current breadth (monitoring, business, layered config) dilutes Engine’s mental model and increases accidental coupling risk.
- Implementation packages (crawler, processor, output, ratelimit) leak design choices that we may want to rework (parallelism model, retry semantics, queues) – keeping them public ossifies them prematurely.
- Unified / layered config encourages indirect configuration paths; a narrow `Config` struct keeps configuration explicit and reviewable.
- Telemetry handler exposure inside core would force HTTP and Prometheus dependencies on embedders who may not need them – adapter pattern validated by moving logic to CLI.
- Removing stubs eliminates “placeholder gravity” where future contributors might resurrect abandoned patterns.

## 6. Execution Order (Small, Focused Commits)

1. Remove: `adapters/telemetryhttp`, `resources/`, `strategies/` dir (redundant), delete `config/runtime.go` stub.
2. Internalize: `monitoring/` (or delete if zero references), `business/*`.
3. Slim config: Copy required fields from `UnifiedBusinessConfig` into `Config`; update `engine.New`; remove `unified_config.go` + tests relying on its internals.
4. Internalize implementations: move `crawler/`, `processor/`, `output/` (all concrete sinks) under `internal/`; adjust imports and tests.
5. Ratelimit move: `ratelimit/` → `internal/ratelimit`; re-export snapshot struct from root if still consumed.
6. Telemetry slimming: move events, tracing, policy, advanced metrics constructs internal; keep `SelectMetricsProvider` only (or replace with enum switch if simplified).
7. Internalize / relocate `configx/` → `internal/configlayers` (or delete if unused by `engine.New`).
8. Final pass: purge any now-unused snapshots / types; regenerate API report; tighten allowlist tests.

Each step: run engine + cli tests, regenerate API report, update CHANGELOG (Added: none / Removed: list), adjust allowlist guard.

## 7. Immediate Next Commit (Scope Definition)

Commit 1 ("prune: remove adapters/resources/strategies stubs") does:

- Delete `engine/adapters/telemetryhttp/`.
- Delete `engine/resources/`.
- Delete `engine/strategies/` directory (keep `strategies.go`).
- Delete `engine/config/runtime.go`.
- Update allowlist guard tests + API report.

## 8. Risk Assessment (Post-Decision)

| Risk                                           | Impact                    | Mitigation                                                              |
| ---------------------------------------------- | ------------------------- | ----------------------------------------------------------------------- |
| Hidden test reliance on removed packages       | Failing build             | Move instead of delete first; grep for imports before hard delete.      |
| Over-pruning removes needed extension point    | Slows future plugin story | Strategy interfaces retained (root) until explicit removal decision.    |
| Telemetry provider customization need emerges  | Requires re-export        | Document internal package structure to allow selective future exposure. |
| Config slimming misses a field used indirectly | Behavior regression       | Add focused test capturing all current `Config` usages before refactor. |

## 9. Success Criteria

- Exported symbol count reduced significantly (goal: >30% reduction vs current report).
- Public packages <= 3 (`engine`, `engine/models`, `engine/config`).
- No exported concrete implementations of internal behaviors (crawler, processor, sinks, ratelimit logic, monitoring aggregator).
- All telemetry wiring outside engine except provider selection.
- Tests green; API report diff matches intentional removal set.

## 10. Divergences From Previous Draft Plan

| Old Plan Element                     | New Decision                                                                   |
| ------------------------------------ | ------------------------------------------------------------------------------ |
| Staged deprecations                  | Immediate removal (pre‑1.0)                                                    |
| Health/Events new façade types first | Retain existing `HealthSnapshot` short-term; rename later only if value proven |
| Monitoring evolution                 | Drop entirely (not core)                                                       |
| Business metrics layering            | Removed; reintroduce externally if revived                                     |

## 11. Open Items (Explicitly Deferred)

- Whether to drop `AssetStrategy` interface (decide after internalization wave 3; investigate real external demand).
- Potential consolidation of snapshots into a single composite struct if it simplifies surface further.
- Converting `SelectMetricsProvider` to an unexported helper plus a simple config enum mapping.

---

## 12. Quick Reference – Action Checklist

```
[x] C1 Remove adapters/, resources/, strategies/ dir, runtime stub (branch: c1-prune-initial)
* [x] C2: Internalized monitoring/ and business/* packages (moved to internal/, updated imports, allowlists unaffected except runtime import path, changelog pending entry) – reduced public surface; legacy adapter in telemetry now references internal monitoring
* [x] C3: Remove `config/unified_config.go` (+ tests) – no external usage; shrink surface (branch: c3-slim-config-remove-unified)
* [x] C4 Internalize crawler/, processor/, output/ implementations (moved impl packages under internal/, deleted public impl tests, updated all import paths, regenerated API report; facade strategy interfaces only)
[] C5 Internalize ratelimit/
[] C6 Internalize telemetry internals (events, tracing, policy)
[] C7 Internalize configx/ (or delete)
[] C8 Final allowlist + API report shrink commit
```

---

## 13. Progress Summary (Post-C4) & Positioning

### Achievements So Far

- C1: Removed legacy adapter & scaffolding (telemetry HTTP handlers, resources stub, strategies/ dir, runtime stub) – immediate surface shrink.
- C2: Internalised monitoring & business implementation packages under `internal/`, eliminating broad implementation leakage while keeping tests green.
- C3: Deleted unified config layer – simplified configuration story; `engine/config` now intentionally empty (guarded) preventing re-expansion.
- C4: Internalised crawler, processor, output (all sinks & enhancement pipelines) implementations. Public surface now offers only interfaces (`Fetcher`, `Processor`, `OutputSink`, `AssetStrategy`) + facade types. API report updated; symbol count reduced (implementation packages disappeared).

### Current Public Surface Snapshot (Delta Focus)

Remaining larger implementation exposure: `engine/ratelimit` (interface + concrete adaptive limiter + internal state types). Telemetry packages also still public (events, metrics, tracing, policy) pending C6. Config layering (`configx/`) still public (C7). Asset subsystem exports intentionally retained pending decision (deprecation plan not yet executed).

### Readiness for C5 (Ratelimit Internalisation)

Coupling review:

- Engine facade depends on: `RateLimiter` interface, `LimiterSnapshot`, `AdaptiveRateLimiter` constructor (indirectly in `engine.New`).
- Internal pipeline depends on interface + `ErrCircuitOpen`, `Permit`, `Feedback` types.
- Tests rely on concrete `AdaptiveRateLimiter` (unit tests in `ratelimit/` package) and on `NewAdaptiveRateLimiter` inside integration tests.

Design intent post-internalisation: external users require only snapshot visibility & _optional_ configuration fields (already in `models.RateLimitConfig`). They do not require direct construction or subtype awareness of the adaptive limiter.

### Proposed Minimal Public Set After C5

Keep (possibly relocated to root `engine` package OR a tiny shim subpackage):

- Interface: `RateLimiter` (question: do we need to expose this? Only if embedders may plug a custom limiter soon. If not, we can hide and replace with unexported field; snapshot suffices.)
- Error: `ErrCircuitOpen` (exposed only if callers should distinguish circuit state during Start/stream operations; otherwise convert to generic transient classification internally).
- Snapshot: `LimiterSnapshot`, `DomainSummary` (read-only data needed for observability).
- Opaque handle types: `Permit`, `Feedback` (can be internal if callers never emit feedback; currently only internal pipeline calls feedback). => Make internal.

Decision recommendation: Expose ONLY `LimiterSnapshot` via `Engine.Snapshot()` and remove public `RateLimiter` + `AdaptiveRateLimiter` entirely (leanest future-proof path). If extension story needed later, reintroduce a narrow `rate` interface in a versioned path.

### Migration Mechanics

1. Move entire `engine/ratelimit` directory to `engine/internal/ratelimit`.
2. Introduce (in `engine/snapshot.go` or existing snapshot file) any type aliases required if we preserve exported `LimiterSnapshot` / `DomainSummary` (e.g., `type LimiterSnapshot = internalratelimit.LimiterSnapshot`). Alternatively redefine a trimmed struct filled by an adapter function.
3. Remove imports of `engine/ratelimit` from public files; replace construction in `engine.New` with internal factory `internal/ratelimit.NewAdaptiveLimiter(cfg)`.
4. Update pipeline & tests to new internal path.
5. Delete / migrate ratelimit unit tests under `internal/ratelimit` (preserve coverage). Public allowlist guard for ratelimit package becomes obsolete and is removed.
6. Run API report; ensure ratelimit package disappears and only chosen aliases remain (or none if we eliminate all types but snapshot via root embedding).

Fallback / rollback: Single directory move revertible; alias strategy allows phased removal if unexpected external need arises before release tag.

### Critical Positioning Assessment

Strengths:

- High test coverage around rate limiter logic (unit + integration) reduces regression risk during move.
- Prior moves validated import path update process & API report governance.

Weak Spots / Risks:

- Public removal of `RateLimiter` interface may break any hypothetical downstream mocks (low probability pre‑1.0, but note). Mitigation: keep interface temporarily in root if uncertain.
- Snapshot struct shape may still evolve; internalising now keeps agility but we should freeze field naming pre v0.2.0.
- Telemetry still broad; delaying its internalisation until after ratelimit is fine, but large surface remains temporarily (accept risk for sequencing simplicity).

Decision: Proceed with C5 adopting leanest approach (no public ratelimit package, possibly alias snapshot types) – improves future algorithm experimentation (different adaptation algorithms) with zero API churn.

---

## 14. Updated Risk Assessment (Incremental Delta for C5–C8)

| Risk                                                                            | Phase | Likelihood           | Impact      | Mitigation                                                                                   |
| ------------------------------------------------------------------------------- | ----- | -------------------- | ----------- | -------------------------------------------------------------------------------------------- |
| Hidden external reliance on `engine/ratelimit` concrete types                   | C5    | Low                  | Low         | Pre‑1.0; document CHANGELOG; provide clear migration note (no replacement needed).           |
| Losing ability to inject custom limiter post C5                                 | C5    | Medium (future need) | Medium      | Keep a private hook; if demand emerges add a stable `LimiterProvider` option later.          |
| Aliasing vs redefining snapshot leads to accidental re-export of internal types | C5    | Low                  | Low         | Prefer redefining minimal snapshot struct over alias if uncertain; verify via API report.    |
| Telemetry internalisation (events/tracing) touches many tests simultaneously    | C6    | Medium               | Medium/High | Stage: internalise packages one at a time (events → tracing → policy) with interim adapters. |
| Config layering removal breaks undocumented internal tests                      | C7    | Low                  | Low         | Grep usages first; migrate tests to new config pattern before delete.                        |
| Final allowlist tightening misses a straggler symbol                            | C8    | Low                  | Low         | Add temporary audit script diffing `go list -deps` exported sets vs API report.              |

---

## 15. C5 Execution Plan (Ready to Start)

Step-by-step (PR scope ≤ ~300 LOC net diff expected):

1. Branch: `c5-internalize-ratelimit`.
2. Move directory: `git mv engine/ratelimit engine/internal/ratelimit`.
3. Create new file `engine/ratelimit_facade.go` (if we retain snapshot structs) OR enhance existing engine snapshot file to embed limiter metrics.
4. Define (if retained):
   - `type LimiterSnapshot = internalratelimit.LimiterSnapshot` (or trimmed copy if we want to prune fields).
   - Optionally drop `DomainSummary` if not required externally (verify usage outside tests).
5. Update `engine/engine.go` constructor to import `engine/internal/ratelimit` instead of public path.
6. Update internal pipeline imports.
7. Remove obsolete allowlist guard test `engine/ratelimit/ratelimit_allowlist_guard_test.go` (will be moved with package; delete since package no longer public).
8. Move ratelimit unit tests alongside new internal path (they become `package ratelimit` inside internal); adjust any import paths referencing `engmodels` unaffected.
9. Run `go test ./engine/...` ensuring coverage preserved.
10. Regenerate API report; confirm removal of `ratelimit` package section (only snapshot alias remains if defined).
11. Update CHANGELOG (Removed: public ratelimit package; Changed: snapshot still available via Engine snapshot alias if applicable).
12. Update plan (mark C5 `[x]`).

Open decision before coding: alias vs redefine snapshot. Default: ALIAS (fast, zero struct copy) unless we intend to prune fields immediately.

Success Criteria (C5):

- API report no longer lists `Package \`ratelimit\``.
- All tests green; no public compile errors.
- Snapshot access unchanged for external caller code referencing `engine.Snapshot().Limiter`.
- Commit message: `prune(c5): internalize ratelimit implementation`.

---

## 16. Go / No-Go Gate for C5

All prerequisites satisfied (C1–C4 complete, plan updated, risks mapped). No blocking dependencies identified. Proceed with C5.

---

### Branch / PR Workflow

For each checklist item:

1. Create branch off `main` named `c<N>-<short-description>`.
2. Apply scoped changes (single concern; keep diff lean).
3. Run module tests: `go test ./engine/... ./cli/...`.
4. Regenerate API report (Makefile target) and include diff in PR description.
5. Update this plan (mark item `[~]` while in progress, `[x]` when merged).
6. Open PR with title: `prune(c<N>): <summary>`; label `api-surface`.
7. After merge, follow with a CHANGELOG update PR if not part of the same patch.

Reject combining unrelated internalizations in one PR—enables clean bisect and revert.

---

This document is authoritative until superseded; update it in the same PRs that materially change the sequence or scope.

- Provide engine-level method: `EnableAssets(policy AssetPolicy)` (or stays config-based).
- Translate asset events into summary counters only; events bus category stays (optional).

**Deprecations**:

- `DefaultAssetStrategy`, `AssetRef`, `AssetAction`, `MaterializedAsset`, `AssetEvent*`.

**Success Metrics**:

- No external code references asset types (grep). Smaller docs.

---

## Phase 5 – Shim Removal & Enforcement

**Actions**:

- Delete deprecated functions/types.
- Update README, remove deprecation notes.
- CI: enforce allowlist; failing build if deprecated symbols reintroduced.
- Update `API_REPORT.md` with final contracted surface; freeze for `v0.2.0` tag.

**Success Metrics**:

- Net exported symbol reduction ≥ 30% vs Phase 0 baseline.
- All integration tests green.
- Lint: no `// Deprecated:` markers remain (unless intentionally long-lived).

---

## Phase 6 (Optional) – Plugin Extension Story

If external customization for assets, rate limiters, or telemetry backends is required:

- Define versioned plugin interfaces (e.g., `engine/plugin/assets/v1` minimal surface).
- Provide adapter shim translating plugin to internal strategy.
- Maintain small stable contract & semantic version inside submodule.

---

## Cross-Phase Implementation Checklist

| Task                                                 | Phase | Done? |
| ---------------------------------------------------- | ----- | ----- |
| Add API report tooling                               | 0     |       |
| Introduce Health facade types                        | 1     |       |
| Deprecate EventBus() & HealthSnapshot()              | 1     |       |
| Migrate tests off direct health/events imports       | 1     |       |
| Add TelemetryPolicy slim struct                      | 2     |       |
| Add MetricsHandler()                                 | 2     |       |
| Deprecate MetricsProvider()                          | 2     |       |
| Internalize metrics/tracing packages                 | 2     |       |
| Internalize ratelimit & resources                    | 3     |       |
| Add limiter/resource snapshot trimming               | 3     |       |
| Asset internalization decision recorded              | 4     |       |
| Remove exported asset types or slim plugin interface | 4     |       |
| Remove deprecated aliases (events/health/metrics)    | 5     |       |
| Update README & stability policy                     | 5     |       |
| Final API report regenerate                          | 5     |       |
| Tag v0.2.0                                           | 5     |       |

---

## Risk & Mitigation Matrix

| Risk                                                | Impact                  | Likelihood      | Mitigation                                                |
| --------------------------------------------------- | ----------------------- | --------------- | --------------------------------------------------------- |
| Hidden downstream reliance on ratelimit/resources   | Break builds            | Low (mono-repo) | Search dependents; staged alias release                   |
| Test flakiness after moving telemetry internals     | Slows merge             | Medium          | Incremental PRs; keep old path until tests migrated       |
| Asset strategy future plugin need                   | Re-expose churn         | Medium          | Document decision; keep internal design loosely decoupled |
| Over-trimming health probes (loss of observability) | Operational blind spots | Low             | Keep additive probe fields; version docs                  |

---

## Success Metrics (Holistic)

- Exported symbol count reduction (target ≥ 30%).
- p95 API review time per PR < 10 minutes (smaller patches achieved).
- Zero breaking changes to core facade semantics (`New`, `Start`, `Stop`, `Snapshot`) across phases.
- No regressions in existing integration tests (green pipeline at each phase).
- Time-to-add-new-internal-metric (prototype to merged) reduced (qualitative dev feedback).

---

## Communication Plan

1. Add this document and link from `README.md` under a new section: _API Surface Evolution_.
2. Start a `DEPRECATION.md` listing symbols + scheduled removal phase.
3. PR template checkbox: "Touches public surface? Updated internalisation plan?".
4. Release notes each tagged pre-release summarizing removals & next-phase heads-up.

---

## Example Deprecation Comment Template

```go
// Deprecated: will be removed in Phase 3 (see internalisation-plan.md). Use engine.Snapshot().Limiter instead.
```

---

## Execution Ordering Heuristic

Always land **additions before removals**:

1. Add new facade types/methods.
2. Switch internal call sites/tests.
3. Mark old API deprecated (doc comment only).
4. Remove only after next minor pre-release (or next phase if pre-1.0 acceptable).

---

## Open Questions

| Question                                                  | Owner              | Deadline              |
| --------------------------------------------------------- | ------------------ | --------------------- |
| Do we promise plugin asset strategies in 2025?            | Maintainers        | Before Phase 4 start  |
| Keep per-domain limiter details externally visible?       | Observability lead | Phase 3 design review |
| Provide structured event export (JSON stream) externally? | TBD                | After Phase 2         |

---

## Next Immediate Action (Phase 0 → 1 Start)

1. Implement API report script & commit allowlist.
2. Introduce `engine/health.go` & `engine/events.go` (facade types).
3. Migrate a single test to the new health facade to validate mapping.
4. Submit PR: _Phase 1A: Add health & events facade (no removals yet)._

---

**End of Plan**
