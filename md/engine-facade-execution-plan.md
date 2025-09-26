# Engine Facade & Resilience Enhancements – Execution Plan

Status: Proposed (Ready to Execute)
Date: September 26, 2025
Owner: Architecture / Platform Track
Related Docs: `engine-decomposition.md`, `phase3.3-retrospective.md`, `phase3-progress.md`

---
## 1. Objectives
| Objective | Description | Success Metric |
|-----------|-------------|----------------|
| Unified Facade | Provide a single high-level API to run crawls programmatically | CLI / future TUI imports only `packages/engine` |
| Test Stability | Remove external HTTPBIN dependency from tests | Full `go test -race ./...` green offline |
| Consistent Introspection | Merge limiter + resource + pipeline metrics | `engine.Snapshot()` returns composite struct |
| Resumability Primitive | Enable optional continuation from prior run | Resume test processes second half of URL set using checkpoint file |
| Observability Ready | Structure for future metrics export without tight coupling | Snapshot fields map 1:1 to potential Prometheus metrics |

---
## 2. Scope (In / Out)
IN:
- Engine facade implementation + integration tests
- Refactor asset tests to local mock server (remove external network)
- Snapshot design spanning pipeline, limiter, resources
- Checkpoint resume (basic: skip already checkpointed URLs)
- Documentation & migration notes

OUT (Deferred):
- Multi-module split
- Distributed coordination/resume across hosts
- Persistent rate limiter state across runs
- Advanced analytics dashboard

---
## 3. Deliverables
| Deliverable | File(s) / Artifact |
|-------------|--------------------|
| Engine facade implementation | `packages/engine/engine.go`, `packages/engine/facade.go` |
| Public snapshot types | `packages/engine/snapshot.go` |
| Integration tests | `packages/engine/engine_integration_test.go` |
| Mock HTTP server utilities | `internal/test/httpmock/server.go` or `testutil/httpmock.go` |
| Refactored asset tests | Updated `internal/processor/asset*_test.go` (no external URLs) |
| Resume-from-checkpoint logic | `packages/engine/resume.go` + tests |
| Updated docs | `engine-decomposition.md`, new section in `phase4-planning.md` |
| Migration guide | `md/engine-migration-notes.md` |

---
## 4. Phased Plan & Sequencing
| Phase | Name | Rationale | Depends On |
|-------|------|-----------|------------|
| P1 | Facade Construction (Read-Only) | Provide API surface early | Existing pipeline/limiter/resources |
| P2 | Snapshot Unification | Needed before CLI adoption | P1 |
| P3 | HTTP Test Isolation | Stabilize test baseline before deeper refactors | none |
| P4 | Resume-from-Checkpoint MVP | Leverage checkpoint log now | P2 (Snapshot includes resume metrics) |
| P5 | Pipeline Import Migration | Route existing `main.go` through facade | P1–P4 stable |
| P6 | Cleanup & Deprecations | Remove direct internal references in CLI path | P5 |
| P7 | Documentation + Version Tagging Prep | Finalize API contracts | P6 |

Execution order: P3 can run in parallel with P1/P2 but merge after facade ready for consistent baseline. Resume (P4) requires facade for injection of checkpoint path + policy.

---
## 5. Detailed Tasks
### P1: Facade Construction
- Define `type Engine struct { cfg Config; pipeline *pipeline.Pipeline; limiter ratelimit.RateLimiter; resources *resources.Manager }`
- Expose: `New(cfg Config) (*Engine, error)`, `Start(ctx context.Context, seeds []string) (<-chan *models.CrawlResult, error)`, `Stop() error`, `Snapshot() Snapshot`
- Provide functional options (e.g., `WithLimiter(l ratelimit.RateLimiter)`, `WithResourceManager(m *resources.Manager)`)
- Add narrow exported `Config` (wrapper referencing existing model configs)
- TDD: Write facade integration test first using seed URLs and asserting result count + snapshot non-zero metrics

### P2: Snapshot Unification
- Create `Snapshot` struct: `{ StartedAt time.Time; Duration; Pipeline PipelineStats; Limiter *LimiterStats; Resources *ResourceStats; Rates map[string]DomainRateSummary }`
- Implement internal adapters: gather metrics from `pipeline.Metrics()`, `limiter.Snapshot()`, resource manager (cache size, spill count, checkpoint entries flushed)
- TDD: snapshot test verifying fields populate, stable after multiple calls (idempotent reading)

### P3: HTTP Test Isolation
- Introduce `internal/test/httpmock` with:
  - Simple multiplexer supporting scripted responses (URL pattern → status, body, delay, headers)
  - Helper: `NewServer(t *testing.T, routes []RouteSpec) *MockServer`
- Refactor asset downloader tests to swap HTTP client base URL to mock server; parameterize downloader with `BaseURL` / injectable `*http.Client`
- Remove external `httpbin.org` references
- TDD: ensure failing tests replaced by deterministic ones; run offline verification (simulate 503, 200, latency)

### P4: Resume-from-Checkpoint MVP
- Extend facade config: `Resume bool`, `CheckpointPath string`
- On `Start`: if `Resume` = true, read checkpoint file lines into set → filter seeds before calling `ProcessURLs`
- Optional: load spilled pages lazily (not required MVP)
- TDD: create temporary checkpoint file with subset; assert resumed run only processes remaining URLs
- Include metrics: `ResumedCount`, `SkippedFromCheckpoint`

### P5: Pipeline Import Migration
- Update `main.go` replacing direct pipeline creation with facade usage
- Provide minimal CLI flags (seed file, resume toggle) – optional scaffolding
- Ensure no external packages outside `packages/engine` used in CLI except logging/config parsing

### P6: Cleanup & Deprecations
- Add deprecation comments to any now-redundant direct constructors (e.g., `pipeline.NewPipeline`) if meant to become internal later
- Optionally introduce `internal/pipeline` move to `packages/engine/pipeline` with forwarding stub (if ready)
- Enforce (grep script) absence of direct `internal/*` imports in CLI path

### P7: Documentation & Version Prep
- Update `engine-decomposition.md` progress table
- Add `API_STABILITY.md` describing facade commitments (what can change vs. stable)
- Prepare CHANGELOG baseline for future semver

---
## 6. TDD Strategy
| Layer | Test Types | Key Assertions |
|-------|------------|----------------|
| Facade | Integration | Start/Stop idempotency, channel closure, snapshot non-zero |
| Snapshot | Unit | Aggregated counters consistent across successive calls |
| HTTP Mock | Unit | Route matching precedence, delay injection, header propagation |
| Asset Downloader (refactored) | Unit | Retries on scripted 503, success on 200, no external network |
| Resume | Integration | Skipped URLs not reprocessed, metrics reflect skip counts |
| Migration | Smoke | CLI main builds & runs with fixture seeds |

Add race detector runs for facade + resume integration tests.

---
## 7. Risks & Mitigation
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Facade API churn | Downstream instability | Define minimal API; defer extras; doc experimental fields |
| Snapshot cost | Performance overhead | Lazy domain detail collection (cap top N domains) |
| Test fragility (resume) | Flaky ordering reliance | Use deterministic seed ordering, wait for channel close |
| Mock HTTP drift | False confidence | Allow injection of chaos (random delay) test variant |
| Parallel migration & feature dev conflict | Merge collisions | Small, frequent PRs per phase |

---
## 8. Metrics & Exit Criteria
| Criterion | Threshold |
|-----------|-----------|
| Full suite (incl. facade tests) | `go test ./...` PASS offline |
| Race detector (core packages) | `go test -race ./packages/engine ./internal/pipeline ./internal/resources ./internal/ratelimit` PASS |
| External network references | 0 occurrences of `httpbin.org` or raw external domains in tests |
| Snapshot fidelity | All non-nil sections populated in integration test |
| Resume correctness | Skipped count == size of pre-populated checkpoint |

---
## 9. Work Breakdown & Estimation (Relative)
| Task | Size | Notes |
|------|------|-------|
| Facade scaffold + test (P1) | S | Wrap existing structs |
| Snapshot adapter (P2) | M | Domain summarization logic |
| HTTP mock infra (P3) | S | Simple mux + test fixtures |
| Asset test refactor (P3) | M | Replace references & inject client |
| Resume MVP (P4) | S | Filter seeds + metrics |
| CLI migration (P5) | S | Narrow change surface |
| Deprecation shims (P6) | XS | Comments + grep enforcement |
| Docs + stability guide (P7) | S | Writing only |

---
## 10. Tracking Checklist
- [ ] P1 Facade integration test passes
- [ ] P2 Snapshot aggregated & documented
- [ ] P3 Mock server added & asset tests offline
- [ ] P4 Resume feature test passes
- [ ] P5 CLI uses facade only
- [ ] P6 No direct internal imports in CLI path
- [ ] P7 Docs & stability guide merged

---
## 11. Rollback / Contingency
| Scenario | Action |
|----------|--------|
| Facade instability | Keep existing direct pipeline usage; gate facade behind build tag temporarily |
| Mock server issues | Temporarily mark refactored tests with skip + revert to prior implementation (no external calls) |
| Resume feature regression | Feature-flag resume (config + off by default) |

---
## 12. Open Questions
1. Should snapshot include raw per-URL latency samples or only aggregates? (Recommendation: aggregates only now.)
2. How many domains to display in snapshot by default? (Proposal: top 10 by recent activity.)
3. Do we gate resume behind experimental flag until user stories surface? (Likely yes.)

---
## 13. Next Immediate Action
Begin P1: write `engine_integration_test.go` describing desired facade contract before implementation; then implement facade to satisfy test.

---
_End of Execution Plan_
