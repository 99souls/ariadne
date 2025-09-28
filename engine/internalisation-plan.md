# Internalisation Plan

A structured, low-risk sequence to reduce the public API surface of the `engine` module while preserving user value and enabling faster internal iteration.

> Status: ACTIVE (hard-cut mode – no deprecation shims pre-1.0)
> Target window: Pre–`v0.2.0` (pre-1.0 so breaking changes allowed; we prefer direct removal over staged deprecation to reduce maintenance drag)

---

## Objectives

1. Encapsulate implementation details (telemetry, ratelimit, resources, asset execution) behind the facade.
2. Present a **small, coherent, documentable public surface**.
3. Prefer immediate hard cuts (direct removal) pre-1.0; CHANGELOG entry is sufficient notice.
4. Unlock future deep refactors (pipeline concurrency, alternative metrics backends, storage backends) without public breakage.

### Key Principles

- "Stable by default" – only types the embedder must configure or consume remain exported.
- "Snapshots not internals" – expose aggregate, immutable views (health, limiter, resources) instead of live objects.
- Phased small PRs (≤ ~400 LOC diff) to keep review focused and bisectable.
- Each phase: introduce/adjust façade (if needed), migrate internal call sites/tests, and remove old symbols in the same change (no deprecation shims).

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

| Phase        | Goal                                 | Deliverables                                                 | Removal (Hard Cut)                                   |
| ------------ | ------------------------------------ | ------------------------------------------------------------ | ---------------------------------------------------- |
| 0            | Baseline & guard                     | API snapshot, allowlist script                               | None                                                 |
| 1            | Health & Events wrap                 | `engine.Health`, `SubscribeEvents()`; internal bus/evaluator | Remove legacy `EventBus()` access when facade ready  |
| 2            | Telemetry policy & metrics narrowing | New `TelemetryPolicy` subset; `MetricsHandler()` facade      | Remove direct provider/tracing concrete exports      |
| 3            | Rate limit & resources internal      | Move packages under `internal/`; expose snapshots only       | Remove public impl packages                          |
| 4            | Asset surface pruning                | Hide strategy internals; keep metrics snapshot + config      | Remove unneeded strategy concretes (maybe interface) |
| 5            | Cleanup & enforcement                | Allowlist + CI enforcement                                   | Remove any residual legacy symbols                   |
| 6 (optional) | Extension plugin story               | Stable plugin interfaces if demanded                         | N/A                                                  |

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
- Functions: `New(Config) (*Engine, error)`, (optional) `Version()` if retained.
  _NOTE_: A previously contemplated public `SelectMetricsProvider` helper was **not** exported. Backend selection is now an internal detail (`selectMetricsProvider`) since C9; configuration is driven solely by `Config{ MetricsEnabled, MetricsBackend }`.
- Methods: `Start`, `Stop`, `Snapshot`, `HealthSnapshot` (or `Health`), `Policy` (if still required), minimal telemetry policy update if retained.
- Errors: only canonical sentinel errors actually used by callers (re-evaluate).

Additional packages retained:

[*] C6 (step 2b) Internalize telemetry policy package and finalize facade span helper decision (policy moved, span helper still deferred)

- `engine/models`: Pure data structures (no behavioral factories beyond constructors).
- `engine/config`: Slim `Config` only; remove unified / business / layered constructs.

Everything else: internal or removed.

## 4. Package Action Matrix

| Package / Path                                    | Action                                           | Rationale                                                        | Notes                                                                            |
| ------------------------------------------------- | ------------------------------------------------ | ---------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `adapters/telemetryhttp`                          | Remove (moved logic to CLI)                      | HTTP exposure belongs outside core                               | Delete tests; CLI already hosts handlers.                                        |
| `resources/` (stub)                               | Delete                                           | Dead namespace; snapshot exposure is via Engine                  | Remove allowlist guard tied to it.                                               |
| `monitoring/`                                     | Delete OR internalize whole file                 | Monolithic mixed concerns; superseded by CLI adapter + snapshots | Prefer delete; reintroduce as external module if ever needed.                    |
| `business/*`                                      | Internalize or delete                            | Historical layering; not part of minimal embed surface           | If unused in tests, drop entirely.                                               |
| `strategies/` dir                                 | Delete (interfaces already in `strategies.go`)   | Redundant; reduces cognitive load                                | Adjust tests to import root.                                                     |
| `config/unified_config.go`                        | Remove                                           | Bloated experimental config; keep only lean `Config` struct      | Inline only fields actually consumed by `New`.                                   |
| `config/runtime.go` (stub)                        | Delete                                           | Vestigial placeholder                                            | Drop commentary; record decision here.                                           |
| `configx/`                                        | Internalize OR extract to `x/configlayers` later | Advanced layering & simulation not core                          | Move to `internal/configlayers` for now.                                         |
| `crawler/` impl                                   | Internalize                                      | Implementation detail; expose `Fetcher` interface only           | Provide default fetcher internally.                                              |
| `processor/` impl                                 | Internalize                                      | Same argument as crawler                                         | Keep interface.                                                                  |
| `output/` concrete sinks                          | Internalize all but maybe `stdout` example       | Trim surface; encourage custom implementations via interface     | Option: internalize `stdout` too and show doc snippet instead.                   |
| `ratelimit/`                                      | Internalize                                      | Allows algorithm redesign without API churn                      | Export only snapshot struct from root.                                           |
| `telemetry/*` (events, tracing, policy internals) | Internalize                                      | Users shouldn't assemble telemetry primitives                    | Keep only provider selection or even internalize that and drive via Config enum. |
| `engine/SelectMetricsProvider`                    | (Dropped – internal helper only)                 | Avoid premature extension point; config flags sufficient         | Decision: INTERNALIZED in C9; no exported selection function remains.            |

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
6. Telemetry slimming: move events, tracing, policy, advanced metrics constructs internal; internalize metrics provider selection logic (no exported selection function).
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
- (Completed) Converted contemplated `SelectMetricsProvider` to an unexported helper (`selectMetricsProvider`) – config flags now the only public knobs.

---

## 12. Quick Reference – Action Checklist

```
[x] C1 Remove adapters/, resources/, strategies/ dir, runtime stub (branch: c1-prune-initial)
* [x] C2: Internalized monitoring/ and business/* packages (moved to internal/, updated imports, allowlists unaffected except runtime import path, changelog pending entry) – reduced public surface; legacy adapter in telemetry now references internal monitoring
* [x] C3: Remove `config/unified_config.go` (+ tests) – no external usage; shrink surface (branch: c3-slim-config-remove-unified)
* [x] C4 Internalize crawler/, processor/, output/ implementations (moved impl packages under internal/, deleted public impl tests, updated all import paths, regenerated API report; facade strategy interfaces only)
[x] C5 Internalize ratelimit/ (implementation moved under `engine/internal/ratelimit`; legacy `engine/ratelimit` package stub REMOVED – physical deletion complete). Limiter snapshot now always non-nil: when limiter disabled an empty `LimiterSnapshot` struct is returned to simplify callers.
[*] C6 (step 1) Add telemetry facade types (TelemetryEvent, TelemetryOptions, RegisterEventObserver) and conditional initialization (metrics/events/tracing/health) – DONE on branch c6-internalize-telemetry (commit 190e4c9). Health change events bridged to observers.
[x] C6 (step 2a) Internalize telemetry tracing package; events pending (migrated usage but package still public at that point)
[x] C6 (step 2b) Internalize telemetry policy package and finalize facade span helper decision (policy package removed)
[x] C7 Delete configx/ subsystem (decided against internalization; rationale in md/configx-internalization-analysis.md)
[x] C8 Initial telemetry contraction (engine now uses internal metrics + internal events/tracing; public telemetry/metrics & telemetry/events packages still present for removal) – groundwork laid (MetricsHandler facade in place, provider selection internalized).
[x] C9 Final telemetry hard cut & governance alignment COMPLETE (public telemetry/metrics & telemetry/events removed; API report regenerated; allowlists updated; facade tests added for health change + MetricsHandler availability; docs/changelog consolidation pending minor polish but structural objectives satisfied)
```

---

## 13. Progress Summary (Post-C4) & Positioning

### Achievements So Far

- C1: Removed legacy adapter & scaffolding (telemetry HTTP handlers, resources stub, strategies/ dir, runtime stub) – immediate surface shrink.
- C2: Internalised monitoring & business implementation packages under `internal/`, eliminating broad implementation leakage while keeping tests green.
- C3: Deleted unified config layer – simplified configuration story; `engine/config` now intentionally empty (guarded) preventing re-expansion.
- C4: Internalised crawler, processor, output (all sinks & enhancement pipelines) implementations. Public surface now offers only interfaces (`Fetcher`, `Processor`, `OutputSink`, `AssetStrategy`) + facade types. API report updated; symbol count reduced (implementation packages disappeared).

### Current Public Surface Snapshot (Delta Focus)

Remaining larger implementation exposure (post C7): telemetry packages (events, metrics, tracing) still public (final narrowing in C8). Asset subsystem exports intentionally retained pending decision (hard-cut approach will remove directly if pruned). New facade types now present so we can safely finalize telemetry internalization.

### C5 (Ratelimit Internalisation) – Completed

Coupling review:

- Engine facade depends on: `RateLimiter` interface, `LimiterSnapshot`, `AdaptiveRateLimiter` constructor (indirectly in `engine.New`).
- Internal pipeline depends on interface + `ErrCircuitOpen`, `Permit`, `Feedback` types.
- Tests rely on concrete `AdaptiveRateLimiter` (unit tests in `ratelimit/` package) and on `NewAdaptiveRateLimiter` inside integration tests.

Design intent realised: external users require only snapshot visibility & optional configuration fields (already in `models.RateLimitConfig`). Direct construction & subtype awareness removed. Snapshot invariant strengthened (always non-nil) eliminating nil-guard boilerplate in consumers.

### Minimal Public Set After C5 (Result)

Outcome: Only `LimiterSnapshot` (plain data) remains externally reachable via `Engine.Snapshot()`. All construction, interfaces, circuit-breaker errors, and permit/feedback mechanics are internal. Future extension (custom limiter) would introduce a fresh narrow interface if justified.

### Migration Mechanics (Executed)

1. Moved `engine/ratelimit` → `engine/internal/ratelimit`.
2. Updated `engine.New`, pipeline, and tests to internal path.
3. Removed public interfaces/errors/permit+feedback types (hard cut, no aliases).
4. Snapshot struct surfaced only via `Engine.Snapshot()`; pointer always populated (empty when disabled).
5. Added tests asserting non-nil limiter snapshot invariant.
6. CHANGELOG updated (C5 entry + invariant note). Legacy stub folder subsequently deleted (hard cut finalized).

### Critical Positioning Assessment

Strengths:

- High test coverage around rate limiter logic (unit + integration) reduces regression risk during move.
- Prior moves validated import path update process & API report governance.

Weak Spots / Risks:

- Public removal of `RateLimiter` interface may break any hypothetical downstream mocks (low probability pre‑1.0, but note). Mitigation: keep interface temporarily in root if uncertain.
- Snapshot struct shape may still evolve; internalising now keeps agility but we should freeze field naming pre v0.2.0.
- Telemetry still broad; delaying its internalisation until after ratelimit is fine, but large surface remains temporarily (accept risk for sequencing simplicity).

Decision: Proceed with C5 adopting leanest approach (no public ratelimit package, possibly alias snapshot types) – improves future algorithm experimentation (different adaptation algorithms) with zero API churn.

### C6 (Telemetry Internalisation) – In Progress

Step 1 (DONE): Introduced additive facade layer (`TelemetryOptions`, `TelemetryEvent`, `EventObserver`, `Engine.RegisterEventObserver`) plus conditional subsystem initialization. Health evaluator change events now dispatched to registered observers, establishing the bridging mechanism needed to remove public bus exposure.

Upcoming (Step 2):

1. Move `engine/telemetry/events`, `.../tracing`, `.../policy` (and potentially advanced metrics selection logic if narrowed) under `engine/internal/telemetry/`.
2. Replace public `EventBus()` and `Tracer()` with either removal (preferred) or thin helpers (`StartSpan(ctx, name, labels...)`) if external span creation remains needed.
3. Expose only high-level policy mutation (`UpdateTelemetryPolicy`) and observer registration.
4. Regenerate API report; update allowlist guard removing bus/tracer symbols.
5. Update CHANGELOG with removal list and facade addition note.
6. Add tests asserting observer receives health change events (already implicit) plus new synthetic event injection test (may publish internal test-only event through bus to ensure bridging holds after internalization).

Risks / Notes:

- Ensure metrics provider selection remains stable; selection logic remains internal (no exported helper) – document config-only workflow.
- Tracer removal must not break any existing test relying on span creation; search tests for `Tracer()` usage before removal.
- Event bus direct usage in tests must be migrated to observer assertions prior to deletion.

---

## C9 – Final Telemetry Hard Cut & Governance Alignment (COMPLETED)

Purpose: (Completed) Removed the now-orphaned public `telemetry/metrics` and `telemetry/events` packages, aligning governance artifacts (API report, allowlists, docs, changelog) to the true facade: `Engine.{MetricsHandler, RegisterEventObserver, UpdateTelemetryPolicy}` plus `telemetry/health` & `telemetry/logging`.

### Scope (Executed)

1. Deleted directories: `engine/telemetry/metrics/`, `engine/telemetry/events/`.
2. Removed references in CLI & tests; adjusted imports to facade-only APIs.
3. Regenerated `API_REPORT.md` – legacy provider/event bus symbols absent; facade methods present.
4. Updated allowlist guard tests to fail if `metrics`, `events`, or `tracing` public subpackages reappear.
5. Added facade tests: health change event observer + `TestMetricsHandlerAvailability` (enabled/disabled + backend matrix).
6. Fixed latent Prometheus gauge nil pointer (post-internalization) ensuring handler safe.
7. Simplified pre-commit hooks to reduce friction (removed whitespace / go mod tidy drift hooks).
8. CHANGELOG updated with breaking entries (final consolidation pass still pending to merge multi-line telemetry section into a single authoritative BREAKING block).
9. Plan updated marking C9 complete; transition to Phase 5 enforcement tasks next.

### Exit Criteria

| Criterion                                        | Validation                                                                                                                |
| ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------- |
| Public metrics & events packages no longer exist | `grep -R "telemetry/metrics" engine/telemetry` returns only health/logging; commands exit 1 for removed dirs              |
| API report signature changed                     | New signature hash differs; removed symbols absent                                                                        |
| Engine exposes only intended telemetry facade    | API report lists `Engine.MetricsHandler` & observer methods; no legacy provider/event bus constructors                    |
| Tests green                                      | `go test ./...` passes                                                                                                    |
| Docs free of legacy references                   | Grep shows zero references to removed constructor names (`NewPrometheusProvider`, etc.) outside changelog historical note |
| Allowlist guard enforces absence                 | Deliberate re-add triggers failure locally                                                                                |

### Risks & Mitigations

| Risk                                                                     | Impact                              | Likelihood | Mitigation                                                                       |
| ------------------------------------------------------------------------ | ----------------------------------- | ---------- | -------------------------------------------------------------------------------- |
| Accidental lingering benchmark/import keeps metrics pkg from deletion    | Incomplete removal causes confusion | Low        | Pre-delete grep + remove tests first                                             |
| CLI silently breaks metrics endpoint if handler becomes nil unexpectedly | Operational observability loss      | Low        | Add explicit test for handler presence when enabled                              |
| Missed CHANGELOG consolidation leaves conflicting guidance               | User confusion                      | Medium     | Single authoritative BREAKING section rewrite in C9 commit                       |
| Asset refactoring later needs event bus semantics                        | Re-exposing churn                   | Low        | Observer API suffices; can add structured asset events later under stable facade |

### Implementation Order (Actual)

1. Removed references & deleted packages (multiple passes due to transient stub reappearance).
2. Tightened allowlist guard (hard fail on reintroduction of removed packages).
3. Regenerated API report; resolved pre-commit friction by pruning noisy hooks.
4. Added observer & metrics handler tests; fixed Prometheus gauge nil bug.
5. Updated CHANGELOG & docs (pending final consolidation step).
6. Committed final C9 changes (commit 6dc6f0a + follow-up test commit 2a390da).

### Commit Message Template

```
prune(c9): remove public telemetry metrics/events packages; finalize telemetry facade & governance

BREAKING: Removed engine/telemetry/metrics and engine/telemetry/events. Configure via engine.Config { MetricsEnabled, MetricsBackend } and expose Prometheus with Engine.MetricsHandler(); register observers with Engine.RegisterEventObserver().
```

### Post-C9 Next Step

Enter Phase 5 (enforcement): freeze surface, add CI job verifying API report hash & deny reintroduction of removed directories.

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

## 15. (Archived) Original C5 Execution Plan

Historical reference only – superseded by executed hard cut. Retained to document rationale and sequencing; no further action required.

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

| Task                                                                     | Phase | Done? |
| ------------------------------------------------------------------------ | ----- | ----- |
| Add API report tooling                                                   | 0     |       |
| Introduce Health facade types                                            | 1     |       |
| Deprecate EventBus() & HealthSnapshot()                                  | 1     |       |
| Migrate tests off direct health/events imports                           | 1     |       |
| Add TelemetryPolicy slim struct                                          | 2     |       |
| Add MetricsHandler()                                                     | 2     | [x]   |
| Deprecate MetricsProvider() (removed; replaced by MetricsHandler facade) | 2     | [x]   |
| Internalize metrics/tracing packages (initial facade + internal copies)  | 2     | [x]   |
| Internalize ratelimit & resources                                        | 3     |       |
| Add limiter/resource snapshot trimming                                   | 3     |       |
| Asset internalization decision recorded                                  | 4     |       |
| Remove exported asset types or slim plugin interface                     | 4     |       |
| Remove deprecated aliases (events/health/metrics)                        | 5     |       |
| Update README & stability policy                                         | 5     |       |
| Final API report regenerate (post C9)                                    | 5     |       |
| Tag v0.2.0                                                               | 5     |       |

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
