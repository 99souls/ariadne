# Phase 5F Plan – Engine Module Extraction & CLI Enablement

Status: Draft (Pending Approval)
Date: 2025-09-27
Owner: Architecture / Core Engine

## 1. Executive Summary

Phase 5F elevates `engine` to a **standalone, minimally-coupled Go module** and introduces a first-class **CLI layer** that consumes only its public API. This establishes a clean contract boundary before Phase 6 (CLI polish & UX), reduces coupling, simplifies dependency surfaces, and paves the way for future multi-surface delivery (API server, TUI, hosted service).

Primary Outcomes:

- `engine/` becomes an independent Go module with curated public surface and internal boundaries.
- Root repository becomes a multi-module workspace via `go.work`.
- A new `cli/` (or `cmd/ariadne/`) module provides a user-facing entrypoint with configuration, metrics/health endpoints wiring, and graceful lifecycle.
- Backward-compatible migration path with temporary forwarding shims (time-boxed) for existing internal imports.
- Documentation, versioning, and stability annotations aligned with API expectations.

## 2. Problem Statement

Current state (post-Phase 5E):

- `packages/engine` acts as a facade but still resides inside the monolithic `ariadne` module.
- Business logic (crawler, processor, assets, pipeline, rate limiting, resources) already colocated under `packages/engine/`, reducing extraction friction.
- Consumers (future CLI / TUI) will currently gain transitive access to many internal details if kept single-module, risking API leakage and uncontrolled coupling.

Pain Points:
| Issue | Impact | Example |
|-------|--------|---------|
| Monolithic module | Hard to publish stable embedding API | External users must vendor entire codebase |
| Unscoped dependencies | CLI-only deps risk polluting engine | Future Cobra/TUI libs pulled into all builds |
| Ambiguous stability | No semantic boundary; accidental exports | Tests rely on internal types not meant for public use |
| Limited upgrade path | Refactors risk silent breakage | No version tags or contract docs |

## 3. Architectural Goals

1. Isolation: Engine module builds independently with only required dependencies (HTML parsing, telemetry, etc.).
2. Minimal Public Surface: Export only curated APIs (Engine facade, Config, Models, Strategy Interfaces, Telemetry hooks).
3. Internal Encapsulation: Implementation packages (pipeline, processor, assets) internalized where feasible.
4. Extensibility: Strategy interfaces (Fetcher, Processor, OutputSink, AssetStrategy) formalized for plug-ins.
5. Observability Adapters: HTTP exposure (metrics/health) remains outside engine (adapter pattern) to keep core headless.
6. Deterministic Versioning: Introduce semantic version baseline (v0.x with documented stability tiers).
7. Smooth Migration: Forwarders + deprecation notices; no immediate breakage for in-repo references during transition wave.

## 4. Option Analysis

### Option A – Stay Single Module (Status Quo)

Pros: Simplicity, no tooling change.
Cons: Dependency bloat, unclear API boundary, harder external adoption.

### Option B – Multi-Module: Engine + Root (Selected)

Structure:

```
go.work (workspace root)
engine/        (module: github.com/99souls/ariadne/engine)
cli/           (module: github.com/99souls/ariadne/cli)  # or cmd/ariadne
shared/ (TBD)  (only if genuine cross-cutting packages emerge; avoid pre-optimizing)
```

Pros: Clean contract, independent tagging, prevents CLI deps leaking.
Cons: Slight complexity (go.work), import path churn, need migration doc.

### Option C – Inverted: Root is Engine, CLI submodule

Pros: No forwarding needed.
Cons: Forces consumers to pull full repo; still couples CLI path.

Decision: **Option B** – Balanced isolation + incremental migration feasibility with low refactor risk.

## 5. Target Module Layout

### Engine Module (`/engine`)

Public Packages (initial):

- `engine` (facade & lifecycle)
- `engine/models`
- `engine/config` (optional: may fold into root engine if small)
- `engine/telemetry` (only stable integration points; internal subpackages private when possible)
- `engine/ratelimit` (if declared stable; otherwise `internal/ratelimit` inside module)

Internal (non-exported) or `internal/` packages:

- `internal/pipeline`
- `internal/processor`
- `internal/assets`
- `internal/crawler`
- `internal/resources`

Adapters (future, maybe separate module(s)):

- Metrics/health HTTP handlers (stay out of engine core).
- Fetcher implementations (e.g., `fetcher/colly`) may remain public if plugin ecosystem planned.

### CLI Module (`/cli`)

Responsibilities:

- Flag & config parsing (YAML + overrides)
- Engine instantiation & seeding
- Progress + snapshot display (human JSON / table)
- Optional: metrics server, health endpoint binding
- Graceful shutdown & signal handling

Out-of-scope for Phase 5F (defer to Phase 6 polish): TUI-rich UI, interactive wizard, plugin discovery.

## 6. Public API Curation Strategy

Stability Tiers:
| Tier | Tag | Policy |
|------|-----|--------|
| Stable | (none yet; goal Phase 6) | Backward compatible within minor versions |
| Experimental | `// Experimental:` | May change with minor release; documented risks |
| Internal | no doc / in `internal/` | Not part of public contract |

Immediate Actions:

- Audit exported symbols; add doc comments & stability annotations.
- Collapse rarely used helper exports into internal.
- Introduce strategy interfaces: `Fetcher`, `Processor`, `OutputSink`, `AssetHandler` (naming finalization).

## 7. Migration Waves & Work Breakdown

### Wave 0 – Planning & Guard Rails

Tasks:

- Freeze main branch for engine-surface changes until curation complete.
- Create `phase5f-plan.md` (this file) & update CHANGELOG Unreleased section.
- Add architecture enforcement test (ensures CLI imports `engine` only, not its internals).

Exit Criteria: Plan approved; test failing (red) until modules exist.

### Wave 1 – Module Skeleton

Tasks:

- Move `packages/engine` → `engine/` (filesystem move only).
- Create `engine/go.mod` with module path `github.com/99souls/ariadne/engine` (adjust to canonical repo path; placeholder if private).
- Add `go.work` at repo root with `use ./engine` and root module (renamed or retained for CLI/dev tooling).
- Run `go vet`, `go test ./...` inside engine module.

Risk Mitigation: Leave original `packages/engine` directory as stub with `README-MOVED.md` + forwarding build tags (very short-lived) for clarity.

### Wave 2 – Import Path Refactor

Tasks:

- Update all internal repo imports from `ariadne/packages/engine/...` → `github.com/99souls/ariadne/engine/...`.
- Introduce temporary forwarding aliases (one file per old path) using `// Deprecated: moved` + type/function re-exports (remove by Wave 5).
- Ensure tests pass across workspace via `go.work` root invocation.

### Wave 2.5 – Root Purge (Part 1)

Purpose: Eliminate root runtime entrypoint to prevent dual-path drift.

Tasks:

- Move `main.go` → `cli/cmd/ariadne/main.go` (temporary location until CLI module Wave 4 is created; may stage under root then finalize in Wave 4 if sequencing requires).
- Add guard script / test: fail build if any non-test `.go` file exists at repo root (except tooling stubs).
- Document change in CHANGELOG (Unreleased: "root executable moved to cli module").
- Inventory remaining root `cmd/` directories; flag for Wave 3.5 disposition.

Exit Criteria: Root contains no executable Go files; tests still green.

### Wave 3 – API Surface Audit & Pruning

Tasks:

- Identify unintended exports (grep for `exported but not documented`).
- Move implementation details to `internal/` inside engine module.
- Introduce `interfaces.go` for strategy contracts.
- Add doc comments & stability annotations.

### Wave 3.5 – Root Purge (Part 2)

Purpose: Remove lingering legacy command code & shadow copies of business logic.

Tasks:

- Migrate or archive `cmd/scraper` & `cmd/test-phase1`.
  - If kept for historical reference: move to `archive/` with `//go:build ignore`.
- Remove / migrate any root `internal/*` duplicates now superseded by engine module.
- Add enforcement test: ensure no imports reference old paths (`ariadne/packages/engine`).
- Add CI check (simple grep) preventing reintroduction of root executables.

Exit Criteria: Root tree = docs + workspace files only (no active code, no legacy commands).

### Wave 4 – CLI Module Bootstrapping

Tasks:

- Create `cli/go.mod` (module `github.com/99souls/ariadne/cli`).
- Implement minimal command:
  - `ariadne crawl --seeds urls.txt --config config.yaml`
  - Options: `--metrics :9090`, `--health :9091`, `--resume`, `--snapshot-interval 10s`
- Provide graceful shutdown (SIGINT/SIGTERM) + periodic snapshot log.
- Add smoke integration test launching engine with in-memory site fixture.

### Wave 5 – Forwarding Shim Removal

Tasks:

- Remove deprecated re-export files.
- Run `staticcheck`, `golangci-lint` (if configured) to ensure no stale references.
- Update documentation: migration guide + stability matrix.

### Wave 6 – Versioning & Tag Baseline

Tasks:

- Tag engine module `v0.5.0` (example) once CLI uses only curated API.
- Add `API_STABILITY.md` section for engine module.
- Update root README with embedding example.

## 8. Detailed Task Matrix

| ID  | Task                          | Wave | Owner | Blocking | Result                     |
| --- | ----------------------------- | ---- | ----- | -------- | -------------------------- |
| T01 | Create go.work                | 1    | Arch  | None     | Workspace active           |
| T02 | Init engine/go.mod            | 1    | Arch  | T01      | Independent build          |
| T03 | Filesystem move               | 1    | Arch  | T01      | New layout                 |
| T04 | Update imports                | 2    | Dev   | T03      | Builds green               |
| T05 | Add forwarders                | 2    | Dev   | T04      | Transitional compatibility |
| T06 | Strategy interfaces file      | 3    | Arch  | T04      | Stable extension points    |
| T07 | Internalize impl packages     | 3    | Arch  | T06      | Reduced surface            |
| T08 | API doc comments + tiers      | 3    | Docs  | T06      | Stability clarity          |
| T09 | CLI module skeleton           | 4    | Dev   | T02, RP1 | Basic binary               |
| T10 | CLI integration test          | 4    | QA    | T09      | Regression guard           |
| T11 | Metrics/health adapter wiring | 4    | Dev   | T09      | Observability usable       |
| T12 | Remove forwarders             | 5    | Arch  | T05      | Clean tree                 |
| T13 | Migration guide doc           | 5    | Docs  | T12      | User adoption              |
| T14 | Tag engine v0 baseline        | 6    | Maint | T12      | Versioned API              |
| T15 | README embedding example      | 6    | Docs  | T14      | Developer onboarding       |

### Root Purge Task Additions

| ID  | Task                                      | Wave | Owner | Blocking | Result                           |
| --- | ----------------------------------------- | ---- | ----- | -------- | -------------------------------- |
| RP1 | Move root `main.go`                       | 2.5  | Arch  | T04?\*   | No root executable               |
| RP2 | Root guard script/test                    | 2.5  | Dev   | RP1      | Prevent regression               |
| RP3 | Inventory root legacy dirs                | 2.5  | Arch  | RP1      | Disposition list                 |
| RP4 | Forward root imports → engine module path | 2.5  | Dev   | T04      | Single authoritative import path |
| RP5 | Archive/remove `cmd/scraper` & others     | 3.5  | Arch  | RP3      | Clean root tree                  |
| RP6 | Enforce no old import paths (test)        | 3.5  | QA    | RP4      | Early failure on drift           |
| RP7 | CI grep check (no root \*.go)             | 3.5  | Dev   | RP2      | Automated enforcement            |

\*If sequencing prefers, RP1 can occur immediately after T03 (filesystem move) before full import refactor completes.

## 9. Risk & Mitigation

| Risk                 | Description                                              | Likelihood | Impact            | Mitigation                                               |
| -------------------- | -------------------------------------------------------- | ---------- | ----------------- | -------------------------------------------------------- |
| Hidden cross-imports | Residual code outside engine still referencing old paths | Medium     | Build breaks      | Global grep + CI fail-fast                               |
| Forwarder drift      | Temporary aliases linger too long                        | Medium     | API confusion     | Deadline + lint rule (disallow old path)                 |
| Over-exposed APIs    | Forget to internalize helpers                            | High       | Future break cost | Wave 3 audit & doc coverage gate                         |
| CLI config sprawl    | Config duplication between CLI & engine                  | Medium     | Inconsistency     | Single `engine.Config` authoritative + small CLI overlay |
| Telemetry coupling   | CLI accidentally reimplements health/metrics logic       | Low        | Code duplication  | Provide adapter package or examples                      |
| Versioning churn     | Frequent API tweaks after tag                            | Medium     | Consumer friction | Delay tag until Wave 5 cleanup complete                  |

## 10. Acceptance Criteria

Engine Module:

- `go build ./...` inside `engine/` succeeds without referencing CLI dependencies.
- Public API documented (godoc) with stability tags.
- Internal implementation packages not importable externally (enforced by placement under `internal/`).

CLI Module:

- `ariadne crawl` command runs a crawl against test fixture and exits cleanly.
- Supports: seeds file, resume, config file, metrics endpoint flag.
- Emits periodic JSON snapshot (configurable interval) to stderr or log.

Migration:

- All original imports updated; forwarders removed by Wave 5 completion.
- CHANGELOG includes engine extraction and any breaking changes.
- Embedding example in root README compiles.

Quality Gates:

- Test suite passes across workspace (engine + cli) via `go work` driven run.
- Benchmark smoke (select one pipeline benchmark) still executes in engine module.
- Lint (if configured) passes; no TODO unaddressed in public exported API comments.

## 11. Tooling & Automation

Add Make targets (root):

- `make bootstrap` – create/update `go.work`.
- `make test-all` – run engine & cli tests.
- `make lint` – optional static analysis.

Add CI matrix (future):

- Job 1: `engine` module tests.
- Job 2: `cli` module tests.
- Job 3: Integration smoke (CLI invoking engine).

## 12. De-scoped / Deferred Items

- TUI / interactive progress dashboard (Phase 6/7).
- External trace exporter & advanced adaptive sampling logic.
- Plugin discovery registry (fetchers, processors) – design later.
- Multi-module for adapters (metrics/trace exporters) – revisit after initial split stability.

## 13. Migration Guide (Draft Outline)

Original Import Example:

```
import "ariadne/packages/engine"
```

New Import:

```
import "github.com/99souls/ariadne/engine"
```

Key Changes:

- Some subpackages now moved under `engine/internal/*` (no longer importable).
- Use `engine.New(cfg)` and strategy injection via `engine.NewWithStrategies` (stable name TBD).
- Metrics/health HTTP handlers now provided via adapter examples (`examples/observability`).

## 14. Open Decisions (To Settle Early Wave 1)

| Topic            | Options                             | Recommendation                    | Rationale                                         |
| ---------------- | ----------------------------------- | --------------------------------- | ------------------------------------------------- |
| Module path      | Temporary local vs canonical GitHub | Use canonical                     | Avoid churn for external adopters                 |
| Strategy naming  | Fetcher vs Crawler                  | Fetcher                           | Generalizes beyond HTML crawling                  |
| CLI module path  | `cli` vs `cmd/ariadne`              | `cli` module + `cmd/ariadne` main | Cleaner separation + room for future `cmd/` tools |
| Config layering  | Single struct vs layered            | Single authoritative struct       | Prevent drift                                     |
| Version baseline | v0.5.0 vs v0.1.0                    | v0.5.0                            | Reflect maturity while signaling pre-stable       |

## 15. Next Immediate Actions (Post-Approval Checklist)

1. Create `go.work` + initialize `engine/go.mod` (Wave 1)
2. Filesystem move & root import refactor (Waves 1–2)
3. Add forwarders + update tests (Wave 2)
4. API audit & internalization (Wave 3)
5. Bootstrap CLI module + minimal command (Wave 4)
6. Remove forwarders & tag baseline (Waves 5–6)

---

Prepared for review. Please annotate decisions in Section 14 or raise blocking concerns before Wave 1 execution.
