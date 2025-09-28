# Site Scraper Implementation Plan

_Central execution plan synchronized with GitHub issues (authoritative backlog)._

## 🚀 PROGRESS UPDATE (2025-09-29)

**MAJOR MILESTONE ACHIEVED: Live Test Site Phase 6.1 Complete ✅**

We have successfully implemented a comprehensive live test site (`tools/test-site`) that provides a realistic HTTP origin for crawler testing. This represents significant progress toward production-ready integration testing capabilities.

### Key Accomplishments This Session
- **Complete Test Site Implementation**: Full Bun + React application with TypeScript transpilation
- **Rich Wiki-Style Content**: 7+ routes with comprehensive content patterns (typography, code, admonitions, tables, metadata)
- **Professional UI**: shadcn/ui component integration with responsive Tailwind CSS design
- **API Endpoints**: `/api/ping`, `/api/posts`, `/api/slow` for comprehensive crawler testing
- **Asset Testing**: Working and intentionally broken assets for error handling validation
- **Performance**: Sub-200ms startup time with deterministic content generation

### Next Priority (Phase 6.2)
**Go Test Harness Integration** - Implement `WithLiveTestSite()` helper and migrate integration tests to use the live site instead of synthetic mocks.

---

This document now maps every phase/sub-phase item to a GitHub Issue for single-source-of-truth tracking. Closed checkboxes here should mirror closed issues; creation of new scope MUST create an issue first.

## Development Philosophy

- **Iterative Development**: Build, test, refine in small increments
- **Early Validation**: Test each component independently before integration
- **Measurable Progress**: Clear success criteria for each milestone
- **Fail-Fast Approach**: Identify and resolve blockers quickly

### Expanded Non-Functional Objectives (2025-Q4 Refresh)

We are adding explicit non-functional goals to prevent implicit drift:

- **Determinism**: Integration + end-to-end tests must exhibit zero flakes over 20 consecutive CI runs (target <0.5% flake rolling 30‑day). All dynamic content (timestamps, random IDs) must be fixture-stable or normalized in assertions.
- **Observability Baseline**: Engine exposes health + metrics (when enabled) with <50ms handler overhead p95 under synthetic load; minimal surface (no internal provider leakage).
- **Performance Budgets**: Establish & enforce throughput and latency budgets (see Success Metrics). Any PR breaching budget must include perf justification & follow-up issue.
- **Resource Efficiency**: Memory ceiling for 1k page crawl: <2GB RSS; CPU utilization target: <400% on 8 vCPU machine (leaves headroom for CI noise).
- **Security & Compliance**: Respect robots.txt (allow override flag for test harness only), never fetch off-scope domains, and redact sensitive headers in logs.
- **Accessibility (A11y) Forward Prep**: Output HTML passes a basic a11y lint (heading hierarchy + alt text presence) by Phase 7.
- **Reproducible Dev Environment**: Single make target (`make dev-env`) bootstraps toolchain (Go, Bun) with pinned versions; CI validates lockfiles.
- **Documentation Quality**: For each new public facade symbol, doc coverage (godoc) ≥95% (auto-checked in Phase 7 gate).
- **Extensibility Guardrails**: No new exported surface without an accompanying issue labeled `api-surface` and plan delta.

These are enforceable via lightweight CI scripts introduced incrementally (see new Phase 6 & 7 gates).

---

## Phase 1: Foundation & Core Architecture (Complete)

**Timeline: Day 1-2 | Success Metric: Basic crawler extracts content from 10 URLs**

### 1.1 Project Setup & Dependencies

- [x] Initialize Go module with proper versioning
- [x] Add core dependencies:
  - `github.com/gocolly/colly/v2` - Web crawling framework
  - `github.com/PuerkitoBio/goquery` - HTML parsing
  - `github.com/JohannesKaufmann/html-to-markdown/v2` - Content conversion
  - `gopkg.in/yaml.v3` - Configuration management
- [x] Create project structure:
  ```
  /cmd/scraper/         - CLI entry point
  /internal/crawler/    - Crawling logic
  /internal/processor/  - Content processing
  /internal/output/     - Output generation
  /internal/config/     - Configuration handling
  /pkg/models/          - Data structures
  ```

### 1.2 Core Data Models

- [x] Define `Page` struct (URL, content, metadata, links)
- [x] Define `ScraperConfig` struct (workers, rate limits, output formats)
- [x] Define `CrawlResult` struct for pipeline communication
- [x] Create error types for domain-specific error handling

### 1.3 Basic Crawler Implementation

- [x] Initialize Colly collector with domain restrictions
- [x] Implement URL queue with `sync.Map` for visited tracking
- [x] Add basic HTML content extraction (target main content areas)
- [x] Implement link discovery and queue management
- [x] Add request/response logging for debugging

**Validation Test**: Successfully crawl 10 pages from a test wiki site, extract titles and content.

---

## Phase 2: Content Processing Pipeline (Complete)

**Timeline: Day 2-3 | Success Metric: Clean markdown output from crawled content**

### 2.1 HTML Content Cleaning

- [x] Implement content selector targeting (article, .content, main)
- [x] Remove unwanted elements (nav, footer, sidebar, ads)
- [x] Handle relative URL conversion to absolute URLs
- [x] Extract and preserve metadata (title, author, date)

### 2.2 Content Processing Workers

- [x] Create worker pool pattern for content processing
- [x] Implement HTML-to-Markdown conversion pipeline
- [x] Add content validation and quality checks
- [x] Handle special content types (code blocks, tables, images)

### 2.3 Asset Management

- [x] Implement image URL extraction and cataloging
- [x] Add option for image downloading vs. URL preservation
- [x] Handle asset optimization (resize, compress)
- [x] Create asset manifest for output generation

**Validation Test**: Process 50 pages, generate clean markdown with preserved formatting and working links.

---

## Phase 3: Concurrency & Performance Optimization (Partially Complete)

**Timeline: Day 3-4 | Success Metric: Process 500+ pages efficiently with resource monitoring**

### 3.1 Multi-Stage Pipeline Architecture

- [x] Implement channel-based pipeline stages:
  - URL Discovery → Content Extraction → Processing → Output
- [x] Add backpressure handling between stages
- [x] Create worker pools for each stage with configurable sizing
- [x] Implement graceful shutdown with context cancellation

### 3.2 Intelligent Rate Limiting

- [x] Implement adaptive rate limiter based on server responses (AIMD + EWMA) – implemented
- [x] Add per-domain rate limiting capabilities – implemented
- [x] Create circuit breaker for server overload detection – implemented
- [x] Add retry logic with exponential backoff – implemented
- [ ] Persistent limiter state across runs (Issue #14)

### 3.3 Resource Management

- [x] Implement memory monitoring and management (lightweight sampling) – implemented
- [x] Add content caching with LRU eviction – implemented
- [x] Create disk spillover for large datasets – implemented
- [x] Add progress checkpointing for resumable crawls – implemented
- [ ] Spill retention / cleanup policy (Issue #8)

**Validation Test**: Crawl 500+ page site within memory limits, complete within expected timeframe.

---

## Phase 4: Output Generation System (Scheduled)

**Timeline: Day 4-5 | Success Metric: Generate high-quality PDF, MD, and HTML outputs**

### 4.1 Multi-Format Output Engine

- [ ] Markdown compilation with table of contents (Issue #19)
- [ ] HTML template system for web output (Issue #20)
- [ ] PDF generation pipeline (Issue #21)
- [ ] Output format plugin registration (Issue #22)

### 4.2 Document Assembly

- [ ] Document structure by URL hierarchy (Issue #23)
- [ ] Navigation and cross-reference generation (Issue #24)
- [ ] Content deduplication logic (Issue #25)
- [ ] Metadata extraction & inclusion (Issue #26)

### 4.3 Content Enhancement

- [ ] Automatic table of contents generation (Issue #27)
- [ ] Index of pages and sections (Issue #28)
- [ ] Search functionality for HTML output (Issue #29)
- [ ] Custom CSS styling system (Issue #30)

**Validation Test**: Generate publication-ready documents in all three formats from 100+ page crawl.

---

## Phase 5: Production Readiness (Queued)

**Timeline: Day 5-6 | Success Metric: Production-ready tool with comprehensive error handling**

### 5.1 Configuration Management

- [ ] Comprehensive YAML configuration system (Issue #31)
- [ ] Configuration validation & defaulting (Issue #32)
- [ ] Site-specific configuration profiles (Issue #33)

### 5.2 Error Handling & Recovery

- [ ] Comprehensive error classification (Issue #34)
- [ ] Failed URL retry queue with backoff (Issue #35)
- [ ] Partial completion support (Issue #36)
- [ ] Detailed error reporting & logging (Issue #37)

### 5.3 Monitoring & Observability

- [ ] Real-time progress reporting (Issue #38)
- [ ] Performance metrics collection (Issue #39)
- [ ] Completion time estimation (Issue #40)
- [ ] Structured logging & verbosity levels (Issue #41) (may integrate with Issue #9 if merged)
- [ ] Snapshot JSON Schema (Issue #5)
- [ ] Prometheus metrics exporter (Issue #6)

**Validation Test**: Handle various failure scenarios gracefully, provide clear progress feedback.

---

## Phase 6: Live Test Site & Realistic Integration Surface (IN PROGRESS) ⚠️

**Timeline: Interleaved after Phase 5 (Production Readiness) | Success Metric: Stable, deterministic live wiki-style site powering ≥1 replaced mock-based integration test**

Purpose: Introduce a lightweight “Ariadne Wiki” live site (Bun + React) that emulates common knowledge base / Obsidian Quartz style patterns: nested pages, asset references, broken links, slow endpoints, robots variants, metadata richness. This becomes the canonical real HTTP origin for end-to-end crawler validation (superseding scattered synthetic fixtures).

### 6.1 Test Site Implementation (P1 – Minimal) ✅ COMPLETE

- [x] Create `tools/test-site` with formalized package name `@ariadne/testsite` ✅
- [x] Core routes: `/`, `/about`, `/docs/getting-started`, `/docs/deep/n1/n2/n3/leaf`, `/blog/*`, `/tags/index` ✅
- [x] Assets: 2 working SVG images + intentionally missing PNGs, CSS, JS, comprehensive styling ✅
- [x] API endpoints: `/api/ping`, `/api/posts` (static JSON), `/api/slow` (400–600ms latency injection) ✅
- [x] Metadata: Complete `<meta>` tags, canonical links, OpenGraph tags, frontmatter simulation ✅
- [x] Deterministic build: Fixed timestamps, no random content, stable for testing ✅
- [x] Startup banner: `TESTSITE: listening on http://127.0.0.1:5173, Robots mode: allow` ✅

**Additional Achievements Beyond Original Scope:**
- [x] **Rich Content**: Comprehensive wiki-style content with typography, code fences, admonitions, tables, math placeholders
- [x] **shadcn/ui Integration**: Professional UI components (Card, Button, Alert) throughout site
- [x] **TypeScript Transpilation**: Bun.build() pipeline with ESM format for browser compatibility
- [x] **Responsive Design**: Mobile-friendly layout with Tailwind CSS utilities
- [x] **Error Handling**: Proper 404s, JSON error responses, graceful failure modes
- [x] **Performance**: Sub-200ms startup time, efficient asset serving

### 6.2 Go Test Harness

- [ ] Helper `WithLiveTestSite(t *testing.T, fn func(baseURL string))` in `engine/internal/testutil/testsite` (or cli equivalent) with reuse (`TESTSITE_REUSE=1`).
- [ ] Port selection: pick configured `TESTSITE_PORT` else scan free port (avoid collisions in parallel CI matrix).
- [ ] Readiness wait ≤5s (fail fast with diagnostic log tail on timeout).
- [ ] Graceful teardown (process kill + wait) unless reuse set.

### 6.3 Integration Test Migration (P1 Scope)

- [ ] Replace at least one current discovery integration test to crawl live site root and assert discovered page set & asset counts.
- [ ] Assert: depth limiting, broken image tracked, slow endpoint does not stall entire crawl (timeout respected), robots allow variant honored.
- [ ] Add golden snapshot (normalized) for a representative page to detect content regression in test site.

### 6.4 Determinism & Metrics (P2)

- [ ] Introduce alternate `robots.txt` via env `TESTSITE_ROBOTS=deny` and test deny-all gating.
- [ ] Add latency jitter boundaries test (ensure p95 within expected window).
- [ ] Add sitemap `/sitemap.xml` and validate crawler ingestion (if supported by then).

### Success Criteria (Phase 6 Gate)

| Criterion               | Target                                                  |
| ----------------------- | ------------------------------------------------------- |
| Site cold start         | < 200ms Bun process ready on M2 Mac / CI small instance |
| Test harness reuse      | Achieved (second test run reuses process)               |
| Flakiness               | 0 flakes in 20 consecutive CI runs                      |
| Integration replacement | ≥1 legacy mock test removed / refactored                |
| Deterministic content   | No timestamp/random diffs across runs                   |

### Risks & Mitigations

| Risk                    | Impact                   | Mitigation                                                   |
| ----------------------- | ------------------------ | ------------------------------------------------------------ |
| Port collisions         | Test flakes              | Dynamic port selection with retry                            |
| Bun install variability | CI failures              | Pin version in `bunfig.toml`; cache install layer            |
| Latency variance        | Flaky slow endpoint test | Constrain delay with fixed seed RNG or deterministic cycle   |
| Content drift           | Assertion churn          | Golden snapshots + PR review checklist for test-site changes |

### Outputs

- New Make targets: `testsite-dev`, `integ-live`.
- Readme section: “Live Test Site Usage”.
- CI job step for Bun install + reuse of test site across integration suite.

### Exit Gate

Phase 7 (CLI Polish) work may begin only after Phase 6 success criteria met and at least one integration test depends on the live site.

---

## Phase 7: Polish & CLI Experience (Queued)

**Timeline: Day 6-7 | Success Metric: Professional CLI tool ready for distribution**

### 7.1 Command-Line Interface

- [ ] CLI subcommands (crawl, resume, status) (Issue #42)
- [ ] Progress bars & real-time status UI (Issue #43)
- [ ] Interactive configuration prompts (Issue #44)
- [ ] Help system & usage examples (Issue #45)

### 7.2 Output Quality Improvements

- [ ] Markdown output formatting refinements (Issue #46)
- [ ] PDF generation optimization (Issue #47)
- [ ] Responsive HTML output enhancements (Issue #48)
- [ ] Output validation & quality metrics (Issue #49)

### 7.3 Documentation & Testing

- [ ] Comprehensive README expansion with examples (Issue #50)
- [ ] Unit tests for critical components (Issue #51)
- [ ] Integration tests with sample sites (Issue #52)
- [ ] Configuration options & best practices documentation (Issue #53)

**Validation Test**: Tool can be used by external user with minimal guidance.

---

## Critical Gap Analysis (Added Q4)

| Gap                                 | Current State                    | Impact if Unaddressed                           | Planned Mitigation                                                 |
| ----------------------------------- | -------------------------------- | ----------------------------------------------- | ------------------------------------------------------------------ |
| Lack of realistic crawling surface  | ✅ Phase 6.1 COMPLETE           | ✅ RESOLVED: Live test site implemented         | Phase 6.2: Go test harness + integration tests                     |
| Determinism enforcement             | Ad hoc                           | Flaky CI undermines confidence                  | Add flake detector script + golden snapshots                       |
| API surface governance in root plan | Implicit via internalisation doc | Risk of accidental re-export                    | Introduce Phase 7 CI check: exported symbol diff vs allowlist      |
| Performance regression visibility   | Manual benchmarking              | Slow unnoticed creep                            | Add periodic benchmark run + budget enforcement (Phase 6 add hook) |
| Output a11y quality                 | Not evaluated                    | Harder future adoption & accessibility debt     | Phase 7 a11y lint pass + alt tag enforcement                       |
| Config sprawl risk                  | Expansion in upcoming phases     | Harder onboarding & docs drift                  | Config schema + validation (#31/#32) before adding new flags       |

Additional gaps to re-evaluate after Phase 6: plugin architecture clarity, multi-language content support, structured data (JSON-LD) extraction fidelity.

---

## Test Coverage Strategy (New)

Current Baseline (engine module quick scan):

| Area / Package (sample) | Approx Coverage | Notes                                                                   |
| ----------------------- | --------------- | ----------------------------------------------------------------------- |
| engine (facade/core)    | 83%             | Healthy, integration + unit blend                                       |
| internal/business/\*    | 74–95%          | High; keep above 80% floor                                              |
| internal/pipeline       | ~70%            | Complex concurrency paths missing edge cases                            |
| internal/output (root)  | 37%             | Needs targeted tests for error paths & large doc assembly               |
| internal/assets         | 22%             | Lowest – add discovery failure & rewrite tests                          |
| internal/processor      | 48%             | Missing branch/edge handling (malformed HTML, encoding)                 |
| telemetry/health        | 71%             | Improve degraded/recovery transitions coverage                          |
| telemetry/logging       | 55%             | Exercise error pathways & context cancellation                          |
| misc internal packages  | 0%              | Stubs / simple structs (resources, ratelimit impl tests live elsewhere) |

Global module total (raw tool output) reported ~58% because many intentional no‑logic packages count as 0%; focus on critical-runtime weighted coverage instead.

### Targets (Phased)

| Phase Gate     | Target (Critical Runtime Weighted) | Hard Failing CI Threshold | Notes                                |
| -------------- | ---------------------------------- | ------------------------- | ------------------------------------ |
| Post Phase 5   | ≥70%                               | Warn below 65%            | Add coverage reporting (no gate yet) |
| Phase 6 Exit   | ≥75%                               | Fail below 70%            | Add live test site induced branches  |
| Phase 7 Start  | ≥80%                               | Fail below 75%            | Enforce with `make coverage-check`   |
| Pre v0.2.0 tag | ≥85%                               | Fail below 80%            | Stretch: raise engine core ≥90%      |

Weighted coverage = (sum covered statements in designated critical packages) / (total statements in those packages). Critical set: engine/, internal/pipeline, internal/business/\*, internal/output, internal/processor, internal/assets, telemetry/health.

### Action Backlog

- [ ] Add `make coverage` (profile) and `make coverage-check` (enforce thresholds via Go tool + awk script).
- [ ] Write asset negative path tests (missing image, rewrite failure fallback) to elevate internal/assets to ≥60%.
- [ ] Add pipeline cancellation & timeout tests (raise internal/pipeline to ≥80%).
- [ ] Add processor malformed HTML & encoding tests (target ≥70%).
- [ ] Add telemetry health state transition regression test (unknown → degraded → healthy → degraded). Raise to ≥85%.
- [ ] Add logging error/ctx tests (target ≥80%).
- [ ] Exclude pure data-only packages from weighted metric (document rationale).

### CI Integration Plan

1. Phase 5: Introduce coverage job (uploads HTML report artifact) – no gating.
2. Phase 6: Start gating on threshold (soft fail turns to hard fail once target met in two consecutive runs).
3. Phase 7: Threshold bump + badge generation (README coverage badge via simple script).
4. Pre-release: Review uncovered lines >200 LOC; create follow-up issues or explicitly annotate as acceptable (rare defensive code).

### Principles

- Favor scenario coverage (integration + live site) rather than brittle line-by-line micro tests.
- When skipping coverage (conditional OS, defensive panic guards), add `// coverage:ignore` style comment (custom lint later).
- Every new exported function MUST have accompanying test or explicit issue.

---

---

## Cross-Cutting: Engine Package Decomposition & Facade (In Progress)

Tracking moved to dedicated issues already created / to be created:

- Persistent limiter state (Issue #14)
- Advanced resume (mid-pipeline) (Issue #15)
- Output plugin system (Issue #16) – overlaps Phase 4.1/6.2
- Legacy banner mode (Issue #18)
- Telemetry architecture design (Issue #12)
- Snapshot rotation (Issue #13)
- Coverage threshold (Issue #11)
- Structured logging & verbosity (Issues #9, #41)

Additional tasks (future issues when ready):

- Engine facade public API reference doc (TBD)
- Migration guide promotion (Issue #7)

### Adapter Strategy Alignment (New)

To reduce coupling as we expand capabilities, we are adopting a consistent **adapter pattern**:

- Telemetry HTTP endpoints (metrics, health, readiness) will live in an adapter package consuming pure engine APIs (`HealthSnapshot`, metrics registry handler) rather than being embedded in `engine`.
- Fetching will evolve toward a `Fetcher` interface with the current Colly integration moved into a `colly` adapter (`packages/engine/fetcher/colly`). Future browser / API / cached fetchers can then be added without altering core orchestration.
- Output sinks already follow this pattern; we will formalize registration for additional formats in later phases.

Benefits: clear dependency direction, easier swapping/testing, smaller core surface, and alignment with future multi-module decomposition.

Acceptance remains: architectural enforcement test (DONE), facade snapshot stability (DONE), incremental refactor continues under test safety net.

### Migration Work Breakdown (M0–M3 Active)

| Slice                   | Status      | Scope                                                      | Key Actions                                                                                                      | Risks                                                       |
| ----------------------- | ----------- | ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| M0 Models consolidation | IN PROGRESS | Create `packages/engine/models`, alias legacy `pkg/models` | Copy types (`Page`, `CrawlResult`, `RateLimitConfig`), add aliases & deprecation comments, update facade imports | Duplicate error constants (ensure single source)            |
| M1 Rate limiter move    | TODO        | Relocate `internal/ratelimit` under engine                 | Move code, add forwarding shim at old path, adjust imports, keep tests green                                     | Import cycles if pipeline references internals unexpectedly |
| M2 Resources move       | TODO        | Relocate `internal/resources`                              | Same shim pattern, verify checkpoint & spill paths unaffected                                                    | Hidden coupling with pipeline config                        |
| M3 Pipeline isolation   | TODO        | Move `internal/pipeline` under engine                      | Introduce small interfaces (RateLimiter, ResourceManager), migrate tests                                         | Largest diff; risk of race regressions                      |
| M4 Processor & Assets   | PLANNED     | Move `internal/processor`, `internal/assets`               | Keep them internal (non-export) under engine tree                                                                | Surface creep if exported prematurely                       |
| M5 Crawler & Output     | PLANNED     | Relocate remaining modules                                 | Evaluate need for sub-interfaces                                                                                 | Output phase may overlap feature dev                        |
| M6 Shim removal         | PLANNED     | Delete forwarding aliases                                  | Update docs & stability file                                                                                     | Downstream break if external users lag                      |

Gate to start Phase 4 Output: Complete through M3 (pipeline moved & stable) plus schema + metrics tasks (#5, #6).

## Success Metrics & Validation Strategy

### Performance Benchmarks

- **Small sites (< 100 pages)**: Complete in under 2 minutes
- **Medium sites (100-500 pages)**: Complete in under 10 minutes
- **Large sites (500+ pages)**: Maintain >50 pages/minute throughput

### Quality Standards

- **Content Fidelity**: 95% of formatting preserved in conversion
- **Link Integrity**: 98% of internal links working in output
- **Error Rate**: <5% failed pages on typical wiki sites
- **Memory Efficiency**: <2GB peak memory for 1000+ page sites

### User Experience Goals

- **Configuration**: Site scraping configured in <5 minutes
- **Progress Visibility**: Clear progress indication throughout process
- **Error Communication**: Clear, actionable error messages
- **Output Quality**: Professional-looking documents suitable for distribution

---

## Risk Mitigation

### Technical Risks

- **Memory Overflow**: Implement streaming processing and disk spillover
- **Rate Limiting**: Build adaptive throttling and respectful crawling
- **Site Variations**: Create flexible content extraction patterns
- **Format Compatibility**: Extensive testing across different site structures

### Implementation Risks

- **Scope Creep**: Stick to core features, defer advanced features to v2
- **Over-Engineering**: Focus on working solution first, optimize later
- **Testing Gaps**: Test with multiple real-world sites throughout development
- **Performance Issues**: Profile and optimize at each phase boundary

---

## Development Environment Setup

### Required Tools

- Go 1.21+ with module support
- Git for version control
- Text editor with Go support
- Sample wiki sites for testing
- PDF viewer for output validation

### Testing Strategy

- Unit tests for core algorithms
- Integration tests with sample sites
- Performance tests with large datasets
- Manual testing with various wiki structures

### Deployment Considerations

- Single binary distribution
- Cross-platform compatibility (Linux, macOS, Windows)
- Docker image for containerized environments
- Documentation for common use cases

---

_This plan is designed for systematic implementation with clear validation points. Each phase builds upon the previous one, with early testing to catch issues before they compound._
