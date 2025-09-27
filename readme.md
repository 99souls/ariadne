# ariadne

a web scraper of sorts

## what it actually does (right now)

Think: point it at some seed URLs, it fans out (politely), pulls pages, normalizes them, extracts the useful bits, and spits out structured markdown (and later other formats) you can drop into notes, docs, or pipelines.

Core flow (simplified):

1. seeds in
2. crawler fetches (respecting domain pacing + robots if configured)
3. processor extracts title + main content (minimal HTML -> MD pass)
4. pipeline assembles results + enrichment hooks
5. output sinks write (stdout JSONL now; richer markdown / html variants wired; pdf planned)

## key bits under the hood

- polite rate limiting: sliding window + token bucket mash so we don't hammer domains
- resource manager: tracks seen URLs, spill-to-disk when memory pressure grows, handles resume checkpoints
- pipeline stages: fetch -> extract -> enhance -> assemble -> emit (each with bounded concurrency + backpressure)
- output layer: simple sinks now (stdout / markdown compiler), structured so custom sinks are trivial
- config layering: defaults -> env -> file(s) -> inline overrides (merging logic already in place, more hardening tests coming)
- telemetry hooks: metrics/tracing scaffolding exists (basic events + health snapshots) — not overexposed yet
- asset strategy: experimental logic for deciding “is this worth recrawling / enriching” (still being iterated)

## why it exists

Most generic scrapers either: a) dump raw HTML, b) get blocked fast, or c) try to be browsers. This aims for a narrower middle lane: fast structural extraction + markdown quality suitable for knowledge bases, without the overhead of headless everything.

## current status / honesty notes

- duplication: there are legacy paths sitting next to the new `engine/` module while migration settles
- api stability: not promised yet — names may churn while we prune surface area
- content extraction: intentionally minimal right now (headline + body). Rich semantics (tables, code fences, nav trimming) queued.
- pdf: placeholder goal; not wired until markdown fidelity is tightened

## quick taste

Run from repo root (basic stdout sink emitting JSONL of crawl results):

```
go run . --seeds https://example.com --limit 25
```

You’ll get lines like:

```json
{
  "url": "https://example.com/",
  "title": "Example Domain",
  "markdown_excerpt": "# Example Domain..."
}
```

Pipe to a file and you can post-process / pick pages to keep.

## config sketch (subject to change)

| layer    | purpose                             |
| -------- | ----------------------------------- |
| defaults | sane crawl + rate limits            |
| env vars | ops overrides in containers         |
| yaml/dir | structured site / environment rules |
| flags    | last-mile tweak                     |

Goal: zero required config for a “tiny polite crawl”, progressive enhancement for bigger controlled runs.

## roadmap (trimmed)

- finish import path cleanup + remove duplicated legacy tree
- richer markdown extraction: lists, code blocks, tables, image alt text
- structured front‑matter (title / canonical / tags)
- pdf (wkhtmltopdf or headless patch — TBD after markdown pass quality)
- advanced filtering / inclusion policies (path patterns, content heuristics)
- smarter change detection + incremental update mode

## contributing (lightweight guidance, not corporate)

Right now the focus is rapid internal iteration. If you do poke at it:

- keep PRs small
- add / update the nearest test; no “later” bucket
- don’t expand the public surface unless there’s a test proving why

## license

See `LICENSE` in the repo root. Standard permissive. If something feels ambiguous, open a brief issue and we’ll tighten wording.

## vibe check

Goal is usefulness > flash. If something feels over‑engineered, it probably is and should be simplified. If something feels too magic, we likely need one more explicit knob or a doc note.
