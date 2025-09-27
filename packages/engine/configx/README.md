# ConfigX Subsystem

Phase 5C hierarchical configuration & policy management for the engine.

## Features (MVP)

- Layered spec model (merge semantics implemented in `resolver.go`)
- Versioned append-only store with SHA256 integrity (`store.go`)
- Validation gate (`validation.go`)
- Simulation heuristic (latency + rule delta acceptance) (`simulation.go`)
- Apply & rollback orchestration (`apply.go`)
- Audit log & hash verification (`audit.go`, `store.go`)
- Observability (events dispatcher + pluggable metrics) (`events.go`, `metrics.go`)
- Hardening tests (corruption, edge cases, concurrency)

## Quickstart

```go
store := configx.NewVersionedStore()
sim := configx.NewSimulator()
ap := configx.NewApplier(store, sim)
dispatcher := configx.NewDispatcher()
metrics := &configx.InMemoryMetrics{}
ap.WithDispatcher(dispatcher).WithMetrics(metrics)

candidate := &configx.EngineConfigSpec{
	Global: &configx.GlobalConfigSection{MaxConcurrency: 8, LoggingLevel: "info"},
	Policies: &configx.PoliciesConfigSection{EnabledFlags: map[string]bool{"feature_x": true}},
}
res, err := ap.Apply(nil, candidate, configx.ApplyOptions{Actor: "deployer"})
if err != nil { panic(err) }
fmt.Println("Applied version", res.Version, res.Hash)
```

Dry-run simulation:

```go
impact, _ := ap.Apply(nil, candidate, configx.ApplyOptions{Actor: "preview", DryRun: true})
fmt.Println("Acceptable?", impact.SimImpact.Acceptable)
```

Rollback:

```go
_, _ = ap.Rollback(1, "operator")
```

## Docs

- `md/config-platform-overview.md` – architecture & lifecycle
- `md/config-api.md` – programmatic API usage
- `md/config-operations-guide.md` – operator workflow & runbook
- `md/phase5c-progress.md` – iteration log

## Test Data

Sample layered fragments in `testdata/`:

- `global.json`
- `environment_prod.json`
- `site_example_com.json`

## Next (Future Phases)

- CLI tooling (Phase 7)
- Persistence adapter (WAL / snapshot)
- Advanced rollout strategies & cohort hashing
- Rich simulation based on live metrics

## Integrity & Safety

Use `store.Verify(version)` to detect tampering.
Simulation gating can be overridden only with `Force: true` on applies.

## License

MIT (see root LICENSE).
