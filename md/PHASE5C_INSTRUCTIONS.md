# Phase 5C Configuration & Policy Management - Implementation Instructions

**DOCUMENT STATUS**: Authoritative Plan (READ THIS FIRST)  
**Audience**: Automated & Human Contributors (Agents MUST follow processes exactly)  
**Current Engine Status**: ✅ Phase 5B COMPLETE (Business Logic Centralized)  
**Target of This Phase**: Introduce hierarchical, versioned, dynamically reloadable configuration & policy management with safety, auditability, and impact visibility.  
**Branch**: `engine-migration` (continue) or `phase5c-config` (recommended new feature branch)  
**Planned Duration**: ~2 weeks (4–6 focused sessions)  
**Blocking Dependencies**: None (all prerequisites satisfied)  
**Exit Criteria**: All Success Criteria (Section 2) met, zero lint issues, all tests passing.  

> Revision History:
> - v1 (initial): Included prospective CLI integration drafts.
> - v2 (current, deferred CLI): Removed in-phase CLI implementation; CLI moved to future Phase 7 per updated roadmap alignment (see Rationale Section 0.2).

## 0. Rationale Alignment & Adjustments

### 0.1 Alignment With Core Roadmap (plan.md)
The central roadmap sequences user-facing CLI experience after production hardening. Phase 5C focuses on internal platform capabilities (hierarchical config, versioning, audit, rollout). Exposing these through a CLI now would:  
1. Encourage premature external coupling before internal API stabilizes.  
2. Increase surface area for testing & documentation while core semantics are still evolving.  
3. Risk rework when engine facade and package boundaries finalize in later migration slices.

### 0.2 Adjustment Summary
Previously drafted CLI file (`cli.go`) and success criteria that referenced interactive commands are now explicitly deferred to a dedicated “CLI Integration & Operator Tooling” phase (designated Phase 7). Phase 5C will deliver a stable *programmatic* API (Go surface) and internal events so that later layers (CLI, service endpoints, automation) can attach without churn.

### 0.3 Implications
- Scope narrowed: fewer artifacts this phase; deeper focus on correctness & safety.  
- Interfaces will be documented in `config-api.md`; no user CLI docs yet.  
- Tests will exercise API functions directly (simulate, apply, rollback) via Go calls.  
- Observability remains in-scope to guarantee future tooling can consume change events immediately.

---

## 1. Executive Summary

Phase 5C elevates configuration from static structs + runtime adjustments to a _governed configuration platform_. It enables:

- Layered configuration resolution (global → environment → domain → site → ephemeral overrides)
- Policy evolution with safe preview & simulation
- Atomic hot reload with rollback guarantees
- Versioning, audit, and integrity verification
- Controlled rollout (canary / percentage / cohort)
- Observability of configuration impact (latency, success, rule hit distribution)

This phase unlocks operational maturity and sets the foundation for plugin ecosystems and multi-tenant deployments in later phases.

---

## 2. Success Criteria (Must ALL Pass Before Completion)

| Category      | Criterion                                                      | Validation Method                               |
| ------------- | -------------------------------------------------------------- | ----------------------------------------------- |
| Hierarchy     | Layered resolution implemented (5 layers)                      | Unit tests (resolution matrix)                  |
| Versioning    | Each applied config assigned immutable version + hash          | Tests assert monotonic version & stored digest  |
| Audit         | Append-only audit log with actor, timestamp, diff summary      | Tests read log & verify immutability            |
| Simulation    | Dry-run evaluation shows diff & projected metric impact (mock) | Simulation test fixtures                        |
| Hot Reload    | Atomic apply with rollback on validation failure               | Integration-style test with induced failure     |
| Rollout       | Percentage / cohort-based staged activation                    | Tests vary cohort & assert selective activation |
| Integrity     | SHA256 (or blake2b) digest mismatch detection                  | Corruption test triggers rejection              |
| Programmatic API | Stable internal Go API (no CLI) for apply/simulate/rollback | Unit tests invoke exported functions            |
| Observability | Config change events exported (metrics/log/trace event)        | Monitoring tests assert emission                |
| Safety        | Validation gate prevents partial invalid merges                | Negative test cases                             |
| Docs          | Developer & operator (programmatic) guides completed           | Markdown existence + checklist                  |
| Quality       | Zero lint issues, all new code covered by tests                | Lint + coverage delta                           |

---

## 3. Scope & Non-Scope

### In Scope

- Hierarchical configuration resolver
- Policy override precedence rules
- Versioned config registry + persistence adapter (in-memory first, file persistence optional)
- Apply pipeline: parse → validate → simulate (optional) → commit → broadcast → observe
- Rollback of last N versions (at least 5) with integrity check
- Basic canary: percentage of _domains_ or _site groups_
- Emission of structured change events (logger + Prometheus counter + trace annotation)
- Programmatic Go API surface (no CLI)

### Out of Scope (Defer)

- CLI command set (`config apply/simulate/diff/rollback/show/promote`) – moved to Phase 7 (dedicated CLI package)
- Multi-tenant boundary enforcement (Phase 6 or later)
- External secret management integration
- Distributed consensus (single-node authoritative assumed)
- Plugin-provided config schema injection (Phase 5F+)

---

## 4. Architecture Overview

### 4.1 Proposed Packages / Files

```
packages/engine/configx/              # Configuration subsystem (internal-programmatic)
  resolver.go                         # Hierarchical resolution logic
  layers.go                           # Layer definitions & precedence constants
  store.go                            # Versioned store (memory + file adapter)
  model.go                            # Canonical config structs (extended)
  validation.go                       # Structural + semantic validation
  simulation.go                       # Dry-run / impact modeling
  apply.go                            # Apply pipeline orchestration
  rollout.go                          # Canary / cohort logic
  audit.go                            # Append-only audit log + hashing
  events.go                           # Emission helpers (metrics/log/trace)
  api.go                              # Public (Go) API surface for engine integration
  middleware.go                       # (optional) hook points for future policy enforcement
  testdata/                           # Sample layered configs & diffs
```

`cli.go` intentionally excluded until Phase 7 (dedicated CLI module / package introduction) to prevent premature coupling.

### 4.2 Layer Precedence (Highest wins)

1. Ephemeral Override (runtime injection)
2. Site Layer (domain/site specific)
3. Domain Layer (pattern-based groups)
4. Environment Layer (dev/staging/prod)
5. Global Base Layer (default constants)

Resolution rule: Evaluate from base upward; each layer overlays only explicitly set fields (deep merge semantics). Provide deterministic merge order; document precedence.

### 4.3 Version & Audit Model

- Version monotonic int64 (start at 1)
- Store record: `{version, timestamp, actor, hash, parent_version, diff_summary, full_spec}`
- Hash: SHA256 of canonical JSON (sorted keys) of `full_spec`
- Audit log append: write-once sequence (in-memory slice + optional file WAL)
- Rollback: re-apply previously stored full spec as new version with `reason: rollback(<targetVersion>)`

### 4.4 Apply Pipeline (States)

```
PARSE -> VALIDATE -> (SIMULATE?) -> STAGE -> COMMIT (version assigned) -> BROADCAST -> OBSERVE
```

Failure before COMMIT => no state mutation. Failure after COMMIT must still record event & allow rollback.

### 4.5 Simulation Engine (Minimal Viable)

- Input: candidate config + synthetic or captured recent metrics snapshot
- Output: struct with projected impacts (rule count deltas, potential latency adjustments, new strategy activation flags)
- Implement with mock heuristics initially; real performance modeling can evolve later.

### 4.6 Rollout Strategy (Phase 5C baseline)

- `RolloutSpec` includes mode: `full|percentage|cohort`
- Percentage: apply config to subset of domains chosen by deterministic hash(domain) < threshold
- Cohort: explicit list of domains / regex groups
- Non-included domains continue using last stable version until promotion

### 4.7 Observability Hooks

Emit on successful commit:

- Logger: level=INFO, fields: version, hash, actor, mode, rolloutWindow, diffSummary
- Metrics: counter `config_applies_total{status="success|failure"}`; gauge `config_active_version`
- Trace: span event `config.apply` w/ attributes (version, size_bytes, changed_fields)
- Optional: histogram of apply latency

---

## 5. Data Structures (Initial Draft)

```go
// layers.go
const (
  LayerGlobal = iota
  LayerEnvironment
  LayerDomain
  LayerSite
  LayerEphemeral
)

// model.go (augment existing engine config where needed)
type EngineConfigSpec struct {
  Global     *GlobalConfigSection     `json:"global,omitempty"`
  Crawling   *CrawlingConfigSection   `json:"crawling,omitempty"`
  Processing *ProcessingConfigSection `json:"processing,omitempty"`
  Output     *OutputConfigSection     `json:"output,omitempty"`
  Policies   *PoliciesConfigSection   `json:"policies,omitempty"`
  Rollout    *RolloutSpec             `json:"rollout,omitempty"`
}

type RolloutSpec struct {
  Mode              string   `json:"mode"` // full|percentage|cohort
  Percentage        int      `json:"percentage,omitempty"`
  CohortDomains     []string `json:"cohort_domains,omitempty"`
  CohortDomainGlobs []string `json:"cohort_domain_globs,omitempty"`
}

type VersionedConfig struct {
  Version     int64
  Spec        *EngineConfigSpec
  Hash        string
  AppliedAt   time.Time
  Actor       string
  Parent      int64
  DiffSummary string
}

type ApplyOptions struct {
  Actor        string
  DryRun       bool
  Force        bool // bypass simulation failure (admin only)
  RolloutStage bool // mark as staged but not active globally
}
```

---

### 6. Deferred CLI Command Sketch (Phase 7 Preview)

The following is retained only as a forward-looking reference and is **not** deliverable in Phase 5C. It will be re-evaluated when the discrete CLI package is established.

| Command (future)               | Purpose                               |
| ------------------------------ | ------------------------------------- |
| `config show [--version N]`    | Display active or specific version    |
| `config apply -f file.yaml`    | Apply new config                      |
| `config simulate -f file.yaml` | Run simulation only                   |
| `config diff --from A --to B`  | Show diff between versions            |
| `config rollback --to N`       | Roll back to prior version            |
| `config rollout promote`       | Promote staged cohort rollout to full |

Acceptance tests for these commands are out-of-scope until Phase 7.

---

## 7. Testing Strategy

| Layer          | Tests                                                                           |
| -------------- | ------------------------------------------------------------------------------- |
| Unit           | Resolver precedence matrix, hashing, diff summarization, validator edge cases   |
| Simulation     | Dry-run heuristic correctness, failure path                                     |
| Apply Pipeline | Success path, validation failure, simulation failure (no commit), forced commit |
| Rollout        | Percentage boundary (0%, 50%, 100%), cohort inclusion/exclusion                 |
| Audit          | Immutability, version monotonicity, rollback semantics                          |
| Observability  | Metrics increments, log pattern, trace annotation injection                     |
| CLI            | Golden output for apply/show/diff/rollback                                      |

Add fast deterministic tests; avoid real network or filesystem except optional persistence tests (behind build tag `integration`).

---

## 8. Incremental Implementation Plan

| Iteration | Focus                          | Output Artifacts                        |
| --------- | ------------------------------ | --------------------------------------- |
| 1         | Layer & model scaffolding      | layers.go, model.go, basic tests        |
| 2         | Resolver + deep merge          | resolver.go + precedence tests          |
| 3         | Store + versioning + hashing   | store.go, audit.go, tests               |
| 4         | Validation + simulation stubs  | validation.go, simulation.go            |
| 5         | Apply pipeline + rollback      | apply.go, rollout.go                    |
| 6         | Observability events + metrics | events.go, monitoring integration tests |
| 7         | Hardening & edge cases         | corruption tests, race detector run     |
| 8 (Phase 7) | CLI integration (deferred)   | Separate CLI pkg, command tests (future)|

Agents MUST commit after each iteration with concise message: `phase5c: <iteration focus>`.

---

## 9. Risk & Mitigation

| Risk                       | Impact                    | Mitigation                                       |
| -------------------------- | ------------------------- | ------------------------------------------------ |
| Merge explosion complexity | Incorrect overrides       | Strict precedence tests (matrix coverage)        |
| Partial invalid apply      | Runtime instability       | Atomic staging + validation gating               |
| Silent config drift        | Undetected divergence     | Digest verification + periodic hash check metric |
| Rollout mis-scoping        | Incorrect domain exposure | Deterministic hashing + explicit cohort tests    |
| Audit tampering (future)   | Trust erosion             | Append-only design + optional signed hash hook   |

---

## 10. Observability Extensions

Add new Prometheus metrics:

- `config_applies_total{status}` counter
- `config_active_version` gauge
- `config_rollout_mode{mode}` gauge (0/1 per mode)
- `config_simulation_failures_total` counter
- `config_rollback_total` counter

Add trace event: `config.apply`. Add structured log at INFO level.

---

## 11. Documentation Deliverables

| File                          | Purpose                                                           |
| ----------------------------- | ----------------------------------------------------------------- |
| `PHASE5C_INSTRUCTIONS.md`     | (This file) authoritative phase plan                              |
| `phase5c-progress.md`         | Running progress log (agents append per iteration)                |
| `config-platform-overview.md` | Conceptual architecture & lifecycle diagrams                      |
| `config-operations-guide.md`  | Operator/programmatic runbook (API usage, rollback, recovery)     |
| `config-api.md`               | Public Go API contracts (godoc excerpts)                          |
| `cli-design-draft.md` (future)| (Deferred) CLI specification placeholder for Phase 7 (not now)    |

---

## 12. Definition of Done Checklist

All MUST be true:

- [ ] All success criteria table rows validated
- [ ] 90%+ test coverage for `configx` package
- [ ] Zero lint issues (golangci-lint)
- [ ] Race detector clean (`go test -race ./...` for new code)
- [ ] Documentation deliverables present & peer-reviewed (excluding deferred CLI docs)
- [ ] Example configuration set (at least 3 layered scenarios) committed
- [ ] Rollback and simulation demonstrated in tests
- [ ] Monitoring metrics visible in test harness
- [ ] Deferred CLI clearly marked and excluded from coverage stats

---

## 13. Immediate First Steps (Agent Playbook)

1. Create directory `packages/engine/configx/` with placeholder `README.md` + empty `model.go`.
2. Implement Iteration 1 (scaffolding) + unit tests.
3. Commit: `phase5c: iteration1 scaffolding`.
4. Proceed to Iteration 2 only after tests & lint pass.
5. Update `phase5c-progress.md` after each commit with: timestamp, iteration, summary.

---

## 14. Style & Conventions

- Keep commits small & purposeful.
- Use feature flags or build tags only if strictly necessary.
- Avoid premature optimization; prefer clarity.
- Public API surfaces must have godoc comments.
- Deterministic ordering for merged config (stable JSON marshal).

---

## 15. Out-of-Band Enhancements (Optional After Core Done)

- File-backed WAL (`config_audit.log`) with rotation rules
- Signed commit hook for config integrity (ed25519)
- Policy diff semantic categorization (add/remove/modify risk levels)
- Event streaming to external bus (future multi-node)

---

**Ready to Execute.** Agents: begin at Section 13. Report deviations immediately.
