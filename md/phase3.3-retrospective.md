# Phase 3.3 Retrospective – Resource Management & Resilience

Status: Complete  
Date: September 26, 2025  
Track: Concurrency & Performance (Phase 3)  
Scope: Caching, Spillover (Disk), In-Flight Memory Guardrails, Checkpointing

---

## 1. Emotional / Team Sentiment ("Feelings")

| Theme           | Sentiment | Notes                                                                                               |
| --------------- | --------- | --------------------------------------------------------------------------------------------------- |
| Confidence      | High      | Existing TDD safety net let us integrate a cross-cutting component (resource manager) without fear. |
| Complexity Load | Moderate  | Balancing cache semantics + pipeline metrics required deliberate API boundaries.                    |
| Momentum        | Positive  | Rapid iteration cycles; minimal backtracking after initial design decisions.                        |
| Risk Perception | Lowered   | Early checkpointing + spill files create psychological safety for future long-running crawls.       |
| Friction Points | Mild      | External network dependency in legacy asset tests caused occasional noise in full-suite validation. |

---

## 2. What We Achieved

### Functional Outcomes

- Introduced a unified `resources.Manager` handling:
  - LRU in-memory cache with transparent retrieval
  - Disk spillover persistence (JSON blobs) for evicted pages
  - Concurrency guard (`MaxInFlight`) to cap simultaneous extraction memory pressure
  - Asynchronous checkpoint journaling of processed URLs
- Integrated extraction-stage cache lookups (cache hit short-circuit path) + metrics recording (new `cache` stage) without inflating total processed counts.
- Ensured safe lifecycle: background checkpoint goroutine terminates cleanly; best-effort strategy (drop when buffer full) prevents shutdown panics.
- Added URL propagation across all `CrawlResult` sources to unify checkpoint writing and future resumability logic.

### Quality & Safety

- All resource-related behavior validated with dedicated unit tests (cache hit/miss, eviction → spill → recovery, checkpoint flush, semaphore acquisition/release).
- Race detector clean for pipeline + resource manager combination.
- Integration tests confirm cache effectiveness (first fetch extracts, second served from cache) and capture spill existence.

### Developer Experience Improvements

- Clear separation of responsibilities: pipeline concerns (flow, retries, rate limits) vs. resource management (state, durability, memory discipline).
- Simplified future incremental additions (e.g., resume-from-checkpoint, persisted domain state for rate limiter) by establishing a persistence primitive.

---

## 3. What Went Well (Processes / Practices)

| Practice                  | Impact                                | Evidence                                                                       |
| ------------------------- | ------------------------------------- | ------------------------------------------------------------------------------ |
| TDD-first for new package | Reduced integration risk              | Resource unit tests written before pipeline wiring.                            |
| Incremental integration   | Avoided large destabilizing diff      | Added manager, then cache hits, then checkpointing.                            |
| Narrow surface area       | Easier reasoning & review             | `Manager` exposes a small API (Acquire/Release, Store/Get, Checkpoint, Close). |
| Metrics discipline        | Prevented double counting regressions | Explicit exclusion of `cache` stage from global totals.                        |
| Early failure simulation  | Exposed error-handling paths          | Tests forced handling of cache store/load failures.                            |
| Conservative concurrency  | Eliminated goroutine leaks            | No fire-and-forget for checkpoint when channel full (dropped intentionally).   |

---

## 4. Challenges & How We Addressed Them

| Challenge                                 | Description                                      | Mitigation                                               | Outcome                               |
| ----------------------------------------- | ------------------------------------------------ | -------------------------------------------------------- | ------------------------------------- |
| Double counting processed metrics         | Cache hits could inflate totals                  | Added conditional totals increment (skip `cache` stage)  | Accurate aggregate metrics            |
| Shutdown ordering & channel safety        | Potential race sending checkpoints after close   | Removed async fallback send; switched to drop-on-full    | Deterministic shutdown                |
| External network flakiness (legacy tests) | HTTPBIN dependency causing intermittent failures | Documented as tech debt (Phase 2 relic)                  | Does not block Phase 3 completion     |
| Spill recovery correctness                | Needed deterministic file naming & retrieval     | Hash-based filename + JSON marshal/unmarshal tests       | Verified by test harness              |
| Semaphore fairness/starvation             | Potential for long waits under load              | Simple buffered channel; revisit fairness only if needed | Adequate for current throughput goals |

---

## 5. Metrics / Observations (Qualitative)

While we did not integrate live telemetry yet, testing produced these qualitative insights:

- Cache hit path eliminates permit acquisition & extraction latency (micro-optimization for repeated URLs or retries).
- Spill file creation observed only when cache capacity < active unique pages; recovery path tested end-to-end.
- Checkpoint flushing interval (default fallback 50ms) strikes a balance between I/O churn and durability.

---

## 6. Technical Debt / Follow-Ups

| Item                                                               | Priority | Rationale                                        |
| ------------------------------------------------------------------ | -------- | ------------------------------------------------ |
| Replace external asset downloader HTTP dependency with mock server | High     | Stabilize full-suite tests under CI & `-race`.   |
| Resume-from-checkpoint implementation                              | Medium   | Leverage log for crash/stop recovery.            |
| Structured metrics export (cache hit ratio, spill count)           | Medium   | Feed into future engine/TUI observability.       |
| Configurable spill pruning / TTL                                   | Low      | Prevent unbounded disk growth in massive crawls. |
| Unified snapshot combining limiter + resources                     | Medium   | Single introspection surface for engine facade.  |

---

## 7. Lessons Learned

- Introduce persistence primitives (checkpoint files) before building resumability logic—low cost, high option value.
- Keep adaptive systems (rate limiter) and stateful memory controls (resource manager) orthogonal; compositions stay simpler.
- Provide explicit metrics boundaries to prevent subtle accounting drift when adding bypass paths (cache hits).
- Favor dropping non-critical telemetry/checkpoint entries under pressure rather than risking shutdown instability.

---

## 8. Readiness for Next Steps

With Phase 3 complete (pipeline + adaptive throttling + resource control), we are positioned to:

1. Implement the Engine facade consolidating lifecycle + snapshots.
2. Layer output generation (Phase 4) without tangling with low-level memory concerns.
3. Add resumability features using existing checkpoint artifacts.
4. Improve observability by aggregating limiter + resource metrics into a unified snapshot.

---

## 9. Retrospective Summary (TL;DR)

Phase 3.3 delivered resilient resource handling that protects memory, accelerates repeated work, and lays groundwork for resumable operations—all without destabilizing prior phases. TDD, incremental integration, and explicit metric discipline were decisive success factors. Remaining friction (external network tests) is isolated and scheduled for cleanup.

---

## 10. Actionable Next Sprint Picks

1. Mock-based replacement for external asset download tests (stabilize CI) – High
2. Engine facade (P1/P2) start – High
3. Unified snapshot struct (limiter + resources) – Medium
4. Optional: spill file pruning + size reporting – Low

---

_End of Phase 3.3 Retrospective_
