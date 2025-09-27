# Phase 5 Engine Architecture Analysis

**Status**: Complete  
**Date**: September 26, 2025  
**Scope**: Critical evaluation of engine package as core business layer  
**Next Phase**: Engine-centric architecture consolidation → staged evolution toward a **multi-module monorepo** (engine core + runtime/API + CLI + presentation surfaces)

---

## Executive Summary

Ariadne has successfully established `packages/engine` as a functional facade but **critical architectural gaps** prevent it from serving as a true core business layer. While the engine provides lifecycle management and metric aggregation, the functional requirements remain scattered across `internal/` packages with inconsistent abstraction boundaries. To achieve the goal of making engine the authoritative core business layer, we need **strategic consolidation, interface standardization, and systematic migration** of business logic.

Additionally, the long‑term target state now explicitly includes **evolving the repository into a structured multi-module monorepo** to enable: (1) independent semantic versioning of the core engine, (2) faster iteration of API / web / CLI layers, (3) clearer contract boundaries, and (4) reduced accidental coupling. This document is augmented to outline that path (see Section 9 and Appendix C).

**Current State**: 🟡 **Partially Achieved** - Facade exists but lacks domain authority  
**Target State**: 🎯 **Engine as Single Source of Truth (Multi-Module Ready)** - All business logic consolidated under engine interfaces; repository partitioned into independent Go modules with clear dependency direction (presentation layers depend on engine, not vice versa).  
**Migration Effort**: **High** - Requires 6-8 phases of careful refactoring plus an activation phase for module extraction with maintained backward compatibility

**Adapter Evolution Update (Phase 5E Alignment)**: As observability endpoints shift to an adapter pattern, the same architectural approach will be applied to business integrations—most notably the current Colly-based fetching implementation. A future `fetcher` adapter boundary will extract `CollyFetcher` into a pluggable module while the engine consumes only a `Fetcher` interface. This prevents HTTP mechanics and library concerns from leaking into core orchestration and mirrors the telemetry adapter separation now planned for health/metrics endpoints.

---

## 1. Current Architecture Assessment

### 1.1 Package Distribution Analysis

| Package Location          | Responsibility                         | Business Logic Density | Engine Integration |
| ------------------------- | -------------------------------------- | ---------------------- | ------------------ |
| **`packages/engine/`**    | Orchestration, Facade                  | **Medium**             | ✅ Core            |
| **`internal/crawler/`**   | HTTP Fetching, Link Discovery          | **High**               | ❌ Isolated        |
| **`internal/processor/`** | Content Processing, Asset Handling     | **Very High**          | ❌ Isolated        |
| **`internal/assets/`**    | Asset Pipeline, Download, Optimization | **High**               | ❌ Isolated        |
| **`pkg/models/`**         | Domain Types, Configuration            | **Medium**             | 🟡 Partially       |

### 1.2 Current Engine Facade Capabilities

**✅ Successfully Implemented:**

- Unified lifecycle management (`Start`/`Stop`)
- Aggregated metrics and snapshot composition
- Pipeline coordination and resource management
- Rate limiting integration
- Checkpoint/resume functionality

**❌ Missing Core Business Capabilities:**

- **Content extraction strategy**: Hardcoded in `internal/processor`
- **Fetch protocol abstraction**: Tightly coupled to Colly in `internal/crawler`
- **Asset handling policies**: Isolated in `internal/assets`
- **Output format strategies**: Minimal interface implementation
- **Processing workflow configuration**: Scattered across internal packages

### 1.3 Architectural Debt & Coupling Issues

#### Critical Issue 1: Business Logic Fragmentation

```
Current Flow:
main.go → engine.Start() → pipeline → [BLACK BOX] → results

Hidden Dependencies:
pipeline → internal/crawler (Colly)
pipeline → internal/processor (HTML cleaning, markdown conversion)
processor → internal/assets (asset pipeline)
```

**Problem**: Core business decisions (what to crawl, how to process, where to output) are embedded in `internal/` packages that the engine cannot directly control or configure.

#### Critical Issue 2: Interface Boundaries Are Insufficient

The engine currently treats business components as implementation details rather than configurable strategies:

```go
// Current: Engine owns orchestration but not business logic
type Engine struct {
    pl       *pipeline.Pipeline  // Pipeline is concrete, not strategy
    limiter  RateLimiter        // Good abstraction
    rm       *Manager           // Good abstraction
}

// Missing: Business strategy injection
type Engine struct {
    fetcher     Fetcher          // How to retrieve content
    processor   Processor        // How to transform content
    outputSinks []OutputSink     // Where to send results
}
```

#### Critical Issue 3: Configuration Authority Split

Business configuration is split between:

- **Engine Config**: Infrastructure concerns (buffers, workers, limits)
- **Internal Configs**: Business rules (content selectors, asset handling, output formats)

This prevents presentation layers from configuring business behavior through the engine interface.

---

## 2. Strategic Analysis: Engine as Core Business Layer

### 2.1 Definition of "Core Business Layer"

For Ariadne, the core business layer should encapsulate:

1. **Content Acquisition Strategy** - How to fetch and discover content
2. **Processing Workflow** - How to clean, transform, and enrich content
3. **Output Generation** - How to format and deliver results
4. **Quality Policies** - Rate limiting, resource management, error handling
5. **Persistence Strategy** - Caching, checkpointing, resume behavior

**Success Criteria**: Presentation layers (CLI, potential TUI/API) should **only** need to:

- Configure business policies through engine interfaces
- Provide content sources (seed URLs, configurations)
- Consume structured results
- Monitor progress through unified snapshots

### 2.2 Current vs Target Architecture

#### Current Architecture (Post-Phase 4)

```
┌─────────────┐    ┌─────────────┐    ┌──────────────────┐
│    CLI      │───▶│   Engine    │───▶│   Pipeline       │
│ main.go     │    │   Facade    │    │  Orchestration   │
└─────────────┘    └─────────────┘    └─────────┬────────┘
                                                │
                          ┌─────────────────────┼─────────────────────┐
                          │                     ▼                     │
                    ┌──────────┐    ┌──────────────┐    ┌──────────────┐
                    │ Crawler  │    │  Processor   │    │   Assets     │
                    │(Colly)   │    │(HTML→MD)     │    │ (Download)   │
                    │internal/ │    │ internal/    │    │ internal/    │
                    └──────────┘    └──────────────┘    └──────────────┘
```

**Problems**:

- Business logic scattered in `internal/` packages
- Engine has no authority over business decisions
- Tight coupling to specific implementations (Colly, specific HTML processors)

#### Target Architecture (Phase 5+)

```
┌─────────────┐    ┌─────────────────────────────────────────┐
│    CLI      │───▶│            Engine Core                  │
│ main.go     │    │         (Business Layer)                │
└─────────────┘    └─────────────────┬───────────────────────┘
                                    │
                   ┌─────────────────┼─────────────────┐
                   │                 ▼                 │
           ┌───────────────┐  ┌─────────────┐  ┌─────────────┐
           │   Fetcher     │  │ Processor   │  │OutputSinks  │
           │ Interface     │  │ Interface   │  │Interface    │
           └───────────────┘  └─────────────┘  └─────────────┘
                   │                 │                 │
           ┌───────────────┐  ┌─────────────┐  ┌─────────────┐
           │ Colly Impl    │  │HTML→MD Impl │  │File/Stream  │
           │ Browser Impl  │  │NLP Impl     │  │DB/Queue     │
           │ API Impl      │  │Custom Impl  │  │Custom Impl  │
           └───────────────┘  └─────────────┘  └─────────────┘
```

**Benefits**:

- Engine owns all business policy decisions
- Pluggable implementations for different use cases
- Clean separation of business logic from infrastructure
- Unified configuration surface for presentation layers
- Consistent adapter pattern (fetchers, output sinks, telemetry endpoints) reducing churn when swapping implementations

---

## 3. Migration Strategy & Effort Analysis

### 3.1 Phase-by-Phase Migration Plan

#### Phase 5A: Interface Definition & Scaffolding (2-3 weeks)

**Goal**: Define core business interfaces without breaking existing functionality

**Tasks**:

1. Define `Fetcher` interface with Colly implementation
2. Define `Processor` interface with current HTML→Markdown implementation
3. Define enhanced `OutputSink` interface (already partially done)
4. Create interface-based engine constructor accepting strategies
5. Maintain backward compatibility with default implementations

**Success Metrics**:

- All existing tests pass
- New interfaces can be dependency-injected into engine
- Default behavior unchanged

#### Phase 5B: Crawler Migration (3-4 weeks)

**Goal**: Move `internal/crawler` business logic under engine control

**Tasks**:

1. Extract Colly wrapper as `packages/engine/crawler/colly/`
2. Implement `Fetcher` interface with configurable policies
3. Update pipeline to use injected `Fetcher` instead of direct crawler calls
4. Add fetch strategy configuration to engine config
5. Deprecate direct `internal/crawler` usage
6. Prepare adapter boundary for future alternative fetchers (browser-based, API-first) – align naming (`packages/engine/fetcher/colly`)

**Risks**:

- **High**: Colly integration complexity, potential request behavior changes
- **Medium**: Rate limiter coordination with fetch strategies
- **Low**: Link discovery algorithm preservation

#### Phase 5C: Processor Migration (4-5 weeks)

**Goal**: Consolidate content processing business logic under engine

**Tasks**:

1. Extract core processing algorithms from `internal/processor`
2. Create pluggable `Processor` interface with content transformation strategies
3. Migrate asset handling as processor plugins or separate strategy
4. Update pipeline to use injected processors
5. Provide processor composition (pipeline of processors)

**Risks**:

- **Very High**: Processor is 632 lines with complex HTML cleaning, asset integration
- **High**: Asset pipeline coupling, markdown conversion behavior preservation
- **Medium**: Configuration surface complexity for multiple processors

#### Phase 5D: Asset Strategy Integration (2-3 weeks)

**Goal**: Make asset handling a configurable business policy

**Tasks**:

1. Design asset handling as either processor plugin or separate strategy interface
2. Extract asset pipeline from `internal/assets` to engine-controlled strategy
3. Enable asset policy configuration (download/skip/optimize policies)
4. Update engine config to include asset handling preferences

#### Phase 5E: Output Strategy Consolidation (2 weeks)

**Goal**: Complete output sink strategy implementation (partially done in Phase 4)

**Tasks**:

1. Enhance existing OutputSink interface with policy configurations
2. Migrate any remaining output logic to sink implementations
3. Add output composition (multi-sink, conditional routing)
4. Update engine to accept configured sink strategies

#### Phase 5F: Configuration Unification (1-2 weeks)

**Goal**: Single configuration surface through engine

**Tasks**:

1. Consolidate all business configuration under engine config
2. Remove direct access to internal package configurations
3. Provide configuration validation and defaults
4. Update CLI to configure only through engine interface

**Success Criteria**: `main.go` should configure business behavior **only** through `engine.Config`, with no knowledge of internal implementation details.

#### Phase 5G (Activation): Multi-Module Extraction (1-2 weeks, conditional trigger)

**Goal**: Split the monolithic Go module into discrete, semantically versioned modules once engine interfaces stabilize (post 5F) and churn rate declines.

**Trigger Conditions** (all must be true):

1. Engine interface changes per week < 2 over a 3-week rolling window
2. Processor / Fetcher strategy contracts marked stable (no TODO:BREAKING markers)
3. Configuration consolidation complete; CLI only consumes `engine` public API
4. Backward compatibility test suite green against last two engine tags

**Planned Module Layout (initial):**

```
 /engine            (module: github.com/yourorg/ariadne/engine)
 /cli               (module: github.com/yourorg/ariadne/cli) – depends on engine
 /services/api      (module: github.com/yourorg/ariadne/api) – depends on engine
 /adapters/output/* (optional future modules if externalized)
 /shared/models     (folded progressively into engine/models; legacy kept temporarily)
```

**Tasks:**

1. Extract `packages/engine` to `/engine` root module (rename path if needed)
2. Move CLI entrypoint(s) to `/cli` with its own `go.mod` importing `engine`
3. Introduce `/services/api` (stub) with placeholder server referencing engine (future expansion)
4. Update import paths; add `replace` directives locally during transition
5. Introduce compatibility layer: mark legacy import paths with forwarding packages (deprecation notices) if externally consumed
6. Update CI to run matrix: (engine tests, cli tests, api tests) + integration harness
7. Introduce tag policy: tag engine releases (`vX.Y.Z`), optional independent CLI tagging (or sync initially)

**Risks:**
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Import Path Breakage | Medium | Add transitional forwarding packages & clear release notes |
| Dependency Drift | Low | Renovate or Dependabot for each module, plus CI module graph diff |
| Version Skew (CLI vs Engine) | Medium | Start with lockstep tagging for first 3 cycles |
| Increased CI Time | Low | Parallelize workflows; cache build per module |

**Success Metrics:**
| Metric | Target |
|--------|--------|
| Engine module tests isolated | 100% engine tests not importing CLI/API |
| Build graph direction | Only presentation modules depend on engine |
| Independent release cadence | CLI or API can release w/o engine tag (after stabilization) |
| Backward compatibility | No breaking API changes without MAJOR bump |

**Deferral Strategy:** If interface churn remains high after 5F, re-evaluate at 2-week intervals; do not prematurely fragment.

### 3.2 Effort & Risk Assessment

| Phase | Duration  | Risk Level | Complexity | Breaking Changes            |
| ----- | --------- | ---------- | ---------- | --------------------------- |
| 5A    | 2-3 weeks | Low        | Medium     | None (additive)             |
| 5B    | 3-4 weeks | High       | High       | Minimal (compatibility)     |
| 5C    | 4-5 weeks | Very High  | Very High  | Moderate (processor API)    |
| 5D    | 2-3 weeks | Medium     | Medium     | Low (asset policies)        |
| 5E    | 2 weeks   | Low        | Low        | None (mostly complete)      |
| 5F    | 1-2 weeks | Low        | Medium     | None (config consolidation) |

**Total Effort**: 14-19 weeks (3.5-4.5 months)  
**Peak Risk**: Phase 5C (Processor migration due to complexity)

### 3.3 Key Technical Challenges

#### Challenge 1: Processor Complexity

The `internal/processor` package is a 632-line monolith handling:

- HTML content extraction with multiple selector strategies
- Markdown conversion with plugin system
- Asset discovery and URL rewriting
- Content cleaning and optimization

**Mitigation Strategy**:

- Break processor into composable sub-interfaces
- Create migration shims during transition period
- Extensive testing during each sub-component extraction

#### Challenge 2: Colly Integration Complexity

Current crawler tightly integrates with Colly's callback system and maintains internal state.

**Mitigation Strategy**:

- Wrap Colly behavior in `Fetcher` interface without changing core logic initially
- Gradual abstraction of Colly-specific features
- Maintain existing rate limiting integration

#### Challenge 3: Asset Pipeline Integration

Assets are currently processed synchronously with content extraction, creating tight coupling.

**Mitigation Strategy**:

- Consider asset handling as separate concern from content processing
- Design asset policies as engine-level configuration
- Evaluate async vs sync asset processing trade-offs

---

## 4. Architectural Recommendations

### 4.1 Immediate Next Steps (Phase 5A)

#### 1. Define Core Business Interfaces

```go
// packages/engine/crawler/fetcher.go
type Fetcher interface {
    Fetch(ctx context.Context, url string) (*FetchResult, error)
    Discover(ctx context.Context, content []byte, baseURL *url.URL) ([]*url.URL, error)
    Configure(policy FetchPolicy) error
}

// packages/engine/processor/processor.go
type Processor interface {
    Process(ctx context.Context, raw *RawContent) (*ProcessedContent, error)
    Chain(next Processor) Processor  // For processor composition
}

// Enhanced from existing packages/engine/output/sink.go
type OutputSink interface {
    Write(result *models.CrawlResult) error
    Flush() error
    Close() error
    Name() string
    Configure(policy OutputPolicy) error  // Add policy configuration
}
```

#### 2. Strategy-Aware Engine Constructor

```go
type EngineStrategies struct {
    Fetcher     Fetcher
    Processors  []Processor  // Processing pipeline
    OutputSinks []OutputSink
}

func NewWithStrategies(cfg Config, strategies EngineStrategies) (*Engine, error)
func New(cfg Config) (*Engine, error)  // Uses default strategies
```

### 4.2 Long-Term Architecture Principles

#### Principle 1: Business Logic Consolidation

**All content acquisition, processing, and output policies should be configurable through engine interfaces.**

#### Principle 2: Strategy Pattern Adoption

**Core business functions (fetch, process, output) should be injectable strategies, not hardcoded implementations.**

#### Principle 3: Configuration Authority

**Engine config should be the single source of truth for all business policies. Internal packages should receive configuration, not define it.**

#### Principle 4: Graceful Migration

**Each phase should maintain backward compatibility with existing functionality while introducing new capabilities.**

---

## 5. Alternative Architectures Considered

### Alternative 1: Full Rewrite Approach

**Pros**: Clean architecture from scratch, optimal interface design  
**Cons**: High risk, extended timeline, potential functionality regression  
**Decision**: Rejected - Too disruptive for current development velocity

### Alternative 2: Plugin System Architecture

**Pros**: Maximum flexibility, runtime pluggability  
**Cons**: Increased complexity, potential performance overhead  
**Decision**: Deferred - Focus on compile-time strategy injection first

### Alternative 3: Microservice Decomposition

**Pros**: Independent scalability, technology diversity  
**Cons**: Operational complexity, network overhead, overkill for current scope  
**Decision**: Rejected - Single-binary goal remains valid

---

## 6. Risk Mitigation Strategies

### Technical Risks

#### Risk 1: Regression During Migration

**Mitigation**:

- Comprehensive test coverage before migration phases
- Feature flag system for new interfaces during transition
- Parallel implementation during critical phases (5B, 5C)

#### Risk 2: Performance Impact from Abstraction

**Mitigation**:

- Interface-based design with minimal indirection
- Benchmark key paths during migration
- Optimize hot paths after interface stabilization

#### Risk 3: Configuration Complexity Growth

**Mitigation**:

- Design configuration DSL for complex processing chains
- Provide sensible defaults and configuration templates
- Validation and error reporting for configuration issues

### Organizational Risks

#### Risk 1: Development Velocity Impact

**Mitigation**:

- Phase migration to maintain development capability
- Prioritize high-value interfaces first (Fetcher, then Processor)
- Maintain existing functionality during transition phases

#### Risk 2: Knowledge Transfer Requirements

**Mitigation**:

- Document interface contracts and migration patterns
- Create example implementations for each interface
- Maintain architectural decision records for future reference

---

## 7. Success Metrics & Validation

### Quantitative Metrics

- **Interface Coverage**: 90%+ of business logic accessible through engine interfaces
- **Configuration Consolidation**: Single engine config controls all business policies
- **Import Reduction**: Zero direct imports of `internal/*` packages outside engine
- **Module Boundary Integrity (post 5G)**: No reverse (presentation → engine) circular dependencies; engine has zero imports from `/cli`, `/services/*`
- **Version Stability**: Average time between engine minor releases increases after stabilization (indicates maturity)
- **Test Coverage Maintenance**: >90% test coverage maintained throughout migration
- **Performance Baseline**: <10% performance regression during migration

### Qualitative Validation

- **CLI Simplification**: `main.go` should configure business behavior only through engine
- **Extension Capability**: New output formats or processing strategies can be added without touching core engine
- **Testing Independence**: Business logic components can be unit tested independently of infrastructure

### Phase Completion Criteria

Each phase must satisfy:

1. **All existing tests pass** with no functional regression
2. **New interfaces are documented** with examples and contracts
3. **Backward compatibility maintained** through migration shims if needed
4. **Performance benchmarks stable** within acceptable tolerance
5. **Configuration surface simplified** from presentation layer perspective

---

## 8. Conclusion & Recommendation

### Current State Assessment: 🟡 Partial Success

Ariadne has successfully established the engine as an **orchestration layer** but not yet as a **core business layer**. The facade pattern provides lifecycle management and metric aggregation, but critical business logic remains fragmented across internal packages.

### Strategic Recommendation: 🚀 Proceed with Phased Migration

The analysis supports **proceeding with engine-centric consolidation** using the proposed 6-phase migration strategy. The benefits of unified business logic control, simplified presentation layer interfaces, and improved testability outweigh the implementation complexity.

### Critical Success Factors

1. **Manage Phase 5C (Processor Migration) carefully** - This is the highest-risk, highest-complexity phase
2. **Maintain backward compatibility** throughout migration to preserve development velocity
3. **Prioritize interface design** over implementation optimization initially
4. **Comprehensive testing strategy** to prevent regression during refactoring

### Expected Outcomes Post-Migration

- **Simplified Integration**: New presentation layers (TUI, API) can integrate through clean engine interfaces
- **Enhanced Extensibility**: Business strategies (fetchers, processors, outputs) can be developed and tested independently
- **Improved Maintainability**: Business logic centralized and clearly bounded
- **Better Testability**: Interface-driven architecture enables comprehensive unit and integration testing

**Next Action**: Initiate Phase 5A (Interface Definition & Scaffolding) with focus on `Fetcher` and enhanced `Processor` interface design. Begin collecting stabilization metrics to determine Phase 5G activation timing.

---

## 9. Multi-Module Monorepo Strategy (Augmentation)

### 9.1 Rationale

Moving to a multi-module monorepo decouples the **contract stability domain** (engine) from the **delivery velocity domain** (CLI, API, web). This supports:

- Independent semantic versioning
- Faster iteration & deployment of runtime surfaces
- Clearer architectural boundaries and dependency direction
- Reduced accidental coupling & easier dependency graph reasoning

### 9.2 Guiding Principles

1. Engine is the only module exposing core business contracts
2. Downstream modules must not duplicate domain model types—source of truth in `engine`
3. No module imports another's `internal/` packages (enforced via CI script)
4. Public surface area changes require release notes & semantic versioning discipline
5. Transitional forwarding allowed for one minor release window only

### 9.3 Module Dependency Graph (Target)

```
┌──────────┐     ┌──────────┐
│   CLI    │     │  API     │
└────┬─────┘     └────┬─────┘
     │                │
     └────────┬───────┘
          ▼
      ┌────────────┐
      │  ENGINE    │
      └────────────┘
```

Potential future optional modules (not immediate): `engine-plugins/*`, `adapters/output/*`.

### 9.4 Incremental Extraction Plan

| Step | Action                                                   | Output                     |
| ---- | -------------------------------------------------------- | -------------------------- |
| 1    | Freeze interface churn (label proposed breaking changes) | Stability window defined   |
| 2    | Carve out `/engine` module (`go.mod`)                    | Independent build          |
| 3    | Adjust imports; add `replace` for local dev              | Transitional build success |
| 4    | Move CLI entrypoint → `/cli` module                      | Separate binary build      |
| 5    | Introduce API skeleton (even if minimal)                 | Future service placeholder |
| 6    | Add CI matrix & module boundary lint                     | Automated enforcement      |
| 7    | First dual release: tag engine + CLI                     | Baseline version alignment |
| 8    | Remove forwarding shims / finalize                       | Clean boundary             |

### 9.5 CI/CD Adjustments

- Add workflow matrix: `{module: engine|cli|api}` running `go test ./...`
- Boundary guard job: script parses `go list -deps` ensuring no reverse dependency
- Release workflow triggers on tags matching `engine/v*` and `cli/v*`
- Contract diff job (engine only) using `apidiff` against last released tag

### 9.6 Versioning & Tagging Policy

- Tag format: `engine/vMAJOR.MINOR.PATCH`, `cli/vMAJOR.MINOR.PATCH` (namespaced tags avoid collision)
- Initial lockstep: `engine/v1.0.0` + `cli/v1.0.0`; divergence allowed after 3 stable cycles
- Changelog grouped by module; root `CHANGELOG.md` links to per-module logs

### 9.7 Local Developer Workflow

```
git clone .../ariadne
cd ariadne/cli
go run ./cmd/ariadne --config ../examples/config.yaml
```

During transition:

```
replace github.com/yourorg/ariadne/engine => ../engine
```

### 9.8 Risk & Mitigation Snapshot

| Risk                   | Mitigation                                                              |
| ---------------------- | ----------------------------------------------------------------------- |
| Dual tagging confusion | Namespaced tags + release templates                                     |
| Orphaned shared code   | Consolidate into engine or explicit shared internal utils inside engine |
| Drift in domain models | Single source in engine; lint disallows duplicate type declarations     |
| Over-fragmentation     | Limit to 3 modules initially; reevaluate before adding more             |

### 9.9 Exit Criteria (Multi-Module Considered Successful)

1. Independent engine release executed without rebuilding CLI/API (beyond re-consume updated module)
2. Contract diff job green for two consecutive minor releases
3. Reduction in average PR size for CLI/API (measure after 4 weeks)
4. No direct imports of legacy `internal/*` by CLI/API (enforced)

### 9.10 Defer / Revert Strategy

If fragmentation overhead > productivity gain (measured by slowed cycle time >15%), pause further module additions and keep engine stabilization work inside monorepo without further splitting.

---

---

## Appendix A: Current Package Dependency Analysis

```
main.go
├── packages/engine (✅ Good - Facade pattern)
    ├── packages/engine/pipeline (✅ Migrated)
    ├── packages/engine/ratelimit (✅ Migrated)
    ├── packages/engine/resources (✅ Migrated)
    └── packages/engine/models (✅ Migrated)

internal/ (❌ Business logic isolation)
├── crawler/ (❌ High coupling - Colly integration)
├── processor/ (❌ Very high complexity - 632 lines)
├── assets/ (❌ Moderate coupling - Asset pipeline)
└── output/ (✅ Mostly resolved - OutputSink interface exists)

pkg/models (🟡 Partially migrated - Some legacy dependencies remain)
```

## Appendix B: Interface Design Examples

### Fetcher Interface (Phase 5A Priority)

```go
type FetchResult struct {
    URL      *url.URL
    Content  []byte
    Headers  map[string]string
    Status   int
    Links    []*url.URL
    Assets   []*AssetReference
    Metadata map[string]interface{}
}

type FetchPolicy struct {
    UserAgent     string
    RequestDelay  time.Duration
    Timeout       time.Duration
    MaxRetries    int
    RespectRobots bool
    FollowRedirects bool
}

type Fetcher interface {
    Fetch(ctx context.Context, url string) (*FetchResult, error)
    Configure(policy FetchPolicy) error
    Stats() FetcherStats
}
```

### Processor Interface (Phase 5C Priority)

```go
type ProcessingContext struct {
    SourceURL     *url.URL
    ContentType   string
    Configuration ProcessingPolicy
}

type ProcessingPolicy struct {
    ContentSelectors []string
    CleaningRules    []string
    AssetHandling    AssetPolicy
    OutputFormat     string
}

type Processor interface {
    Process(ctx context.Context, raw *FetchResult, pc ProcessingContext) (*models.Page, error)
    Chain(next Processor) Processor
    Validate(policy ProcessingPolicy) error
}
```

This analysis provides the foundation for Phase 5 planning and execution, with clear technical guidance and risk mitigation strategies for transforming the engine into Ariadne's authoritative core business layer.

---

## Appendix C: Proposed Initial Multi-Module Layout & Responsibilities

| Module   | Responsibility                                                          | Public Surface                                        | Depends On                             |
| -------- | ----------------------------------------------------------------------- | ----------------------------------------------------- | -------------------------------------- |
| `engine` | Core business logic (fetch/process/output strategies, policies, models) | Stable interfaces, config structs, strategy contracts | Standard lib + vetted third-party libs |
| `cli`    | User-facing command-line, config parsing, UX, invocation orchestration  | CLI commands, flags (no business logic)               | `engine`                               |
| `api`    | (Future) HTTP/gRPC service exposing engine operations                   | Handlers, DTO translation layer                       | `engine`                               |

Transitional legacy areas (`internal/*`) will be incrementally either merged into `engine` or removed. No new code should be added under `internal/` after Phase 5C start—create strategies instead.

---
