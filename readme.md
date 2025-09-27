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
go run ./cli/cmd/ariadne --seeds https://example.com --limit 25
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
- Telemetry hooks: metrics, tracing, event bus, health snapshots (curated public exposure; internals private)
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
