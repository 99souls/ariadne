# Ariadne

Structured, polite web content extraction: point it at seed URLs, it fans out (civilly), pulls pages, normalizes them, extracts the useful bits, and emits structured results you can drop into notes, search indices, or downstream pipelines.

This repository now follows an **Atomic Root Layout**: no root Go module, only curated submodules:

- `engine/` – Embeddable library (public API surface)
- `cli/` – User-facing command (`ariadne`)
- `tools/apireport/` – API surface reporting utility

See `ROOT_LAYOUT.md` for the invariant and allowed top-level entries.

## Quick Start

Clone and sync workspace modules:

```bash
git clone https://github.com/99souls/ariadne.git
cd ariadne
go work sync
```

Run a tiny crawl (stdout JSONL sink):

```bash
go run ./cli/cmd/ariadne --seeds https://example.com --snapshot-interval 5s
```

Example output line:

```json
{
  "url": "https://example.com/",
  "title": "Example Domain",
  "markdown_excerpt": "# Example Domain..."
}
```

Generate / refresh the API report:

```bash
make api-report
```

This invokes the `tools/apireport` module to regenerate `API_REPORT.md` and CI will fail if drift is detected.

Enable metrics (Prometheus) & health endpoints (experimental surface):

```bash
go run ./cli/cmd/ariadne -seeds https://example.com \
  -enable-metrics -metrics :9090 -health :9091 -snapshot-interval 0 &
sleep 1
curl -s http://localhost:9090/metrics | grep ariadne_build_info || echo "metric not found yet"
curl -s http://localhost:9091/healthz
```

Embedding the engine (minimal example):

```go
package main

import (
  "context"
  "log"
  "github.com/99souls/ariadne/engine"
)

func main() {
  cfg := engine.Defaults()
  eng, err := engine.New(cfg)
  if err != nil { log.Fatal(err) }
  defer eng.Stop()
  results, err := eng.Start(context.Background(), []string{"https://example.com"})
  if err != nil { log.Fatal(err) }
  for r := range results {
    _ = r // process result
  }
  snap := eng.Snapshot()
  _ = snap // inspect snapshot fields
}
```

See `md/telemetry-boundary.md` for telemetry stability notes.

## Core Flow (Simplified)

1. Seed ingestion
2. Crawler fetch (rate & domain pacing + robots handling planned)
3. Content extraction (HTML → minimal markdown projection)
4. Orchestration & enrichment (internal pipeline – now fully internalized)
5. Output emission (stdout JSONL; richer markdown / HTML variants in progress)

## Design Highlights

- Adaptive rate limiting (sliding window + token bucket hybrid) per domain
- Resource manager (seen URL set + spill-to-disk + checkpoint journal) enabling resume mode
- Internal pipeline stages with bounded concurrency + backpressure coordination
- Config layering (defaults → env → file(s) → flags) with normalization helpers
- Telemetry hooks: metrics, tracing, event bus, health snapshots (curated public exposure; internals private; see `md/telemetry-boundary.md` for evolving surface & stability)
- Asset & change strategies (experimental) to reduce redundant fetches

## Why Another Crawler?

Many scrapers either dump raw HTML (too low-level) or attempt full browser emulation (heavy, slow). Ariadne targets the middle: fast structural extraction + markdown fidelity suitable for knowledge bases and knowledge graph ingestion without headless overhead.

## Current Status / Honesty Notes

- Public surface still being pruned (Wave 3 upcoming)
- Some exported symbols will gain stability annotations (`API_STABILITY.md` defines tiers)
- Markdown extraction intentionally minimal (headings + body). Rich elements (lists, tables, code fences) in roadmap.
- PDF export deferred until markdown fidelity improves.

## Configuration Layers (Planned Shape)

| Layer    | Purpose                              |
| -------- | ------------------------------------ |
| defaults | sane crawl + rate limits             |
| env vars | container / ops overrides            |
| file(s)  | site / environment structured config |
| flags    | last-mile operational tweaks         |

Goal: zero mandatory config for a polite starter crawl; progressive opt‑in complexity.

## Roadmap (Trimmed)

- Wave 3: API pruning & stability annotations
- Enhanced markdown projection (lists, code fences, tables, alt text)
- Structured front‑matter (canonical URL, tags)
- Incremental recrawl / change detection mode
- PDF / alternate output adapters

## Contributing

Lightweight expectations:

- Keep PRs focused & small
- Add/adjust nearest test with every change
- Avoid expanding public API without stability rationale & tests
- Respect Atomic Root Layout (no new top-level code dirs)

## License

See `LICENSE` (permissive). Open an issue for clarifications.

## API Stability & Reporting

`API_STABILITY.md` documents tiers. `make api-report` regenerates `API_REPORT.md`; CI enforces no unintended drift. Internal packages (`engine/internal/*`) are not part of the contract.

## Guiding Principles

Useful > flashy. Explicit over magic. Small, well-documented surfaces > sprawling implicit behavior.

## Developer Hooks (Optional DX Boost)

This repo ships a `.pre-commit-config.yaml` that runs (fast):

- Formatting / whitespace hygiene
- `golangci-lint` across all modules
- `go test -short` across modules
- API report drift check (regenerates in a temp location and diffs)
- `go mod tidy` drift detector (fails fast if a module file would change)

Install the hooks (after installing `pre-commit`):

```bash
make hooks
```

Install `pre-commit` if missing:

```bash
pipx install pre-commit        # or: pip install --user pre-commit
# macOS (brew): brew install pre-commit
```

Skip hooks for an emergency commit:

```bash
git commit -m "wip" --no-verify
```

Regenerate API report manually when you intend a surface change:

```bash
make api-report
```

Hooks intentionally use `-short` tests to keep latency low; CI still runs the full matrix including race + full tests.

## Live Test Site Usage (Integration Surface)

The repository includes a deterministic Bun + React live site (`tools/test-site`) used by integration tests to exercise real HTTP crawling:

- Link discovery & depth limiting
- Robots allow / deny modes
- Asset success + intentional 404
- Slow endpoint latency impact
- Process reuse (no respawn when `TESTSITE_REUSE=1`)
- Advanced content fixture (admonitions, code fences, footnotes, table)
- Dark mode variant de-dup (theme query param canonicalized)
- Large asset (~200KB) throughput (ensures no crawl stall)
- Latency distribution baseline for `/api/slow`
- URL normalization (cosmetic params: theme + utm\_\* stripped) – see `engine/README.md` section “URL Normalization & Variant De-duplication”

### Make Targets

```
make testsite-dev         # Run dev server (foreground; CTRL+C to stop)
make testsite-check       # Lint + TypeScript type checking for the test site
make integ-live           # Run only LiveSite* integration tests (reuses server)
make testsite-snapshots   # Regenerate normalized HTML snapshot goldens
```

### Environment Variables

| Variable           | Purpose                                 | Default |
| ------------------ | --------------------------------------- | ------- |
| `TESTSITE_PORT`    | Fixed port (otherwise ephemeral)        | 5173    |
| `TESTSITE_REUSE`   | If `1`, keep process alive across tests | unset   |
| `TESTSITE_ROBOTS`  | `allow` or `deny` robots mode           | allow   |
| `UPDATE_SNAPSHOTS` | If `1`, rewrite snapshot golden files   | unset   |

### Running Integration Tests

Run the focused suite:

```bash
make integ-live
```

This sets `TESTSITE_REUSE=1` so the Bun process starts once and all `LiveSite*` tests reuse it, reducing latency.

Individual test example:

```bash
go test ./engine/internal/business/crawler -run TestLiveSiteBrokenAsset -count=1 -v
```

### Snapshot Workflow

The snapshot test (`TestGenerateSnapshots`) fetches a page, normalizes HTML (removes volatile attributes/whitespace and cosmetic query params), and compares it to a golden in `engine/internal/testutil/testsite/testdata/snapshots/`.

1. Edit site content intentionally.
2. Run test and observe failure (drift output shows first differing line).
3. Accept change:

```bash
UPDATE_SNAPSHOTS=1 go test ./engine/internal/testutil/testsite -run TestGenerateSnapshots -count=1
```

4. Commit updated golden file.

See the detailed acceptance criteria & reviewer checklist in `md/snapshot-acceptance.md`.

Flake Detection (optional, local):

Use the flake detector to run the live site tests repeatedly and surface instability (portable on macOS & Linux; set ITER env var or first arg):

```
make flake-live            # default 10 iterations
ITER=25 make flake-live    # custom iteration count (or: make flake-live ITER=25)
```

Outputs pass/fail counts plus basic duration stats (min / max / mean / p95). Any failure causes a non‑zero exit.

### Current Live Integration Tests (Sampling)

| Test                               | Behavior Exercised                                                 |
| ---------------------------------- | ------------------------------------------------------------------ |
| `TestLiveSiteDiscovery`            | Multi-page link discovery (allow robots)                           |
| `TestLiveSiteRobotsDeny`           | Deny-all robots gating (no pages fetched)                          |
| `TestLiveSiteDepthLimit`           | Path segment depth limiting (excludes deep leaf)                   |
| `TestLiveSiteBrokenAsset`          | Broken asset surfaced with >=400 status (non-blocking)             |
| `TestLiveSiteSlowEndpoint`         | Slow `/api/slow` endpoint fetched without excessive wall time      |
| `TestLiveSiteReuseSingleInstance`  | Bun process reuse across tests                                     |
| `TestLiveSiteDarkModeDeDup`        | Dark mode variant collapses to canonical URL                       |
| `TestLiveSiteLargeAssetThroughput` | Large binary asset fetch does not degrade overall crawl throughput |
| `TestLiveSiteLatencyDistribution`  | Latency envelope maintained (distribution within expected bounds)  |
| `TestLiveSiteSearchIndexIgnored`   | `/api/search.json` ignored (non-page JSON endpoint)                |

### Adding New Assertions

Prefer adding new behaviors to the live site (e.g., footnotes, dark mode, large assets) then writing a `LiveSite*` test. Keep runtime <1s per test to maintain fast feedback.

### Determinism Guidelines

- No timestamps or random content in pages.
- Keep latency injection endpoints bounded (current `/api/slow` 400–600ms).
- Normalize volatile HTML (ids, data attributes) in snapshot tests.
- Strip cosmetic query parameters (`theme`, `utm_*`) so variants do not inflate result sets.
