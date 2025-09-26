# Site Scraper Implementation Plan

_Agentic workflow optimized for systematic development with proven TDD methodology_

## Development Philosophy

- **Test-Driven Development (TDD)**: Write tests first, implement to pass, refactor with confidence
- **Iterative Refactoring**: Build, test, refine in small, verifiable increments
- **Early Validation**: Test each component independently before integration
- **Modular Architecture**: Single responsibility principle with clean interfaces
- **Measurable Progress**: Clear success criteria with comprehensive test coverage
- **Fail-Fast Approach**: Identify and resolve blockers quickly through systematic testing

---

## 🧪 TDD Methodology & Best Practices

### Core TDD Principles Applied

#### Red-Green-Refactor Cycle

1. **Red**: Write failing tests that define expected behavior
2. **Green**: Implement minimal code to make tests pass
3. **Refactor**: Improve code structure while maintaining test coverage

#### Test-First Development Benefits

- **Interface Design**: Tests define clean API contracts before implementation
- **Regression Prevention**: Comprehensive test suite catches breaking changes immediately
- **Documentation**: Tests serve as living documentation of system behavior
- **Confidence**: Full coverage enables fearless refactoring and optimization

#### Modular Testing Strategy

- **Unit Tests**: Individual component behavior validation
- **Integration Tests**: Cross-module interaction verification
- **End-to-End Tests**: Complete pipeline functionality validation
- **Performance Tests**: Benchmarking and resource usage monitoring

### Proven TDD Success Stories

#### ✅ Asset Management Module Extraction

**Challenge**: Extract asset management from 1109-line monolith without breaking changes  
**TDD Solution**:

- Created comprehensive test suite before extraction (449 test lines)
- Defined module interfaces through test specifications
- Maintained backward compatibility via type alias testing
- Verified independent operation through isolated test execution

**Result**: 100% test coverage, zero breaking changes, fully modular architecture

#### ✅ Content Processing Pipeline Enhancement

**Challenge**: Fix multiple failing tests while maintaining functionality  
**TDD Solution**:

- Analyzed each test failure to understand expected behavior
- Implemented fixes iteratively with immediate test validation
- Enhanced functionality based on test requirements (image extraction, validation logic)
- Refactored implementation while preserving test contracts

**Result**: 11/11 test suites passing, enhanced functionality, improved code quality

#### ✅ Refactoring Safety Net

**Challenge**: Major architectural changes without regression  
**TDD Solution**:

- Existing test suite provided safety net for refactoring
- Each change validated against complete test suite
- Iterative approach with continuous integration verification
- Test-driven interface design prevented coupling issues

**Result**: 57% code reduction, 83% lint improvement, zero functional regression

---

## Phase 1: Foundation & Core Architecture

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

## Phase 2: Content Processing Pipeline

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

## Phase 3: Concurrency & Performance Optimization

**Timeline: Day 3-4 | Success Metric: Process 500+ pages efficiently with resource monitoring**

### 3.1 Multi-Stage Pipeline Architecture

- [x] Implement channel-based pipeline stages:
  - URL Discovery → Content Extraction → Processing → Output
- [x] Add backpressure handling between stages
- [x] Create worker pools for each stage with configurable sizing
- [x] Implement graceful shutdown with context cancellation

### 3.2 Intelligent Rate Limiting

- [ ] Implement adaptive rate limiter based on server responses
- [ ] Add per-domain rate limiting capabilities
- [ ] Create circuit breaker for server overload detection
- [ ] Add retry logic with exponential backoff

### 3.3 Resource Management

- [ ] Implement memory monitoring and management
- [ ] Add content caching with LRU eviction
- [ ] Create disk spillover for large datasets
- [ ] Add progress checkpointing for resumable crawls

**Validation Test**: Crawl 500+ page site within memory limits, complete within expected timeframe.

---

## Cross-Cutting Initiative: Engine Package Decomposition

**Decision**: Begin incremental extraction NOW (overlapping with late Phase 3) rather than deferring until after all feature phases. Early decomposition reduces future refactor cost, establishes clean public APIs before they ossify, and keeps dependency graphs stable for upcoming output, observability, and CLI/TUI work.

### Rationale

- **Avoid Deep Coupling**: Additional Phase 4–6 features would otherwise expand imports into `internal/*`, increasing migration surface later.
- **API Stabilization**: Defining a facade early allows downstream (CLI / TUI / future API server) to target a stable seam.
- **Test Leverage**: Existing comprehensive suites provide safety net for incremental moves (forwarding shims + aliasing).
- **Parallelizable**: Migration tasks are mostly mechanical and can proceed alongside feature development without blocking critical paths.

### Scope

Create `packages/engine` as the canonical embedding API containing (eventually):

- `models` (public data & config structures)
- `pipeline` (multi-stage orchestration)
- `ratelimit` (adaptive limiter)
- `processor` (content extraction & markdown conversion)
- `assets` (discovery, download, optimization, rewrite)
- `crawler` (URL discovery / queue)
- `output` (future output format generation)

### Non-Goals (Now)

- Multi-module split (stay single `go.mod` initially)
- Public semantic version tagging (defer until facade API stabilized)
- Breaking changes to current internal tests (all must remain green during migration)

### Strategy & Phases

1. **P0 – Facade Skeleton** (DONE scaffold): Create `packages/engine` directory + README + placeholder file.
2. **P1 – Facade Definition**: Introduce `Engine` struct & interfaces delegating to existing internal packages (no code moves yet).
3. **P2 – Low-Risk Moves**: Relocate `ratelimit` & `pipeline` under `packages/engine/` with alias stubs left at old paths:
  - Add deprecated forwarding files (`// Deprecated: use packages/engine/...`) to preserve imports.
4. **P3 – Models Consolidation**: Move `pkg/models` → `packages/engine/models`; add re-export wrappers preserving old package for soft migration.
5. **P4 – Processor & Assets**: Migrate content & asset modules; prune unexported symbols; expose only necessary public API via facade.
6. **P5 – Crawler & Output**: Move remaining modules, introducing interfaces where direct coupling exists.
7. **P6 – Cleanup & Hardening**: Remove forwarding shims after internal imports updated; run `-race`, lint, and regenerate docs.
8. **P7 – CLI/TUI Layering**: Implement new `cmd/scraper` using only facade; prepare future `packages/tui`.

### Task Checklist

- [ ] Define `Engine` facade interfaces (Start, Stop, Snapshot, Configure)
- [ ] Add integration test for facade (ensures parity with direct pipeline usage)
- [ ] Move `ratelimit` package + add forwarding shim
- [ ] Move `pipeline` package + add forwarding shim
- [ ] Introduce `packages/engine/models` + re-export `pkg/models`
- [ ] Update imports across repo to use new paths (incremental)
- [ ] Move `processor` & `assets` with tightened export surface
- [ ] Move `crawler` & `output`
- [ ] Remove deprecated forwarding shims
- [ ] Add documentation: migration log & public API reference
- [ ] Establish versioning policy (doc only, no tags yet)

### Acceptance Criteria

- All existing tests pass after each migration step
- No circular imports introduced
- Facade integration test green, demonstrating end-to-end crawl
- Forwarding shims removed with zero unresolved imports
- Public API documented (README + `godoc` comments) and stable for CLI adoption

### Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Hidden internal coupling | Migration stalls | Introduce interfaces during move (dependency inversion) |
| Large diffs reduce review quality | Slower iteration | Phase-by-phase small PRs with shims |
| Accidental API surface bloat | Long-term maintenance cost | Explicit facade; keep subpackages internal except necessary exports |
| Test flakiness during moves | Lost confidence | Run full suite (incl. `-race`) per step |

### Follow-Up Opportunities

- After stabilization: consider multi-module (engine vs. cli) if external embedding demand arises.
- Add semantic version tagging once API adoption starts.

---

## Phase 4: Output Generation System

**Timeline: Day 4-5 | Success Metric: Generate high-quality PDF, MD, and HTML outputs**

### 4.1 Multi-Format Output Engine

- [ ] Implement markdown compilation with table of contents
- [ ] Create HTML template system for web output
- [ ] Add PDF generation pipeline (markdown → HTML → PDF)
- [ ] Implement output format plugins for extensibility

### 4.2 Document Assembly

- [ ] Create document structure organizing by URL hierarchy
- [ ] Generate navigation and cross-reference links
- [ ] Implement content deduplication logic
- [ ] Add metadata extraction and inclusion

### 4.3 Content Enhancement

- [ ] Generate automatic table of contents
- [ ] Create index of pages and sections
- [ ] Add search functionality for HTML output
- [ ] Implement custom CSS styling for professional appearance

**Validation Test**: Generate publication-ready documents in all three formats from 100+ page crawl.

---

## Phase 5: Production Readiness

**Timeline: Day 5-6 | Success Metric: Production-ready tool with comprehensive error handling**

### 5.1 Configuration Management

- [ ] Create comprehensive YAML configuration system
- [ ] Add command-line argument parsing
- [ ] Implement configuration validation and defaults
- [ ] Add site-specific configuration profiles

### 5.2 Error Handling & Recovery

- [ ] Implement comprehensive error classification
- [ ] Add failed URL retry queue with intelligent retry logic
- [ ] Create partial completion support (generate from successful pages)
- [ ] Add detailed error reporting and logging

### 5.3 Monitoring & Observability

- [ ] Implement real-time progress reporting
- [ ] Add performance metrics (pages/sec, memory usage, error rates)
- [ ] Create completion time estimation
- [ ] Add structured logging with multiple verbosity levels

**Validation Test**: Handle various failure scenarios gracefully, provide clear progress feedback.

---

## Phase 6: Polish & CLI Experience

**Timeline: Day 6-7 | Success Metric: Professional CLI tool ready for distribution**

### 6.1 Command-Line Interface

- [ ] Create intuitive CLI with subcommands (crawl, resume, status)
- [ ] Add progress bars and real-time status updates
- [ ] Implement interactive configuration prompts
- [ ] Add help system and usage examples

### 6.2 Output Quality Improvements

- [ ] Fine-tune markdown output formatting
- [ ] Optimize PDF generation (fonts, spacing, page breaks)
- [ ] Enhance HTML output with responsive design
- [ ] Add output validation and quality metrics

### 6.3 Documentation & Testing

- [ ] Create comprehensive README with examples
- [ ] Add unit tests for critical components
- [ ] Create integration tests with sample sites
- [ ] Document configuration options and best practices

**Validation Test**: Tool can be used by external user with minimal guidance.

---

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
