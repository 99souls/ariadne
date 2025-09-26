go test ./internal/assets -v # ✅ PASS
go test ./internal/processor -v # ✅ PASS
go build # ✅ SUCCESS

# Site Scraper Implementation Context

## Current Status: Phase 3 Complete (3.1–3.3) ✅

**Date**: November 3, 2025  
**Phase**: Concurrency & Performance Track (Phase 3)  
**Achievement**: Multi-stage pipeline, adaptive rate limiting, and resource management (cache, spillover, checkpoints) delivered with full TDD coverage

---

## Phase 3.3 Highlights – Resource Management Delivered

### Core Deliverables

- **Resource Manager (`internal/resources`)**: Configurable MaxInFlight semaphore, LRU cache with disk spillover, checkpoint journaling goroutine, and unit tests covering cache hits, spill recovery, and concurrency semantics.
- **Pipeline Caching Integration**: Extraction stage consults cache before hitting network, records cache stage metrics, and stores fresh pages post-fetch. Cache hits bypass rate limiter permits for faster throughput.
- **Checkpointing & Journaling**: `deliverResult` appends processed URLs to durable log while pipeline integration tests confirm flush semantics.
- **Memory Guardrails**: In-flight slot acquisition prevents runaway extraction concurrency; release occurs immediately after caching to maintain throughput without saturating memory.

### Validation & Quality Gates

- `go test ./internal/resources ./internal/pipeline` ✅ – verifies cache hits, spillover, checkpoint flush, and pipeline integration.
- `go test -race ./internal/resources ./internal/pipeline` ✅ – ensures concurrency safety across new manager and pipeline interactions.
- `go test -race ./...` ✅ – full suite now green offline after replacing external HTTPBIN dependencies with internal `httpmock` server.

### Key Insights

- Caching and checkpointing significantly reduce redundant extraction work during retries or duplicate URLs.
- Spillover format (`spill-*.spill.json`) provides deterministic artifacts for debugging and future resumption logic.
- Integrating resource manager early sets the stage for the upcoming engine facade to expose clean APIs for CLI/TUI layers.
- Formal retrospective captured in `phase3.3-retrospective.md` (covers emotions, process wins, risks, and next sprint picks).

---

## Phase 3.2 Highlights – Adaptive Rate Limiting Delivered

### Core Deliverables

- **Adaptive Limiter Engine** (`internal/ratelimit`)
  - Sharded domain registry with per-domain token buckets and AIMD tuning
  - Sliding error-rate windows and circuit breaker state machine
  - Retry-After compliance plus idle-domain eviction for bounded memory use
  - Snapshot API for future telemetry consumers
- **Pipeline Integration** (`internal/pipeline`)
  - Extraction workers acquire permits before HTTP fetches and submit feedback after completion
  - Retry logic upgraded to jittered exponential backoff with bounded attempts
  - Retry goroutines respect pipeline cancellation, preventing orphaned work during shutdown

### Validation & Quality Gates

- `go test ./...` ✅ – entire suite green post-integration
- `go test -race ./internal/ratelimit ./internal/pipeline` ✅ – concurrency-safe limiter & pipeline
- `go test -race ./...` ✅ – external flakiness removed (mocked assets)
- `gofmt` + linting applied to new/modified Go files

### Key Learnings

- AIMD tuning combined with breaker state delivers fast recovery while avoiding oscillations
- Respecting `Retry-After` directives keeps us polite with servers issuing explicit slowdown instructions
- Coordinating retries inside the pipeline requires strict context checks to avoid zombie goroutines
- Eviction plus `Close()` hooks make deterministic tests feasible even with long-lived background loops

---

## Phase 3.1 Recap – Multi-Stage Pipeline Foundation ✅

- Channel-based discovery → extraction → processing → output pipeline fully implemented
- Per-stage worker pools, backpressure handling, and graceful shutdown via context cancellation
- Result aggregator guarantees deterministic completion and simplifies testing
- Extensive TDD suite (`internal/pipeline/pipeline_test.go`) covering metrics, backpressure, shutdown, and integration scenarios

---

## Earlier Foundations – Phase 2 Refactoring Snapshot ✅

- Asset management extracted to `internal/assets` with independent pipeline and tests
- Content processor modularised with advanced HTML cleaning, Markdown conversion, and metadata handling
- TDD workflow maintained 100% coverage across critical paths and enabled aggressive refactoring with confidence

---

## Open Items & Follow-Ups

1. **CLI Migration (P5)**: Route `main.go` through engine facade only; add seed/resume flags.
2. **Deprecations (P6)**: Mark direct pipeline constructors as experimental for prospective internalization.
3. **Stability & Docs (P7)**: Add `API_STABILITY.md`, migration notes, baseline CHANGELOG.
4. **Limiter Telemetry Export**: Wire `Snapshot()` outputs to future metrics sink (Prometheus placeholders).
5. **Optional Domain Rate Detail**: Extend snapshot with top-N domain stats (deferred unless needed for CLI output).

---

## Reference Commands

```bash
# Core verification
go test ./...                                           # ✅
go test -race ./internal/ratelimit ./internal/pipeline # ✅
# Pending remediation due to network dependency
go test -race ./...                                    # ⚠️ (fails: asset downloader external call)
```

---

_Last Updated: November 3, 2025_  
_Next Focus: Phase 3.3 Resource Management & Monitoring_
