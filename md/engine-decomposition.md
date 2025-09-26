# Engine Package Decomposition Design

Status: Active (Initiated during Phase 3)
Owner: Architecture / Platform Track
Last Updated: September 26, 2025

---

## 1. Problem Statement

Current business logic (pipeline, rate limiting, processing, assets, crawler) resides under `internal/` with public models in `pkg/models`. As we introduce richer interfaces (CLI, potential TUI, API endpoints, output formats), the absence of a cohesive public engine boundary risks:

- Tight coupling between presentation layers and low-level internals
- Expanding import surface area and cross-package dependencies
- Higher refactor cost later when behavior is entrenched

## 2. Goals

| Goal                   | Description                                                             | Success Metric                                           |
| ---------------------- | ----------------------------------------------------------------------- | -------------------------------------------------------- |
| Stable Embedding API   | Provide a single facade for starting/stopping crawls & retrieving stats | New CLI uses only `packages/engine`                      |
| Maintainability        | Reduce direct imports of legacy `internal/*` packages                   | 0 new imports added outside engine after migration start |
| Incremental Migration  | Avoid big-bang refactor; keep tests green each step                     | All tests pass per phase                                 |
| Backward Compatibility | No breaking changes for existing internal tests & build pipeline        | `go test ./...` unchanged                                |

## 3. Non-Goals

- Immediate multi-module split (stay single module now)
- Public API version tagging (defer until facade stabilizes)
- Rewriting core algorithms (structural relocation only initially)

## 4. Architectural Approach

Introduce `packages/engine` containing a facade encapsulating internal subsystems:

```
+------------------+        +------------------+        +------------------+
| CLI / TUI / API  |  --->  |  Engine Facade   |  --->  |  Subsystems      |
| (Presentation)   |        | (packages/engine)|        | (pipeline, etc.) |
+------------------+        +------------------+        +------------------+
```

The facade will own construction, lifecycle, and aggregation of metrics.

### Facade Draft

```go
package engine

type Engine interface {
    Start(ctx context.Context, seeds []string) error
    Stop(ctx context.Context) error
    Snapshot() Snapshot
}

// Concrete implementation wires pipeline + limiter + ancillary subsystems.
```

## 5. Migration Phases

| Phase | Description                                  | Output                               |
| ----- | -------------------------------------------- | ------------------------------------ |
| P0    | Scaffold engine package                      | README + placeholder (DONE)          |
| P1    | Define facade & interfaces                   | `engine/engine.go` with constructor  |
| P2    | Move `ratelimit` + forwarding shim           | `internal/ratelimit` deprecated file |
| P3    | Move `pipeline` + forwarding shim            | Updated imports                      |
| P4    | Relocate `models` → `packages/engine/models` | Re-export wrappers in old path       |
| P5    | Move `processor` & `assets`                  | Minimized exported symbols           |
| P6    | Move `crawler` & `output`                    | Unified config path                  |
| P7    | Remove shims & finalize facade               | Clean dependency graph               |
| P8    | Refactor CLI to depend solely on facade      | New simplified `main.go`             |

## 6. Forwarding Shim Pattern

```go
// Old path: internal/ratelimit/doc.go
// Deprecated: use packages/engine/ratelimit
package ratelimit

import engineRL "site-scraper/packages/engine/ratelimit"

type RateLimiter = engineRL.RateLimiter
// (repeat for required exported symbols)
```

This preserves backward compatibility while encouraging migration via deprecation comments.

## 7. Testing Strategy

- Re-run full suite + `-race` after each phase.
- Add a new `engine_integration_test.go` verifying an end-to-end crawl using only the facade.
- Use forwarding shims until all imports updated, then remove them in a single diff after CI green.

## 8. Risk Analysis

| Risk                 | Impact                        | Mitigation                                                 |
| -------------------- | ----------------------------- | ---------------------------------------------------------- |
| Shim Divergence      | Stale aliases cause confusion | Keep shims minimal; remove promptly at P7                  |
| Export Surface Creep | Hard to maintain API          | Centralize exports via facade; internalize helpers         |
| Hidden Coupling      | Blocked migration mid-phase   | Introduce interfaces (e.g., ContentProcessor) where needed |
| Test Flakiness       | Slows iteration               | Run targeted package tests before full suite               |

## 9. Metrics & Exit Criteria

- 0 direct imports of moved packages outside `packages/engine/*` (enforced via grep check script)
- All legacy tests still green (no semantic behavior change)
- New facade integration test passes
- Documentation updated (this file + plan.md section)

## 10. Open Questions

- Do we provide functional options for Engine construction vs. config struct embedding?
- Should rate limiter and pipeline stats be unified into a single snapshot schema now or later?
- Do we expose low-level subsystems (e.g., `ratelimit.Snapshot`) publicly or only aggregated view?

## 11. Immediate Next Steps

1. Implement P1: add concrete Engine struct delegating to existing internal packages.
2. Write initial facade integration test referencing current pipeline startup logic.
3. Prepare P2 move plan for `ratelimit` (inventory exported symbols & usages).

## 12. Change Log

| Date       | Change                          |
| ---------- | ------------------------------- |
| 2025-09-26 | Initial design document created |

---

This document evolves with each migration phase; keep updates atomic and test-backed.
