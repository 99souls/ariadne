# Engine Package (Proposed Extraction)

This package will encapsulate the reusable, headless crawling and processing engine logic, separating it from any CLI/TUI or distribution concerns.

## Goals

- Provide a stable Go API for embedding the site scraping pipeline in other applications.
- Isolate business logic (discovery, rate limiting, extraction, processing, asset handling) from presentation and interface layers.
- Enable future packages like `tui`, `cli`, or `api` to depend on `engine` without import cycles or leaking internal details.

## Candidate Migrations

The following internal packages are prime candidates to move (with minimal or no exported surface changes) into `packages/engine`:

| Current Path                | New Path (Proposed)                     | Notes |
|----------------------------|-----------------------------------------|-------|
| `pkg/models`               | `packages/engine/models`                | Public data structures. Keep backward compatibility via re-export stubs if needed. |
| `internal/pipeline`        | `packages/engine/pipeline`              | Core multi-stage pipeline. Export high-level orchestration API. |
| `internal/ratelimit`       | `packages/engine/ratelimit`             | Adaptive limiter. Will become public API surface (careful versioning). |
| `internal/processor`       | `packages/engine/processor`             | Content extraction & markdown conversion. Audit for internal-only helpers. |
| `internal/assets`          | `packages/engine/assets`                | Asset discovery/downloading/rewriting. Consider making downloader interface-driven. |
| `internal/crawler`         | `packages/engine/crawler`               | URL discovery and queue management. |
| `internal/config`          | `packages/engine/config`                | Configuration loader & defaults. Possibly unify with models. |
| `internal/output`          | `packages/engine/output`                | Future output pipeline components. |

## Architectural Approach

1. Introduce `packages/engine` as a new module layer WITHOUT moving code initially.
2. Define a cohesive `Engine` facade interface that composes pipeline + limiter + configuration.
3. Incrementally relocate packages from `internal/` preserving import paths temporarily via forwarding shims (type aliases) to avoid massive single diff.
4. Once stability is confirmed, deprecate old `internal/*` entry points and update imports.
5. Add versioning strategy (semver tags at repo root) documenting public API expectations.

## Engine Facade (Draft)

```go
// packages/engine/engine.go (future)
package engine

type Engine struct {
    cfg    Config
    pl     *pipeline.Pipeline
    limiter ratelimit.RateLimiter
}

func New(cfg Config) (*Engine, error) { /* wire components */ }
func (e *Engine) Start(ctx context.Context, seeds []string) error { /* start discovery */ }
func (e *Engine) Stop(ctx context.Context) error { return e.pl.Stop() }
func (e *Engine) Snapshot() Snapshot { /* aggregate stats */ }
```

## Migration Strategy

Phase 0 (Now): Document plan and create engine package scaffold.  
Phase 1: Define facade interfaces + minimal wiring referencing existing `internal/*` packages.  
Phase 2: Move stateless/shared models (`pkg/models` → `packages/engine/models`). Provide re-export file at `pkg/models` to maintain compatibility.  
Phase 3: Relocate `ratelimit` & `pipeline` (lowest external dependencies).  
Phase 4: Move `processor`, `assets`, `crawler` with careful public function curation.  
Phase 5: Remove/alias deprecated imports; update `go vet` & tests.  
Phase 6: Introduce TUI/CLI packages consuming the engine.  

## Open Questions

- Do we want a separate Go module (multi-module repo) or a single module with subpackages? (Recommendation: stay single-module initially for simplicity.)
- Which APIs become officially supported vs. experimental? Need doc tags.
- How will configuration validation errors be surfaced (error types vs. sentinel errors)?

## Immediate Next Steps

1. Approve directory strategy (`packages/engine`).
2. Add draft `engine.go` defining interface skeletons.
3. Create compatibility plan for `pkg/models` re-export.
4. Start with non-invasive facade referencing `internal/*` to prove layering.

---

This README is a living document and will evolve as the extraction proceeds.
