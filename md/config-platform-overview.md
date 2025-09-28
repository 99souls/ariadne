> ARCHIVED: The dynamic configuration subsystem (ConfigX) was removed. Historical reference only. See `md/archive/configx-README-legacy.md` and ADR `md/decisions/2025-09-configx-removal.md`.

# (Archived) Configuration Platform Overview (ConfigX)

This document (historical) explained the architecture, lifecycle, and design principles of the removed Phase 5C configuration & policy management subsystem (ConfigX).

## Goals

- Deterministic hierarchical merging (global -> environment -> domain -> site -> ephemeral)
- Safe evolution via validation + simulation gating
- Immutable versioned history with cryptographic integrity (SHA256)
- Fast rollback and auditability
- Observability: events + metrics for change tracking

## High-Level Flow

```
Layered Specs -> Resolver -> Candidate Spec -> Validation -> Simulation -> Apply (Store Append) -> Events + Metrics -> Active Version
```

## Layering Semantics

Higher precedence overwrites lower only where fields are explicitly set. Maps merge (overwriting specific keys), slices replace entirely, pointer sections overlay atomically. All mutations operate on clones to preserve immutability of historical specs.

## Versioning & Audit

Each successful apply produces a `VersionedConfig` with monotonic version number and SHA256 hash of canonical JSON. An append-only in-memory log is retained; tests assert immutability by deep copying audit entries on read.

## Simulation

Current heuristic (MVP):

- Projected latency = +1ms per added rule.
- Acceptable if added rules < 20 and latency increase < 10ms.
- Flags and rules diff summarized for operators.

Future extensions: integrate live metrics baselines, richer risk scoring.

## Rollout & Rollback

`RolloutEvaluator` provides per-domain active version selection:

- full: all domains receive head version.
- percentage: deterministic FNV32a hash(domain) < percentage threshold -> head, else previous version.
- cohort: listed domains receive head; others fall back.

Rollback re-applies a previous `Spec` as a new version with `DiffSummary` = `rollback(<target>)`. It does not delete history, preserving traceability.

## Observability

Events emit through a `Dispatcher` with types: `apply`, `rollback`, `validation_error`, `simulation_reject`, `append_error`.
Metrics (in-memory recorder for now):

- Apply success / failure counters
- Rollback counter
- Active version gauge

## Safety Guarantees

- Validation before commit prevents structurally invalid specs.
- Simulation gating (unless forced) prevents high-impact changes by default.
- Hash verification allows detection of tampering.

## Non-Goals (Phase 5C)

- Persistent storage (file/WAL) — future enhancement.
- CLI tooling — deferred to Phase 7.
- Multi-tenant isolation boundaries.

## Extensibility Points

- Implement `MetricsRecorder` with Prometheus.
- Add persistence adapter wrapping `VersionedStore`.
- Stream events to external bus.
