# Phase 3.2 Architecture & Implementation Plan – Intelligent Rate Limiting

Status: ✅ Completed (Phase 3.2 Delivered)
Owner: Concurrency & Performance Track
Target Completion: Phase 3.2 (Adaptive Rate & Reliability Controls)

---

## 1. Objectives & Success Criteria

**Primary Goal**: Prevent overwhelming target servers while maximizing safe throughput across domains.

**Implementation Summary**

- `internal/ratelimit` package now provides a sharded adaptive limiter with AIMD token buckets, sliding error windows, and circuit breaker state per domain.
- Pipeline extraction workers acquire permits, respect retry-after directives, and feed real-time feedback for adaptive tuning.
- Retry orchestration now uses jittered exponential backoff and cooperates with pipeline cancellation to avoid runaway goroutines.
- Comprehensive unit and integration tests cover limiter components and pipeline behavior; full suite passes with `go test ./...`.
- Race detector passes for limiter/pipeline packages; global `go test -race ./...` currently blocked by legacy asset downloader tests hitting external network (documented follow-up).

| Capability             | Description                                                                         | Success Metric                                                                        |
| ---------------------- | ----------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| Adaptive Rate Limiting | Dynamically adjust request pacing per domain using observed latency & error signals | Stable throughput without sustained 429/5xx bursts (>90% success after stabilization) |
| Per-Domain Isolation   | Each domain has independent budget and breaker state                                | One misbehaving domain doesn’t throttle others                                        |
| Circuit Breaker        | Fast detect & pause on overload/error patterns                                      | Open state entered within 5s of overload & recovers automatically                     |
| Retry with Backoff     | Intelligent exponential backoff + jitter for transient failures                     | Reduced duplicate failure storms; <3 retry attempts per transient failure avg         |
| Metrics & Visibility   | Real-time introspection of limiter decisions                                        | Exposed counters & gauges available to pipeline                                       |

---

## 2. Scope (In / Out)

In-Scope:

- Request admission control before Extraction (HTTP fetch) stage
- Per-domain adaptive pacing, circuit breaker, retry orchestration
- Configurable strategies via `ScraperConfig`
- Telemetry (counters, moving averages, percentile latencies, error rates)
- TDD-first implementation with deterministic simulation harness

Out-of-Scope (Deferred):

- Global distributed coordination across processes
- ML-based predictive scheduling
- Persistent historical model of domains
- Adaptive concurrency scaling (will integrate later)

---

## 3. Assumptions & Constraints

- Existing multi-stage pipeline (Discovery → Extraction → Processing → Output) is stable.
- Extraction workers perform network I/O; they will block awaiting rate limiter tokens.
- We can wrap HTTP client calls to collect latency + status metrics.
- Domains identified via `url.Host` (normalized lowercase; remove default ports 80/443).
- Short-lived crawl sessions (< hours) so in-memory statistics suffice.
- Need low allocation overhead (avoid per-request heap churn).

Constraints:

- Must be race-free under high concurrency.
- No global mutex bottleneck; per-domain structures + sharded maps.
- Avoid unbounded memory growth with many unique domains (LRU eviction for inactive domain state).

---

## 4. Baseline Architecture Reference

Current pipeline utilizes buffered channels and worker pools. The Extraction stage boundary is the _control point_ for rate limiting. We will interpose an admission layer _before_ initiating network fetch.

---

## 5. High-Level Design Overview

```
[Discovery] → urlChan → [Extraction Worker]
                 | (Acquire Permit)
                 v
        +----------------------+
        | Rate Limit Orchestrator |
        |  - DomainRegistry       |
        |  - Adaptive Algorithms  |
        |  - Circuit Breakers     |
        |  - Retry Scheduler      |
        +-----------+-------------+
                        | (Granted / Deferred / Dropped)
                        v
                  HTTP Fetch → Response Metrics → Feedback → Orchestrator
```

Decision Loop:

1. Extraction worker requests permit (domain).
2. Orchestrator consults domain state: breaker, tokens, adaptive delay.
3. Worker either waits (sleep/delay), is denied (breaker open), or proceeds.
4. Post-response metrics (latency, status code, error) fed back.
5. Domain adaptive model updates rate + breaker.

---

## 6. Core Components & Interfaces

### 6.1 Public Interfaces (Proposed)

```go
// RateLimiter mediates request admission per domain.
type RateLimiter interface {
    Acquire(ctx context.Context, domain string) (Permit, error) // blocks or returns error (breaker open / ctx cancel)
    Feedback(domain string, fb Feedback)                       // async safe
    Snapshot() LimiterSnapshot                                 // aggregated metrics
}

type Permit interface { Release() }

type Feedback struct {
    StatusCode int
    Latency    time.Duration
    Err        error
    // raw size, headers for Retry-After may be added later
    RetryAfter time.Duration // populated if server sent directive
}
```

### 6.2 Implementation Types

- `AdaptiveRateLimiter` (concrete): maintains shard array of `domainState` maps.
- `domainState`:
  - Token bucket parameters (capacity, tokens, lastRefill, fillRate/sec)
  - MovingLatency (EWMA + P95 ring buffer)
  - ErrorWindow (sliding window counters: total, 5xx, 429, network errors)
  - CircuitBreaker (state machine + state timestamps + half-open probe budget)
  - RetryBackoff policy config
  - Recent activity timestamp (for LRU eviction)

### 6.3 Circuit Breaker State

States: Closed → Open → HalfOpen → Closed.
Transitions:

- Closed→Open: (errorRate >= threshold AND sampleSize >= minSamples) OR consecutiveFail >= maxConsecutive.
- Open→HalfOpen: after `openTimeout` passes.
- HalfOpen→Closed: first `successesForRecovery` successful probes.
- HalfOpen→Open: first failure.

### 6.4 Retry Manager

Retry logic mostly stateless; we compute next delay given attempt number + jitter. Integrates with Acquire:

- Worker encountering transient error calls Feedback, then decides to re-enqueue URL with attempt++ & scheduled time (Delay scheduling via min-heap or simple time.After channel). For 3.2, we adopt a simpler approach: immediate requeue after sleeping backoff in the worker (bounded concurrency maintained). Later we can externalize to scheduling stage.

---

## 7. Adaptive Algorithms

### 7.1 Token Bucket with Dynamic Fill Rate

Initial fill rate = `config.InitialRPS` (per domain). Each successful response with acceptable latency adjusts target RPS.

Adaptive Strategy (AIMD variant):

- Success w/ latency < latencyTarget: `fillRate += alpha` (small additive increase)
- Elevated latency (>= latencyDegradeFactor _ latencyTarget) OR 429/503: `fillRate _= beta` (multiplicative decrease, beta in (0,1))
- Hard floor: `fillRate >= minRPS`; Hard ceiling: `<= maxRPS`.

Latency Target: dynamic EWMA anchored to initial expectation. `EWMA_new = (1-λ)*EWMA_old + λ*observed` with λ ≈ 0.2.

### 7.2 Error Rate Computation

Sliding window length W (e.g., 30s) with bucket granularity G (e.g., 2s). Maintain ring of counts. Error rate = (errors / total) in window.

### 7.3 Circuit Breaker Trip Condition

If (errorRate >= errorRateThreshold AND totalSamples >= minSamples) OR consecutive 429/5xx >= consecutiveThreshold.

### 7.4 Jittered Exponential Backoff

`base = config.RetryBaseDelay` (e.g., 200ms). Delay_n = min(maxDelay, base \* 2^(n-1)). Apply Full Jitter: sleep random(0, Delay_n).

### 7.5 Retry-After Header Compliance

If `Feedback.RetryAfter > 0`, override next Acquire earliest time using that delay (store per domain `nextEarliest` timestamp). Circuit breaker not tripped solely by a directed Retry-After; instead treat as soft throttle (also triggers multiplicative decrease once).

---

## 8. Data Structures

```go
type domainState struct {
    mu sync.Mutex
    tokens float64
    lastRefill time.Time
    fillRate   float64 // tokens per second (adaptive)

    latencyEWMA float64
    latencyP95  *p95Estimator // fixed-size ring buffer / t-digest later

    window *slidingWindow // counts

    breaker circuitBreakerState
    nextEarliest time.Time // due to Retry-After
    lastActivity time.Time
}
```

Sharded registry: `[]*domainShard` (power-of-two length). Hash(domain) → shard. Each shard holds `map[string]*domainState` guarded by RWMutex. Minimizes contention.

---

## 9. Control Flow (Acquire + Feedback)

Sequence (Acquire):

1. Compute shard; read-lock to fetch domainState (create if absent – upgrade to write lock). Update lastActivity.
2. Lock domainState.mu.
3. If breaker open and now < reopenTime → return `ErrCircuitOpen`.
4. Refill tokens: `elapsed * fillRate` (cap at capacity). If tokens < 1, compute wait = (1 - tokens)/fillRate.
5. Respect `nextEarliest` if in future (may extend wait).
6. If wait > 0: unlock domainState.mu, sleep (bounded by ctx). Loop (with jitter if long wait) or return context error.
7. Deduct 1 token, return Permit{releaseFn increments nothing; Release may adjust tokens if cancellation before fetch? (Simpler: Release no-op)}.

Feedback:

1. Lookup domainState; lock.
2. Update latencyEWMA, ring buffer.
3. Update sliding window counts.
4. If status 2xx/3xx adjust fillRate via AIMD rules.
5. If 429/5xx apply multiplicative decrease & increment consecutive fails.
6. Evaluate breaker transitions.
7. If Retry-After present set nextEarliest.
8. Unlock.

---

## 10. Configuration Additions (`ScraperConfig`)

```go
type RateLimitConfig struct {
    Enabled             bool
    InitialRPS          float64 // default starting rate per domain
    MinRPS              float64
    MaxRPS              float64
    TokenBucketCapacity float64 // burst allowance

    AIMDIncrease        float64 // alpha
    AIMDDecrease        float64 // beta multiplier (e.g., 0.5)
    LatencyTarget       time.Duration // initial target
    LatencyDegradeFactor float64 // e.g., 2.0

    ErrorRateThreshold  float64 // e.g., 0.5
    MinSamplesToTrip    int
    ConsecutiveFailThreshold int
    OpenStateDuration   time.Duration // base open timeout
    HalfOpenProbes      int // successes to close

    RetryBaseDelay      time.Duration
    RetryMaxDelay       time.Duration
    RetryMaxAttempts    int

    StatsWindow         time.Duration // e.g., 30s
    StatsBucket         time.Duration // e.g., 2s
    DomainStateTTL      time.Duration // idle eviction
    Shards              int // power of two (default 16)
}
```

---

## 11. Metrics & Observability

Expose through `LimiterSnapshot`:

- Global: totalRequests, throttledRequests (wait>threshold), denied (breaker), retries, openCircuits, halfOpenCircuits
- Per-domain sample (top N active domains):
  - domain, fillRate, tokens, avgLatency, p95Latency, errorRate, state, consecutiveFails, nextEarliest - now (ms)

Implementation: Snapshot iterates shards; collects stats (bounded by configurable cap to avoid large payloads). Provide streaming channel or periodic logger integration.

Add Prometheus-style counters/gauges later (not mandatory in 3.2 minimal). Provide internal struct now to allow future exporter.

---

## 12. Error Handling & Classification

Transient (retry): timeouts, temporary network errors, 429, 502, 503, 504.
Permanent (no retry): 400, 401, 403, 404 (unless config overrides), content parsing errors (handled downstream, not by limiter).
Circuit Breaker increments only on transient server overload patterns (5xx, 429) + network errors.

---

## 13. Concurrency & Safety

- Sharded map reduces contention.
- Each domain uses its own mutex; Acquire waiting uses unlocked sleep loops to avoid convoying.
- Feedback is lock-minimized.
- Eviction pass (goroutine) runs every `DomainStateTTL/2`: collects idle domain keys & deletes them (only if not active; double-check lastActivity).
- All time-dependent operations reference monotonic clock (`time.Since` / `time.Now()` captured once per loop where possible).

---

## 14. Testing Strategy (TDD Plan)

Incremental TDD Sequence:

1. Domain normalization utility tests.
2. Token bucket basic: refill, capacity cap, waiting logic.
3. AIMD adjustments: increase on fast success, decrease on throttling statuses.
4. Sliding window error rate computations (deterministic buckets with mocked time provider).
5. Circuit breaker transitions (Closed→Open; Open→HalfOpen; HalfOpen→Closed/Open) via simulated feedback.
6. Retry backoff sequence + jitter bounds.
7. Acquire respects Retry-After (injected feedback).
8. Eviction removes inactive states without affecting active ones.
9. Concurrency stress test: multiple goroutines Acquire+Feedback; assert no race (run with `-race`).
10. Integration test: simulated HTTP server returning scripted sequence (200, 200 (slow), 429, 200...) verifying fillRate trajectory and open circuit timing.
11. Pipeline integration test: injecting limiter causing controlled slowdown (observe extraction worker wait durations > threshold).

Use a controllable clock interface:

```go
type Clock interface { Now() time.Time; Sleep(time.Duration) }
```

Real clock + fake clock for tests (enables fast simulation of bucket refill and breaker timers).

---

## 15. Implementation Phases

Phase 3.2.a: Scaffolding & Interfaces (no adaptive logic yet) – pass compile.
Phase 3.2.b: Token bucket + basic Acquire tests.
Phase 3.2.c: Feedback latency & AIMD adjustments.
Phase 3.2.d: Error window + breaker transitions.
Phase 3.2.e: Retry logic + integration harness.
Phase 3.2.f: Eviction + snapshot API.
Phase 3.2.g: Pipeline wiring & pipeline tests extension.
Phase 3.2.h: Documentation & metrics instrumentation.

---

## 16. Risks & Mitigations

| Risk                                    | Impact                | Mitigation                                                              |
| --------------------------------------- | --------------------- | ----------------------------------------------------------------------- |
| Over-throttling due to transient spikes | Throughput drops      | Use multiplicative decrease only on definite overload signals (429/5xx) |
| High contention on shard maps           | Latency in Acquire    | Default 16 shards; allow config tuning                                  |
| Memory growth for many domains          | OOM risk              | TTL-based eviction & cap on max domain states                           |
| Time drift or test flakiness            | Unstable tests        | Abstract clock; deterministic tests                                     |
| Starvation under high load              | Some domains dominate | Future: fairness scheduling (not required now)                          |

---

## 17. Future Extensions (Beyond 3.2)

- Adaptive concurrency (vary worker pool size per domain).
- Predictive pacing using historical latency distributions (PID controller).
- Integration with robots.txt crawl-delay directives.
- Prometheus exporter & live dashboard.
- Configurable strategies (AIMD vs. gradient vs. leaky bucket).

---

## 18. Acceptance Checklist

- [x] Interfaces defined (`RateLimiter`, `Permit`, `Feedback`, `LimiterSnapshot`)
- [x] Token bucket per-domain with refill logic
- [x] Adaptive AIMD fill rate adjustments
- [x] Sliding window error metrics
- [x] Circuit breaker full state machine
- [x] Retry logic with exponential backoff + jitter
- [x] Retry-After compliance
- [x] Idle domain eviction
- [x] Snapshot metrics API
- [x] Integration tests with pipeline
- [x] Documentation updates (plan.md Phase 3.2 marked complete; intermediate context refreshed)
- [ ] All new tests pass with `-race` _(blocked by legacy asset downloader integration tests that rely on external HTTPBIN; limiter/pipeline suites succeed under `-race`)_

---

## 19. Minimal Code Skeleton (Illustrative Only)

```go
// internal/ratelimit/limiter.go
package ratelimit

type RateLimiter interface {
    Acquire(ctx context.Context, domain string) (Permit, error)
    Feedback(domain string, fb Feedback)
    Snapshot() LimiterSnapshot
}

type Feedback struct { StatusCode int; Latency time.Duration; Err error; RetryAfter time.Duration }

type Permit interface { Release() }
```

---

## 20. Integration Points

Pipeline Extraction Worker Pseudocode:

```go
permit, err := limiter.Acquire(ctx, domain)
if err != nil { /* check circuit open vs ctx canceled -> maybe enqueue for later or drop */ }
start := time.Now()
resp, err := httpClient.Do(req)
lat := time.Since(start)
fb := Feedback{Latency: lat}
if err != nil { fb.Err = err } else { fb.StatusCode = resp.StatusCode; /* parse Retry-After */ }
limiter.Feedback(domain, fb)
permit.Release()
```

Retries: on transient failure + attempts < max, compute delay, sleep, retry; else propagate failure result downstream.

---

## 21. Performance Considerations

- Acquire fast path: domain state in cache, tokens available → O(1) with one mutex.
- Feedback path updates O(1) structures.
- Sliding window: ring of fixed size; no dynamic allocation per update.
- Estimating p95 via small ring buffer (configurable sample size, e.g., 64) for low overhead.

---

## 22. Open Questions (To Validate Early)

1. Should retries be centrally scheduled instead of in worker goroutine? (Deferring to simplicity now.)
2. Should we expose blocking time metrics per Acquire? (Likely yes; add later if needed.)
3. Do we need domain grouping (e.g., subdomains share budget)? (Not in 3.2.)

---

## 23. Next Immediate Actions

1. Add `RateLimitConfig` to `ScraperConfig` + defaulting logic.
2. Create `internal/ratelimit` package with interfaces, clock abstraction, placeholder limiter returning immediate permits (tests first).
3. Write initial token bucket tests (refill & wait logic using fake clock).
4. Iterate through TDD phases outlined in Section 15.

---

This document serves as the authoritative blueprint for Phase 3.2 implementation. Adjustments will be tracked via incremental updates and reflected in progress documentation.
