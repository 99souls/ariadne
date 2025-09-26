# Engine Migration Notes

Status: Draft (P5 in progress)

## Goal
Migrate all production entrypoints (currently `main.go`) to rely exclusively on the `packages/engine` facade, avoiding direct imports of `internal/*` packages.

## Rationale
- Centralize lifecycle (Start/Stop)
- Provide unified snapshot + future telemetry hook
- Prepare for packaging / semver boundary

## Required Steps
1. Replace legacy placeholder banner CLI with facade-driven execution (DONE).
2. Add minimal flags: seeds, seed-file, resume, checkpoint, snapshot-interval (DONE).
3. Stream results as JSON lines (DONE).
4. Periodically emit snapshot to stderr (DONE).
5. Introduce deprecation notice around `pipeline.NewPipeline` (DONE).
6. Add API stability guide (DONE). 
7. Enforce absence of `internal/*` imports in root module `main.go` (implicit; grep check recommended in CI later).

## Post-Migration Follow Ups
- Add richer output formatting (table / summary) once telemetry narrative defined.
- Provide `--snapshot-json` option to write snapshots to rotating file.
- Add structured logging mode toggled by flag.

## Flags Reference
| Flag | Description | Default |
| ---- | ----------- | ------- |
| `-seeds` | Comma-separated seeds | "" |
| `-seed-file` | File with one URL per line | "" |
| `-resume` | Enable resume filtering | false |
| `-checkpoint` | Path to checkpoint log | `checkpoint.log` |
| `-snapshot-interval` | Periodic snapshot interval (0 disables) | 10s |
| `-version` | Print version information | false |

## Compatibility
Existing scripts relying on prior banner output should adjust to consume JSON lines (each `CrawlResult`). A future `--legacy` mode can reintroduce banner if needed.

## Known Limitations (Current)
- No graceful interrupt handling (Ctrl+C) customizing yet beyond context cancellation (to add signal handling later).
- Snapshot does not include domain top-N limiter stats (deferred).
- Resume only filters initial seeds; mid-run restarts don't reconstruct in-flight state beyond checkpointed URLs.

---
These notes will promote to formal `MIGRATION.md` after P7 documentation pass.
