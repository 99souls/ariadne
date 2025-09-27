# ConfigX Programmatic API

This is the Go-facing API exposed by the configuration subsystem. CLI exposure is deferred (Phase 7).

## Core Types

- `EngineConfigSpec` – Root configuration payload composed of section pointers.
- `VersionedConfig` – Stored immutable version with hash & metadata.
- `VersionedStore` – Append-only in-memory store with audit log.
- `Simulator` – Produces `SimulationImpact` describing projected change effects.
- `Applier` – Orchestrates validate -> simulate -> commit -> emit.
- `RolloutEvaluator` – Determines per-domain active version during staged rollout.
- `Dispatcher` – In-process event broadcaster.
- `MetricsRecorder` – Interface for metrics integration.

## Construction

```go
store := configx.NewVersionedStore()
sim := configx.NewSimulator()
applier := configx.NewApplier(store, sim)

// Optional instrumentation
dispatcher := configx.NewDispatcher()
metrics := &configx.InMemoryMetrics{}
applier.WithDispatcher(dispatcher).WithMetrics(metrics)
```

## Applying a Change

```go
currentSpec := (*configx.EngineConfigSpec)(nil) // first apply
candidate := &configx.EngineConfigSpec{
  Global: &configx.GlobalConfigSection{MaxConcurrency: 8, LoggingLevel: "info"},
  Policies: &configx.PoliciesConfigSection{EnabledFlags: map[string]bool{"feature_x": true}},
}
res, err := applier.Apply(currentSpec, candidate, configx.ApplyOptions{Actor: "deployer"})
if err != nil { /* handle */ }
fmt.Println("Applied version", res.Version, "hash", res.Hash)
```

Dry-run + simulation only:

```go
impactResult, _ := applier.Apply(currentSpec, candidate, configx.ApplyOptions{Actor: "preview", DryRun: true})
// impactResult.Version == 0
```

Force override simulation rejection:

```go
res, err := applier.Apply(currentSpec, riskyCandidate, configx.ApplyOptions{Actor: "admin", Force: true})
```

## Rollback

```go
rollbackRes, err := applier.Rollback(3, "operator")
```

## Rollout Evaluation

```go
ev := configx.NewRolloutEvaluator(store)
activeVersion := ev.ActiveVersionForDomain("example.com")
```

If the head spec uses `percentage` or `cohort` rollout modes, excluded domains fall back to the previous version until promotion.

## Events

Register a listener:

```go
collector := &configx.InMemoryCollector{}
dispatcher.Register(collector)
```

`ChangeEvent` fields:

- Type (apply|rollback|validation_error|simulation_reject|append_error)
- Version
- Hash
- Actor
- Error (may be nil)
- Timestamp

## Metrics

Implement `MetricsRecorder` to integrate with your telemetry backend. In tests we use `InMemoryMetrics` to assert increments.

```go
type PromMetrics struct { /* counters & gauges */ }
func (p *PromMetrics) IncApplySuccess() {}
// ... implement remaining interface
```

## Integrity Verification

```go
if err := store.Verify(5); err == configx.ErrHashMismatch { /* handle corruption */ }
```

## Validation Errors

`ValidateSpec` returns errors for:

- Invalid rollout mode
- Percentage out of bounds
- Negative concurrency / retries

These surface through `Applier.Apply` before any commit.

## Simulation Heuristic (MVP)

- Acceptable if added rules < 20 and projected latency increase < 10ms.
- `SimulationImpact.Acceptable` indicates gating outcome.

## Extending

- Add rollout logic (percentage/cohort) inside a new `rollout.go`.
- Replace heuristic in `simulation.go` with domain metrics.
- Provide persistence layer (e.g. file WAL) calling through to `VersionedStore.Append`.
