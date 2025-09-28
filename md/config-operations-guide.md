> ARCHIVED: The dynamic configuration subsystem (ConfigX) was removed. Operational runbook retained for historical context. See ADR `md/decisions/2025-09-configx-removal.md`.

# (Archived) Config Operations Guide

Historical runbook for using ConfigX programmatically (CLI was never shipped before removal).

## 1. Typical Lifecycle

1. Construct or load layered specs.
2. Resolve + produce candidate (resolver integration – placeholder until layered inputs wired).
3. Validate & simulate using `Applier.Apply` with `DryRun: true`.
4. If acceptable, apply for real (DryRun: false).
5. Monitor events & metrics for success/failure.
6. Rollback if necessary.

## 2. Safety Checklist Before Apply

- [ ] Validation passes (`ValidateSpec`).
- [ ] SimulationImpact.Acceptable OR Force authorized.
- [ ] Hash of previous head verified (optional periodic `Verify`).
- [ ] Rollout spec (if used) has correct mode & parameters.

## 3. Rollback Procedure

1. Identify stable version (e.g., `stableVersion := head.Version - 1`).
2. Call `Applier.Rollback(stableVersion, actor)`.
3. Confirm new version created and active version gauge reflects it.
4. Monitor events for `rollback` type.

## 4. Integrity Monitoring

Periodically run:

```go
for v := int64(1); v <= store.NextVersion()-1; v++ {
  if err := store.Verify(v); err != nil { /* alert */ }
}
```

Automate on a timer or background maintenance goroutine.

## 5. Metrics Integration (Prometheus Example Sketch)

Map interface to counters:

- `IncApplySuccess` -> `config_applies_total{status="success"}`
- `IncApplyFailure` -> `config_applies_total{status="failure"}`
- `IncRollback` -> `config_rollbacks_total`
- `SetActiveVersion` -> `config_active_version`

## 6. Event Consumers

Attach listeners to drive:

- Structured logging
- Change notifications (webhooks, Slack)
- Cache invalidation / hot reload triggers elsewhere in the engine

## 7. Failure Modes & Responses

| Failure Type         | Detection                           | Response                                      |
| -------------------- | ----------------------------------- | --------------------------------------------- |
| Validation error     | Apply returns error                 | Fix spec & retry                              |
| Simulation rejected  | Apply returns ErrSimulationRejected | Adjust change or use Force if approved        |
| Hash mismatch        | `Verify` returns ErrHashMismatch    | Investigate corruption; rebuild from snapshot |
| Rollback target miss | Rollback returns not found error    | Choose valid version                          |

## 8. Forcing a Change

Use only with explicit approval. `Force` bypasses simulation acceptability gate but not validation.

## 9. Extending Simulation

- Add rule complexity weighting
- Integrate latency percentiles from previous versions
- Emit risk score (0-1)

## 10. Future CLI Mapping

Planned commands will wrap operations here (apply, simulate, rollback, diff, show). Keep API stable to minimize translation complexity.

## 11. Disaster Recovery (Future Persistence)

When persistence added:

- Rehydrate `VersionedStore` from WAL.
- Recompute hashes; abort if mismatch (unless using signed commit workflow).

For now (in-memory only), treat process restart as cold start (reapply baseline config at startup).
