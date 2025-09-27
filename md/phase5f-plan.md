# Phase 5F Plan – Engine Module Extraction & CLI Enablement

Status: In Progress (post unplanned big-bang merge of engine-migration)
Date: 2025-09-27 (updated)
Owner: Architecture / Core Engine

## 1. Executive Summary

Phase 5F elevates `engine` to a **standalone, minimally-coupled Go module** and introduces a first-class **CLI layer** that consumes only its public API. This establishes a clean contract boundary before Phase 6 (CLI polish & UX), reduces coupling, simplifies dependency surfaces, and paves the way for future multi-surface delivery (API server, TUI, hosted service).

Update (No Backwards Compatibility Policy): We explicitly do NOT preserve backwards compatibility with the pre-extraction import paths (`github.com/99souls/ariadne/engine`). A hard cut (“lift & shift”) is acceptable. This removes the need for forwarders, staged deprecation shims, or migration guards beyond ensuring the legacy tree is fully deleted.

Primary Outcomes:

- `engine/` is the only authoritative engine implementation (legacy copy removed outright).
- Root repository operates as a multi-module workspace via `go.work`.
- A new `cli/` (or `cmd/ariadne/`) module provides a user-facing entrypoint with configuration, metrics/health endpoints wiring, and graceful lifecycle.
- No compatibility layer: old import paths break immediately once legacy tree removed.
- Documentation & stability annotations reflect a cleaned, curated API surface.
- Atomic Root Layout: The repository root SHALL ultimately contain only two code-bearing directories (`engine/`, `cli/`). All previous root code directories (`internal/`, `pkg/`, `cmd/`, `packages/`, ad-hoc test utility trees) will be removed, migrated, or archived under module-scoped `internal/` folders or `archive/` with `//go:build ignore`.

### Atomic Root Layout Objective (Authoritative Statement)

Goal: Achieve a minimal, unambiguous top-level structure in which every production Go package lives inside exactly one of:

1. `engine/` (the library / embedding surface)
2. `cli/` (the binary surface depending ONLY on exported `engine` API)

Permitted additional root entries: build/workspace manifests (`go.work`, `Makefile`), project metadata (`LICENSE`, `README.md`, `CHANGELOG.md`, `API_STABILITY.md`), architectural documentation (`md/`), and automation assets. No other executable Go sources or code directories remain at root.

Non-compliant directories slated for elimination or migration: `internal/`, `pkg/`, `cmd/`, `packages/`, `test/` (root-level test utilities to move under `engine/internal/test` or `cli/internal/test`), and any lingering historical scaffolds.

## 2. Problem Statement

Current state (post-Phase 5E + ad-hoc full engine duplication merge):

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
7. Hard Cut Simplicity: Remove legacy tree promptly; accept breakage for any unpublished/experimental consumers.

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

### Wave 1 – Module Skeleton (Actual vs Plan)

Planned: Minimal move + stub/forwarders.

Actual: Entire engine tree copied under `engine/` while original `packages/engine/` left intact (full duplication of business logic + tests). `engine/go.mod` created (canonical path `github.com/99souls/ariadne/engine`). `go.work` present. Import normalization incomplete (many files in `engine/` still import `github.com/99souls/ariadne/engine/...`). No stub README or forwarder-only reduction yet.

Implications:

- Faster availability of isolated module path.
- Elevated drift risk (two authoritative copies) until deduplication.
- Larger immediate surface for API pruning work.

Remediation (Revised Wave 2):

1. Authoritative source: `engine/` (DONE).
2. Freeze notice added (DONE).
3. Normalize imports inside `engine/` (DONE).
4. DELETE `packages/engine` entirely (no forwarders) once tests in `engine/` are green and root does not import legacy path.
5. Remove duplicate tests (keep only `engine/`).

### Wave 2 – Hard Dedup & Legacy Removal

Objective: Eliminate the duplicated legacy tree (`packages/engine`) immediately (no forwarders) after confirming `engine/` tests pass.

Tasks:

- Delete `packages/engine` directory (implementation + tests).
- Remove any residual references (grep for `packages/engine`).
- Run full workspace tests to ensure no hidden dependency remained.
- Update CHANGELOG: note breaking removal of old import path.
- Update README (import examples already using new path – verify).

Exit Criteria: No `packages/engine` directory; `grep -R "packages/engine"` (excluding plan/docs) returns zero occurrences in `.go` sources.

### Wave 2.5 – Root Purge (Part 1) (Unchanged Intent, Executed Earlier if Convenient)

Purpose: Eliminate root runtime entrypoint to prevent dual-path drift.

Tasks:

- Move `main.go` → `cli/cmd/ariadne/main.go` (temporary location until CLI module Wave 4 is created; may stage under root then finalize in Wave 4 if sequencing requires).
- Add guard script / test: fail build if any non-test `.go` file exists at repo root (except tooling stubs).
- Document change in CHANGELOG (Unreleased: "root executable moved to cli module").
- Inventory remaining root `cmd/` directories; flag for Wave 3.5 disposition.

Exit Criteria: Root contains no executable Go files; tests still green.

### Wave 3 – API Surface Audit & Pruning

Status: Candidate list drafted (`engine/API_PRUNING_CANDIDATES.md`).

Upcoming Tasks:

1. Review & approve candidate actions (KEEP / INT / TAG / REMOVE).
2. Create `internal/` subpackages and move INT items.
3. Introduce consolidated strategy interfaces file (`strategies.go`) or re-export strategy package (decision pending).
4. Add doc comments & stability annotations (Experimental tags) to all remaining exported symbols.
5. Remove deprecated alias types (e.g., `FetchedPage`).
6. Update CHANGELOG (Unreleased > Changed) summarizing pruning adjustments.
7. Add enforcement test ensuring no forbidden internal package imports from CLI / external modules.

### Wave 3.5 – Root Purge (Part 2)

Purpose: Remove lingering legacy command code & shadow copies of business logic.

Tasks:

- Migrate or archive `cmd/scraper` & `cmd/test-phase1`.
  - If kept for historical reference: move to `archive/` with `//go:build ignore`.
- Remove / migrate any root `internal/*` duplicates now superseded by engine module.
- Add enforcement test: ensure no imports reference old paths (`github.com/99souls/ariadne/engine`).
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

### Wave 5 – Versioning & Tag Baseline (Renumbered; Forwarder Removal Wave Eliminated)

Tasks:

- Tag engine module `v0.5.0` (example) once CLI uses only curated API.
- Add `API_STABILITY.md` section for engine module.
- Update root README with embedding example.

## 8. Detailed Task Matrix

| ID  | Task                           | Wave | Owner | Blocking | Result                   |
| --- | ------------------------------ | ---- | ----- | -------- | ------------------------ |
| T01 | Create go.work                 | 1    | Arch  | None     | Workspace active         |
| T02 | Init engine/go.mod             | 1    | Arch  | T01      | Independent build        |
| T03 | Filesystem move                | 1    | Arch  | T01      | New layout               |
| T04 | Update imports                 | 2    | Dev   | T03      | Builds green             |
| T05 | (Removed) forwarders (N/A)     | -    | -     | -        | Not applicable           |
| T06 | Strategy interfaces file       | 3    | Arch  | T04      | Stable extension points  |
| T07 | Internalize impl packages      | 3    | Arch  | T06      | Reduced surface          |
| T08 | API doc comments + tiers       | 3    | Docs  | T06      | Stability clarity        |
| T09 | CLI module skeleton            | 4    | Dev   | T02, RP1 | Basic binary             |
| T10 | CLI integration test           | 4    | QA    | T09      | Regression guard         |
| T11 | Metrics/health adapter wiring  | 4    | Dev   | T09      | Observability usable     |
| T12 | (Removed) forwarder removal    | -    | -     | -        | Not applicable           |
| T13 | Migration guide doc (hard cut) | 5    | Docs  | T07      | User adoption (new path) |
| T14 | Tag engine v0 baseline         | 5    | Maint | T07      | Versioned API            |
| T15 | README embedding example       | 5    | Docs  | T14      | Developer onboarding     |

### Root Purge Task Additions

| ID   | Task                                                        | Wave | Owner | Blocking | Result                           |
| ---- | ----------------------------------------------------------- | ---- | ----- | -------- | -------------------------------- |
| RP1  | Move root `main.go`                                         | 2.5  | Arch  | T04?\*   | No root executable               |
| RP2  | Root guard script/test                                      | 2.5  | Dev   | RP1      | Prevent regression               |
| RP3  | Inventory root legacy dirs                                  | 2.5  | Arch  | RP1      | Disposition list                 |
| RP4  | (Dropped) Forward root imports (imports already normalized) | -    | -     | -        | Superseded                       |
| RP5  | Archive/remove `cmd/scraper` & others                       | 3.5  | Arch  | RP3      | Clean root tree                  |
| RP6  | Migrate / remove root `internal/` packages                  | 3.5  | Arch  | RP3      | Impl moved under engine/internal |
| RP7  | Remove / alias `pkg/` (models & helpers)                    | 3.5  | Arch  | RP6      | DONE: imports rewired; legacy pkg/models removed (residual stub files in progress removal) |
| RP8  | Remove `packages/` adapters (relocate if still needed)      | 3.5  | Arch  | RP6      | Single engine surface            |
| RP9  | Consolidate test utilities under module-scoped internal     | 3.5  | QA    | RP6      | No root test helpers             |
| RP10 | Enforce no old import paths (test)                          | 3.5  | QA    | RP6      | Early failure on drift           |
| RP11 | CI grep check (no root \*.go) & directory whitelist         | 3.5  | Dev   | RP2      | Automated enforcement            |
| RP12 | Add ROOT_LAYOUT.md (doc) & update plan                      | 3.5  | Docs  | RP5-RP8  | Stable documentation             |

\*If sequencing prefers, RP1 can occur immediately after T03 (filesystem move) before full import refactor completes.

## 9. Risk & Mitigation

| Risk                 | Description                                              | Likelihood      | Impact            | Mitigation                                               |
| -------------------- | -------------------------------------------------------- | --------------- | ----------------- | -------------------------------------------------------- |
| Hidden cross-imports | Residual code outside engine still referencing old paths | Medium          | Build breaks      | Global grep + CI fail-fast                               |
| Hard cut breakage    | External (unpublished) consumers break on removal        | High (accepted) | Low (internal)    | Document breaking change in CHANGELOG & README           |
| Over-exposed APIs    | Forget to internalize helpers                            | High            | Future break cost | Wave 3 audit & doc coverage gate                         |
| CLI config sprawl    | Config duplication between CLI & engine                  | Medium          | Inconsistency     | Single `engine.Config` authoritative + small CLI overlay |
| Telemetry coupling   | CLI accidentally reimplements health/metrics logic       | Low             | Code duplication  | Provide adapter package or examples                      |
| Versioning churn     | Frequent API tweaks after tag                            | Medium          | Consumer friction | Delay tag until Wave 5 cleanup complete                  |

## 10. Acceptance Criteria

Engine Module:

- `go build ./...` inside `engine/` succeeds without referencing CLI dependencies.
- Public API documented (godoc) with stability tags (Experimental vs future Stable).
- Implementation detail packages moved under `engine/internal/...` and not imported by CLI (enforced by tests & grep guard).

CLI Module:

- `ariadne crawl` command runs a crawl against test fixture and exits cleanly.
- Supports: seeds file, resume, config file, metrics endpoint flag.
- Emits periodic JSON snapshot (configurable interval) to stderr or log.
- Imports ONLY `github.com/99souls/ariadne/engine` (and its public packages) – no `engine/internal/` path usage.

Atomic Root Layout:

- Root contains no `internal/`, `pkg/`, `cmd/`, `packages/`, or other legacy code directories.
- Only code-bearing directories at root: `engine/` and `cli/`.
- Guard tests fail if any non-test `.go` file appears at root or if disallowed directories reappear.
- CI script enforces directory whitelist and zero matches for forbidden import paths.

Migration:

- All legacy directories removed or archived with `//go:build ignore` (if historically valuable) under `archive/` (non-built).
- Root aliases (e.g., `pkg/models`) eliminated; external users must import `github.com/99souls/ariadne/engine/models`.
- CHANGELOG captures atomic root milestone and breaking removals.
- Embedding example in README updated to canonical `engine` imports only.

Quality Gates:

- Test suite passes across workspace (engine + cli) via `go work` run.
- Benchmarks (at least one pipeline benchmark) still execute in engine module after internalization.
- Lint (if configured) passes; no undocumented exported symbols.
- Guard tests: (1) no root main, (2) no forbidden directories, (3) no internal imports in CLI.

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

## 13. Migration Guide (Hard Cut Outline)

Original Import Example:

```
import "github.com/99souls/ariadne/engine"
```

New Import:

```
import "github.com/99souls/ariadne/engine"
```

Key Changes (Breaking):

- Old path `github.com/99souls/ariadne/engine` no longer exists (directory deleted).
- Some subpackages may move under `engine/internal/*` (not importable) during pruning.
- Use `engine.New(cfg)` (or future `engine.NewWithStrategies`) for construction.
- Metrics/health handlers will be provided outside core (adapters/examples).

## 14. Open Decisions (To Settle Early Wave 1)

| Topic            | Options                             | Recommendation                    | Rationale                                         |
| ---------------- | ----------------------------------- | --------------------------------- | ------------------------------------------------- |
| Module path      | Temporary local vs canonical GitHub | Use canonical                     | Avoid churn for external adopters                 |
| Strategy naming  | Fetcher vs Crawler                  | Fetcher                           | Generalizes beyond HTML crawling                  |
| CLI module path  | `cli` vs `cmd/ariadne`              | `cli` module + `cmd/ariadne` main | Cleaner separation + room for future `cmd/` tools |
| Config layering  | Single struct vs layered            | Single authoritative struct       | Prevent drift                                     |
| Version baseline | v0.5.0 vs v0.1.0                    | v0.5.0                            | Reflect maturity while signaling pre-stable       |

## 15. Next Immediate Actions (Revised – Hard Cut)

Status Legend: (DONE) already completed on branch.

1. (DONE) Normalize `engine/` internal imports (legacy references = 0).
2. (DONE) Add legacy freeze notice (README-LEGACY.md) – will be removed alongside deletion.
3. (DONE) Delete `packages/engine` tree entirely (implementation + tests).
4. Run full workspace tests (`go test ./...`) to confirm no hidden dependency.
5. Relocate root `main.go` (RP1) to future `cli/` scaffold (start purge early).
6. Create minimal `cli/go.mod` + placeholder command invoking engine.
7. (DONE) Draft API pruning candidate list (prep for Wave 3 internalization) and mark likely `internal/` moves.
8. Rewire legacy `ariadne/pkg/models` imports to `github.com/99souls/ariadne/engine/models` (IN PROGRESS – code rewired, alias package pending deletion).
9. Delete `pkg/models` directory after confirming no external dependency (PENDING).
8. Update CHANGELOG + README (Breaking: legacy path removed).
9. Remove `legacy-imports` target once deletion complete (optional) OR repurpose to assert zero forever.

Gate to enter Wave 3: Legacy tree removed; root main relocated; CLI skeleton present; pruning list drafted; legacy pkg/models aliases removed.

---

Prepared for review. Please annotate decisions in Section 14 or raise blocking concerns before Wave 1 execution.
