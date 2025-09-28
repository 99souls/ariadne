## ADR: Removal of ConfigX Subsystem (C7)

Date: 2025-09-28
Status: Accepted / Implemented
Context: `engine/configx` provided a speculative dynamic configuration platform (versioned store, rollout evaluator, simulation, audit, metrics) not required by current embedders. Maintaining it publicly inflated surface and cognitive load without proven need.

Decision: Delete the entire `engine/configx` package (code, tests, docs) instead of internalizing. Retain only the static `engine.Config` struct for pre-1.0. Dynamic or staged reconfiguration will be revisited only when a concrete use case emerges.

Consequences:

1. Public API shrinks; no perception that dynamic config is supported.
2. CI/test footprint reduced (removed N tests across configx subsystem).
3. Future reintroduction cost increases (will redesign from first principles if needed).
4. Git history preserves prior implementation (cherry-pickable).
5. Documentation updated (CHANGELOG + internalisation plan). README to add explicit note: dynamic config intentionally out of scope pre-1.0.

Alternatives Considered:

- Internalize only: rejected (retains maintenance drag, encourages quiet expansion of unused complexity).

Reintroduction Sketch:
Introduce a minimal `DynamicProvider` interface returning validated `Config` snapshots; layer advanced semantics (rollout, simulation) only after real adoption signals.

References:

- md/configx-internalization-analysis.md
- internalisation-plan.md (C7 checklist entry)
