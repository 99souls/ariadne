# Site Scraper Implementation Plan

_Central execution plan synchronized with GitHub issues (authoritative backlog)._

This document now maps every phase/sub-phase item to a GitHub Issue for single-source-of-truth tracking. Closed checkboxes here should mirror closed issues; creation of new scope MUST create an issue first.

## Development Philosophy

- **Iterative Development**: Build, test, refine in small increments
- **Early Validation**: Test each component independently before integration
- **Measurable Progress**: Clear success criteria for each milestone
- **Fail-Fast Approach**: Identify and resolve blockers quickly

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

## Phase 6: Polish & CLI Experience (Queued)

**Timeline: Day 6-7 | Success Metric: Professional CLI tool ready for distribution**

### 6.1 Command-Line Interface

- [ ] CLI subcommands (crawl, resume, status) (Issue #42)
- [ ] Progress bars & real-time status UI (Issue #43)
- [ ] Interactive configuration prompts (Issue #44)
- [ ] Help system & usage examples (Issue #45)

### 6.2 Output Quality Improvements

- [ ] Markdown output formatting refinements (Issue #46)
- [ ] PDF generation optimization (Issue #47)
- [ ] Responsive HTML output enhancements (Issue #48)
- [ ] Output validation & quality metrics (Issue #49)

### 6.3 Documentation & Testing

- [ ] Comprehensive README expansion with examples (Issue #50)
- [ ] Unit tests for critical components (Issue #51)
- [ ] Integration tests with sample sites (Issue #52)
- [ ] Configuration options & best practices documentation (Issue #53)

**Validation Test**: Tool can be used by external user with minimal guidance.

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
