# Package Architecture Analysis & Proposed Evolution

Date: 2025-09-26

## Executive Summary

The `engine` package should be the authoritative core orchestration layer that encapsulates crawl pipeline coordination, adaptive rate limiting, resource lifecycle, and future extensibility points (fetch, parse, process, output). The current internal packages (`internal/crawler`, placeholder `internal/output`) represent functional domains that can either (a) be promoted into modular, pluggable subsystems under a cohesive `engine` namespace, or (b) split into a higher-level “integration” layer when they provide environment-/protocol-specific behavior rather than core orchestration logic. After analysis, the crawler is a candidate for a dedicated subpackage (e.g. `packages/engine/crawler`) provided it is expressed as an interface-driven fetch/discovery module. The output concern is orthogonal and should become a strategy-oriented plugin point (e.g. `packages/engine/output`) rather than core—allowing multiple output sinks (memory, file, streaming, queue) without bloating the pipeline orchestration code.

## Current State (Post Pipeline Migration)

Core engine components (migrated):

- `packages/engine/models`: shared domain types (Pages, CrawlResult, RateLimitConfig, Errors).
- `packages/engine/ratelimit`: adaptive domain-aware limiter with circuit breaker.
- `packages/engine/resources`: cache + spill + checkpoint + in-flight coordination.
- (internal) `engine/internal/pipeline`: multi-stage processing (discovery → extraction → processing → output) with metrics, retries, backpressure. No longer public; accessed only via facade.
- `packages/engine/engine`: facade aggregating pipeline + limiter + resources + snapshot.

Remaining internally-scoped domain logic:

- `internal/crawler`: A Colly-based fetch + link discovery component performing HTML extraction & link normalization.
- `internal/processor` / `internal/assets` (not yet migrated, assumed to handle content enrichment, asset handling—needs evaluation for overlap with pipeline's placeholder logic).
- `internal/output`: presently empty / placeholder.

## Architectural Principles

1. **Single Source of Truth for Orchestration**: The `engine` package owns lifecycle start/stop, scheduling, concurrency wiring, metrics composition.
2. **Plugin Boundaries over Feature Flags**: Concrete behaviors (fetching, parsing, output sinks) should be injected via interfaces to avoid monolith growth inside pipeline.
3. **Composable Resource & Rate Controls**: Limiter + resource manager are “horizontal” services consumed by plug‑in stages but not coupled to specific content semantics.
4. **Deterministic Shut Down & Observability**: Every pluggable stage must propagate context cancellation and emit structured metrics.
5. **Separation of Concerns**: Distinguish between core (engine) and adaptation layers (protocol-specific crawler, output adapters).

## Crawler: Core or Adapter?

The crawler currently:

- Manages HTTP requests via Colly.
- Performs HTML extraction + metadata enrichment.
- Discovers and normalizes links (scope filtering & dedup).
- Streams results (pages/errors) through an internal channel.

Assessment:

- The crawling action (HTTP fetch + link discovery) is foundational to the pipeline’s “discovery” and “extraction” semantics.
- However, tying the engine directly to Colly constrains future transport strategies (headless browser, API graph traversals, sitemap ingestion, offline HTML sets).

Conclusion:
Treat the crawler as a pluggable “Fetcher + Discovery” provider:

- Define `Fetcher` interface (input URL → (fetched content bytes + metadata + outbound links)).
- Define `LinkExtractor` (content → normalized link set) if the split is beneficial.
- Provide a Colly-based implementation living in `packages/adapters/crawler/colly` OR `packages/engine/crawler/colly` depending on whether we want “all engine code under one tree” vs a clearer adapter boundary.

Recommendation: Place in `packages/engine/crawler` initially for simplicity, but structure as `crawler.Interface` + `crawler.Colly` implementation so future alternative fetchers can coexist. The pipeline’s current synthetic extraction functions will be replaced by calls into a `Fetcher` implementation.

## Output: Core or Strategy?

Output responsibilities (anticipated):

- Persist or emit `CrawlResult` to destinations (filesystem, stdout, JSONL stream, message queue, database, in-memory aggregator).
- Apply formatting (HTML → Markdown, metadata shaping) if not already done upstream.

Assessment:

- Output actions do not influence upstream scheduling or control flow other than backpressure (which is already mediated by bounded channels).
- Variation in output targets is a user-driven concern; should be decoupled via interfaces.

Conclusion:
Output belongs as pluggable strategies, not core orchestration. Provide an `OutputSink` interface consumed by the pipeline’s final stage (currently “output” stage). Keep only the dispatch loop in core; concrete sinks reside in subpackages.

## Proposed Target Package Topology

```
packages/
	engine/
		engine.go          (facade)
		config.go          (composition config)
		models/            (domain types)
		ratelimit/         (adaptive limiter)
		resources/         (cache/spill/checkpoint)
		internal/pipeline/ (stage orchestration, metrics, retries) [internal only]
		crawler/           (interfaces + default Colly implementation)
			colly/
				colly_crawler.go
		output/            (interfaces for output sinks)
			stdout/
				stdout_sink.go
			jsonl/
				jsonl_sink.go
			memory/
				memory_sink.go
		processor/         (optional: transformation / enrichment / analysis APIs)
			markdown/
				html_to_markdown.go
			nlp/
				...
	adapters/ (optional future extraction of non-core implementations)
```

## Interfaces Sketch

```go
// Crawler / Fetch abstraction
type Fetcher interface {
		Fetch(ctx context.Context, u string) (FetchedPage, error)
}
type FetchedPage struct {
		URL       *url.URL
		Content   []byte
		MediaType string
		Status    int
		Links     []*url.URL
		Metadata  map[string]string
}

// Output strategy
type OutputSink interface {
		Write(result *models.CrawlResult) error
		Flush() error
		Close() error
}

// Processor pipeline unit (optional future)
type Processor interface {
		Process(ctx context.Context, page *models.Page) (*models.Page, error)
}
```

## Migration Roadmap (Post-Pipeline)

Phase E1: Introduce interfaces (Fetcher, OutputSink) + minimal Colly fetcher + stdout sink.
Phase E2: Replace synthetic extraction & processing code with Fetcher + Processor invocation inside `pipeline`.
Phase E3: Add output dispatcher stage writing to a configurable `[]OutputSink` set.
Phase E4: Move markdown conversion & enrichment into Processor subpackages.
Phase E5: Observability expansion (structured metrics per interface, tracing).
Phase E6: Adapter separation (optionally move non-core sinks/fetchers to `adapters/`).

## Benefits of This Architecture

| Aspect                 | Benefit                                                                                     |
| ---------------------- | ------------------------------------------------------------------------------------------- |
| Modularity             | Swap fetcher (Colly → headless) with zero pipeline code changes.                            |
| Testability            | Stage unit tests can mock interfaces without channel gymnastics.                            |
| Extensibility          | New sinks (Kafka, S3) added without editing core engine.                                    |
| Performance Tuning     | Benchmark each interface implementation independently.                                      |
| Separation of Concerns | Core engine focuses on orchestration & flow control.                                        |
| Backpressure Control   | Bounded channels remain centralized; implementations remain pull-based via engine contract. |

## Risks / Mitigations

| Risk                                 | Mitigation                                                                                            |
| ------------------------------------ | ----------------------------------------------------------------------------------------------------- |
| Interface drift / leaky abstractions | Version interfaces behind clear semantic guarantees; keep FetchedPage minimal.                        |
| Over-engineering early               | Start with single Colly + stdout implementations; add others only when needed.                        |
| Performance overhead of indirection  | In Go, interface dispatch minimal; hot paths can optimize later (e.g., reduce allocations in Fetch).  |
| Complex configuration layering       | Provide facade-level config sections: EngineConfig{ Fetcher: crawler.Config, Output: output.Config }. |
| Resource manager coupling            | Keep resource manager injection explicit; let fetcher optionally leverage it (caching raw responses). |

## Concrete Next Steps

1. Define `Fetcher` & `OutputSink` interfaces under `packages/engine/crawler` and `packages/engine/output`.
2. Wrap existing Colly logic into `crawler/colly` implementing `Fetcher` (retain existing link normalization logic).
3. Replace pipeline `extractContent` simulation with fetcher invocation (produce `models.Page`).
4. Add configurable sinks array to pipeline config; implement stdout sink.
5. Add tests: fetcher stub test, sink error propagation test, pipeline integration with sink flush.
6. Update engine facade config to accept fetcher & sinks (with defaults autowired if nil).
7. Document new extension points (README updates + examples).

## Open Questions

1. Should link discovery remain inside fetcher or become a separate stage? (Recommendation: keep in fetch domain initially; refactor only if complexity grows.)
2. Do we need streaming partial results before full page processing? (If yes, introduce event bus pattern, otherwise keep simple final results.)
3. Should rate limiter operate at fetcher level (domain) only, or also at asset subfetch level? (Future: multi-tier limit—leave hook in Fetcher API for nested acquisitions.)

## Conclusion

The crawler’s responsibilities align with core engine concerns only at the abstraction level (fetch + discover). Its concrete Colly implementation is _not_ core—it’s a pluggable adapter. The output layer is explicitly a strategy set and should live as interfaces plus implementations, not embedded logic in the pipeline. By codifying these boundaries now, subsequent feature work (headless crawling, new output formats) won’t force disruptive refactors of orchestration code. Proceed with interface introduction (Phase E1) to lock in extension seams while the codebase is still relatively small.
