# Engine API Pruning Candidates (Wave 3 Draft)

Status: Draft – for review before internalization moves.

Objective: Reduce public surface to curated, documented contracts. Items below are proposed to be internalized, aliased, renamed, or stability-tagged.

## Legend

- KEEP: Remains exported (core API)
- INT: Move to internal/ (implementation detail)
- TAG: Keep exported but add stability annotation (Experimental / Internal)
- RENAME: Rename for clarity (with breaking change accepted pre-v1)
- MERGE: Collapse into another type or file

## 1. Facade Package (`engine`)

| Symbol                      | Action | Notes                                                               |
| --------------------------- | ------ | ------------------------------------------------------------------- |
| Engine                      | KEEP   | Core lifecycle facade                                               |
| New                         | KEEP   | Primary constructor                                                 |
| Option                      | INT    | Only used for future extensibility; can re-expose selectively later |
| Config.\* (all fields)      | TAG    | Add doc comments + stability (most Experimental pre-v1)             |
| Snapshot                    | KEEP   | User-facing introspection                                           |
| ResourceSnapshot            | KEEP   | Part of Snapshot                                                    |
| ResumeSnapshot              | KEEP   | Part of Snapshot                                                    |
| AssetStrategy (if exported) | TAG    | Document extension point or replace with more granular interfaces   |
| MetricsProvider()           | TAG    | Mark Experimental – may move behind separate telemetry subpackage   |
| UpdateTelemetryPolicy       | TAG    | Experimental (policy model may evolve)                              |
| Policy()                    | TAG    | Experimental accessor                                               |
| HealthEvaluatorForTest      | INT    | Test-only; relocate to test utility or build tag                    |

## 2. Models Package (`engine/models`)

All current types appear core; propose KEEP with doc comments & stability tags.
| Symbol | Action | Notes |
| ------ | ------ | ----- |
| Page | KEEP | Core data model |
| PageMeta | KEEP | |
| OpenGraphMeta | KEEP | |
| CrawlResult | KEEP | |
| CrawlStats | KEEP | (not fully surfaced yet) |
| RateLimitConfig | TAG | Might relocate under ratelimit if narrowed |
| CrawlError + helpers | KEEP | Provide stable error wrapping |
| Error sentinel vars | TAG | Mark Experimental; may consolidate |

## 3. Crawler Package (`engine/crawler`)

| Symbol                         | Action | Notes                                        |
| ------------------------------ | ------ | -------------------------------------------- |
| FetchResult                    | TAG    | Possibly simplify / hide Metadata map pre-v1 |
| Fetcher (interface)            | KEEP   | Extension point                              |
| FetchPolicy                    | TAG    | Consider reduction; align with Config fields |
| FetcherStats                   | TAG    | May shrink; mark Experimental                |
| Deprecated alias (FetchedPage) | REMOVE | Accept breaking removal now                  |

## 4. Pipeline Package (`engine/pipeline`)

Proposal: Internalize entire package (INT) – expose minimal streaming interface via facade if needed.
| Symbol | Action | Notes |
| ------ | ------ | ----- |
| Pipeline | INT | Implementation detail |
| PipelineConfig | INT | Move construction logic into engine.Config translation |
| StageStatus / StageMetrics / PipelineMetrics | TAG | If snapshots required, wrap inside facade types |
| NewPipeline | INT | Hidden behind Engine.New |
| Stop / Metrics / ProcessURLs etc. | INT | Facade proxies selected behavior |

## 5. Processor / Output / Resources / RateLimit Packages

Intent: Evaluate each for public necessity.
| Package | Action | Notes |
| ------- | ------ | ----- |
| processor | INT | Implementation detail |
| output (interfaces) | KEEP | OutputSink extension point is user value |
| output/stdout | TAG | Example implementation (mark Experimental) |
| resources | INT | Manager internalized; expose high-level snapshot only |
| ratelimit | TAG | Keep if embedding / external tuning desirable; else INT |

## 6. Telemetry Packages

| Package           | Action | Notes                                                    |
| ----------------- | ------ | -------------------------------------------------------- |
| telemetry/metrics | TAG    | Provider selection API may churn                         |
| telemetry/events  | INT    | Internal bus (no stable contract yet)                    |
| telemetry/tracing | TAG    | Adaptive tracer heuristics may change                    |
| telemetry/health  | INT    | Expose only summarized status via facade Policy/Snapshot |
| telemetry/policy  | TAG    | Experimental; subject to field renames                   |
| telemetry/logging | INT    | Internal logging integration                             |

## 7. Strategies / Extension Points

Consolidate extension interfaces into a single `strategies.go` under root engine package or dedicated `engine/strategies` (already present – audit content):
| Interface | Source | Action | Notes |
| --------- | ------ | ------ | ----- |
| Fetcher | crawler | KEEP | Re-export from strategies w/ doc? |
| OutputSink | output | KEEP | |
| AssetStrategy | engine (facade) | TAG | Clarify lifecycle + concurrency contract |

## 8. Test Utilities Exposure

Test-only constructs currently exported should be internalized or placed under `testing/` path.
| Symbol | Package | Action | Notes |
| ------ | ------- | ------ | ----- |
| SetMetricsForTest | pipeline | INT | Move to internal test helper |
| HealthEvaluatorForTest | engine | INT | As above |

## 9. Documentation & Stability Tags

Pattern: Add leading package doc summarizing role; for each exported type/function add stability comment prefix:

```go
// Experimental: This type is not yet stable and may change before v1.
```

Stable (when designated) omit prefix or use:

```go
// Stable: Backward compatible within minor releases after v1.0.
```

## 10. Migration Steps Sequence

1. Approve this candidate list.
2. Create `internal/` subpackages and move INT items.
3. Update imports & facade wrappers.
4. Add doc comments + stability tags (TAG items) in same commit or incremental.
5. Remove deprecated alias types.
6. Run tests, add new tests to cover facade proxies for previously direct APIs.
7. Update CHANGELOG with pruning summary.

## 11. Open Questions

- Should `RateLimitConfig` live under `engine/ratelimit` or remain in models for simplified single-config import?
- Do we re-export strategy interfaces from a central package to reduce import sprawl?
- Is a lightweight event subscription API part of near-term public surface? (currently no – keep internal).

## 12. Deferred Considerations

- Potential split of telemetry providers into separate module if dependency weight grows.
- Plugin registry mechanism (Phase 6 or later).

---

Feedback welcome; after sign-off we proceed with internalization (Wave 3 execution).
