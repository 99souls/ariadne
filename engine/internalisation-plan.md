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
- Create `internal/api_allowlist.txt` (one symbol per line).
- Add a `make api-check` target failing if new unexpected exports appear.

**Success Metrics**:

- Report committed & green in CI.
- Subsequent phases show net _reduction_ in exported symbol count.

---

## Phase 1 – Health & Events Encapsulation

**Motivation**: Current `EventBus()` + `telemetryhealth.Snapshot` leak internal caching, categories, and per-subscriber buffering semantics.

**Public Additions**:

```go
// engine/health.go (new)
type Health struct {
    Overall string
    Generated time.Time
    Probes []HealthProbe // optional, stable subset
}

type HealthProbe struct { Name, Status, Detail string }

func (e *Engine) Health(ctx context.Context) Health

// Events
type Event struct { Time time.Time; Category, Type, Severity string; Fields map[string]any }
func (e *Engine) SubscribeEvents(buffer int) (<-chan Event, func(), error)
```

**Internal Moves**:

- Move `telemetry/health` ⇒ `internal/telemetry/health` (keep evaluator logic).
- Move `telemetry/events` ⇒ `internal/telemetry/events`.

**Deprecations**:

- `EventBus()` (comment: `// Deprecated: use SubscribeEvents`).
- `HealthSnapshot()` (keep temporarily, wrap new internal evaluator, mark deprecated).

**Tests**:

- Update existing tests to use new methods; keep one legacy test verifying deprecated path returns same semantics.

**Success Metrics**:

- No external imports of `telemetry/health` or `telemetry/events` in repository root tests (grep).
- Export diff: - (bus interfaces, evaluator types) + (small facade types) net negative symbol count.

---

## Phase 2 – Telemetry Policy & Metrics Simplification

**Motivation**: Avoid locking internal sampling knobs and provider interfaces.

**Public Changes**:

```go
type TelemetryPolicy struct {
    Health struct { ProbeTTL time.Duration; PipelineMinSamples int; DegradedRatio, UnhealthyRatio float64 }
    Tracing struct { SamplePercent float64 }
}
func (e *Engine) UpdateTelemetryPolicy(p *TelemetryPolicy)
```

(Internally map to richer struct.)

Add:

```go
func (e *Engine) MetricsHandler() http.Handler // returns noop if disabled
```

**Internalize** `telemetry/metrics` & `telemetry/tracing` packages; keep only opaque interfaces inside engine.

**Deprecations**:

- `MetricsProvider()` (comment: `// Deprecated: will be removed; use MetricsHandler`).
- Direct imports of `telemetry/policy` (add alias type for one release):
  ```go
  // Deprecated: use engine.TelemetryPolicy
  type TelemetryPolicy = policy.TelemetryPolicy
  ```

**Success Metrics**:

- Public symbol count reduced; no external code needs metrics interfaces directly.

---

## Phase 3 – Rate Limiter & Resource Manager Internalization

**Motivation**: Allow redesign (e.g., sharded vs adaptive algorithms, new persistence layer) without breaking consumers.

**Actions**:

- Move `ratelimit` → `internal/ratelimit`; `resources` → `internal/resources`.
- Introduce façade snapshots embedded in engine `Snapshot` (already largely present):
  ```go
  type LimiterSnapshot struct { TotalRequests, Throttled, Denied, OpenCircuits int64 }
  ```
  (Trim domain-level details or cap them.)
- Remove direct construction from user: only via `Config.RateLimit`/`Config.Resources`.

**Deprecations**:

- Type aliases for one release; then remove packages.

**Success Metrics**:

- Grep for `"/ratelimit"` & `"/resources"` yields zero outside engine after final removal.
- Snapshot stays backward compatible (additive fields only).

---

## Phase 4 – Asset Strategy Surface Pruning

**Decision Gate**: Do we foresee third-party custom asset strategies in the next 2 releases?

| Path                           | Action                                                                                                                                   |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| No external plugins short-term | Make strategy + related types internal. Keep `AssetPolicy` + `AssetMetricsSnapshot()` only.                                              |
| Yes (plugin story)             | Keep `AssetStrategy` but shrink: only interface + minimal discovery/execution structs; move implementation to internal; events internal. |

**Default Plan (assume internalization)**:

- Move all asset types except: `AssetPolicy`, `AssetMetricsSnapshot` to `internal/assets`.
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
