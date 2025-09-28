# ConfigX Internalization vs Deletion Analysis (C7 Preparation)

Status: Draft (for C7 decision)  
Audience: Maintainers / API surface stewards  
Scope: Evaluate whether to (A) internalize the existing `engine/configx` subsystem under an internal path or (B) delete it outright, retaining only the minimal `engine.Config` facade. Provides criteria, impact assessment, migration notes, and recommended path.

---

## 1. Problem Statement

`configx/` introduces a full configuration platform (versioned store, simulation, rollout evaluator, layered resolution, auditing, metrics). Core embedders today only require a _static_ declarative `Config` struct at engine construction. Keeping `configx` public risks:

- Perceived stability / accidental adoption of a complex experimental system.
- Cognitive load and maintenance drag (tests, docs, metrics semantics) for low current value.
- Headwind for future simplification or alternative config story (e.g., external controller or file-based hot reload) because public APIs become implicit contracts.

We need to decide in C7 whether to hide (internalize) or remove (delete) this subsystem before pre‑1.0 stabilization boundaries tighten.

---

## 2. Option Overview

| Option | Action                                                        | Resulting Public Surface                                                                          | Short-Term Cost                               | Long-Term Cost                       | Re-Introduction Cost                                 | Risk Profile                                             |
| ------ | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | --------------------------------------------- | ------------------------------------ | ---------------------------------------------------- | -------------------------------------------------------- |
| A      | Internalize under `engine/internal/configlayers` (or similar) | Minimal facade remains; internal engine+tests retain advanced layering for future experimentation | Medium (path + import churn, test relocation) | Medium (still maintain code & tests) | Low (already exists internally)                      | Low external breakage; internal complexity persists      |
| B      | Delete entirely (`configx/` + related docs & tests)           | Only `engine.Config` + maybe future narrow reload hook                                            | Low (delete + cleanup)                        | Low (no maintenance)                 | Medium/High (would need redesign & reimplementation) | Potential regret if advanced features become needed soon |

---

## 3. Detailed Pros & Cons

### Option A – Internalize

Pros:

- Preserves investment (logic, tests) for rapid revisit if dynamic config becomes priority.
- Enables incremental experimentation (e.g., selective rollout, policy gating) without committing APIs.
- Facilitates using parts internally (e.g., layered merging) to simplify future features (hot reload) with zero external exposure.
- Minimizes immediate churn in tests referencing these abstractions.
  Cons:
- Continues build + test cost (~execution time + mental overhead reading internal packages).
- Risk of “internal gravity” – future contributors extend a hidden system instead of designing a simpler public story.
- Must maintain allowlist guards to ensure no accidental re-export.
- Adds internal cognitive surface (future refactors must consider dormant code paths).

### Option B – Delete

Pros:

- Immediate surface & complexity reduction (code, tests, docs, metrics, failure modes all vanish).
- Clear narrative: configuration = static struct at startup; anything else is future scope.
- Faster CI/test runs (remove many unit tests & supporting fixtures / metrics counters).
- Eliminates potential security / correctness issues in unused code paths (simulation, audit hash verification, etc.).
  Cons:
- Loses invested design & validated behaviors (rollout evaluator, versioned store invariants, merging semantics).
- If dynamic config becomes urgent later, reimplementation cost + regained uncertainty (bugs reintroduced, test gap until rebuilt).
- Removes a potential internal experimentation bed for policy evolution (e.g., updating telemetry or rate limit policies atomically with rollback).
- Some documentation (platform overview, operations guide) becomes obsolete—requires editorial cleanup anyway.

---

## 4. Usage & Coupling Assessment

Current engine facade (`engine.New`) does NOT depend on `configx`. No exported symbols from `configx` are referenced externally (grep external modules: none in repo). Tests under `engine/configx/*` are self-contained. There is no cross-package import from core runtime that would block deletion. This makes deletion low risk externally.

Indirect value: Some future ideas (progressive rollout of telemetry sampling, feature flag gating for experimental crawler behaviors) _could_ reuse the versioned store + simulation primitives. However, implementing those on-demand later with a narrower design might be preferable.

---

## 5. Decision Criteria Framework

Weight (H=High, M=Medium, L=Low) relative to project goals (small surface, agility):
| Criterion | Weight | A Internalize Score | B Delete Score | Notes |
| --------- | ------ | ------------------- | -------------- | ----- |
| External API Minimization | H | 10 (hidden) | 10 | Both achieve; tie |
| Maintenance Burden Reduction | H | 5 | 10 | Deletion removes entire burden |
| Future Experiment Flexibility | M | 9 | 4 | Internal copy keeps optionality |
| Cognitive Load (contributors) | M | 6 | 10 | Less code to reason about with deletion |
| Risk of Reintroducing Bugs Later | M | 9 | 4 | Internalization preserves battle-tested logic |
| Build/Test Performance | L | 6 | 9 | Fewer tests improves speed |
| Simplicity of Narrative | M | 7 | 10 | “Just a struct” is cleaner |

Normalized indicative scores (sum, unweighted unless we apply weights):

- Internalize: 52
- Delete: 57
  Applying qualitative weighting (High counts double, Medium 1.5x, Low 1x) nudges deletion further ahead because Maintenance + Cognitive are High/Medium where deletion wins decisively.

Conclusion: Deletion slightly outperforms internalization versus our stated contraction goals unless we have a near-term (<= 1–2 milestones) plan requiring dynamic config.

---

## 6. Migration / Execution Plans

### If Internalizing (Option A)

Steps:

1. Create `engine/internal/configlayers/` (or `internal/configx/`).
2. Move all `.go` files & tests; rename package to `configlayers` (internal) to prevent accidental re-import.
3. Adjust imports in moved tests; run `go test ./engine/...`.
4. Delete public `engine/configx` directory.
5. Purge docs: replace with a short note in `internalisation-plan.md` referencing internal retention.
6. Update CHANGELOG (Removed: public configx; Added: internal config layering retained for future experimentation).
7. Update any allowlist guard (if one exists for `configx`) to exclude new internal path.

### If Deleting (Option B)

Steps:

1. Delete `engine/configx/` directory entirely.
2. Remove related docs: `md/config-api.md`, `md/config-platform-overview.md`, `md/config-operations-guide.md` (or archive under `md/archive/`).
3. Remove references in `internalisation-plan.md` (action matrix + checklist). Mark C7 as DONE with note: deleted instead of internalized.
4. Update CHANGELOG (Removed: configx subsystem, rationale). Add explicit note that only static `Config` is supported; dynamic layer future-tracked.
5. Run tests & ensure no residual imports.
6. (Optional) Add a focused guard test asserting `engine` has no `configx` subpackage.
7. (Optional) Keep a concise ADR file `md/decisions/2025-09-configx-removal.md` summarizing why removed.

Time Estimate:

- Internalize: ~1.5–2h (rename + fix tests + docs)
- Delete: ~30–45m (delete + docs + changelog + guard)

---

## 7. Risk & Mitigations

| Risk                                           | Option      | Mitigation                                                                                        |
| ---------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------- |
| Hidden (future) need for partial layer merging | Delete      | Document reintroduction path; snapshot this directory before removal (git history).               |
| Internal code drifts unused (dead weight)      | Internalize | Periodic dead code scan; if unused after 2 milestones, purge.                                     |
| Contributors resurrect complexity prematurely  | Internalize | Add file header comment: "Experimental – do NOT re-export."                                       |
| Lost test patterns for future dynamic config   | Delete      | Capture representative tests (store versioning, rollout eval) in an ADR appendix before deletion. |

---

## 8. Recommendation

Proceed with Option B (Delete) unless a concrete near-term roadmap item (already prioritized) needs dynamic or staged configuration. Current roadmap (through C8) is focused on surface contraction and telemetry hardening – no such item exists.

Rationale: Eliminates maintenance & cognitive overhead now; git history preserves implementation. Reintroduction (if ever) can start from a cleaner design informed by actual use cases instead of speculative generality.

---

## 9. Post-Decision Follow-Ups (if Deleting)

- Draft ADR `decisions/2025-09-configx-removal.md` (point to this analysis; summarize rationale in ~6 bullets).
- Add explicit statement in README under Configuration: "Dynamic / live layered configuration is intentionally not part of the pre‑1.0 scope."
- Close or update any tracking issues referencing ConfigX.

---

## 10. Appendix

### A. Representative Reintroduction Slice (If Needed Later)

Minimal dynamic reconfig could be a single interface:

```go
// DynamicProvider supplies updated Config snapshots.
type DynamicProvider interface { Next(context.Context) (*Config, error) }
```

Engine could accept an optional provider and apply safe in-place transitions guarded by a validation hook. No need for full versioned store or simulation until proven.

### B. Snapshot of Key Capabilities Being Removed

- VersionedStore: historical hash chain + corruption detection.
- Simulator: pre-apply diff impact analysis.
- Applier: force/rollback flags + audit metadata.
- RolloutEvaluator: progressive targeting semantics.
- InMemoryMetrics / Collector: granular counters for operations.

### C. Why History Is Enough as Backup

All code remains in git (tag before removal). Reintroduction can cherry-pick or re-copy with modernization (narrow interfaces, context usage, error vocabulary alignment).

---

Prepared by: (assistant generated)  
Date: 2025-09-28
