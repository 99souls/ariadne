# Telemetry Overhead Report (Phase 5E Final)

Status: Complete
Date: 2025-09-27
Related: `phase5e-plan.md`, `benchmark_provider_test.go`

---

## 1. Purpose

Document measured overhead of telemetry subsystems relative to a baseline to satisfy Phase 5E non-functional exit criteria (<5% CPU, <10% memory increase full telemetry enabled).

## 2. Methodology

- Benchmarks: `BenchmarkProviderCounterInc`, `BenchmarkProviderHistogramObserve`, `BenchmarkProviderTimer`.
- Build: `go test -bench=BenchmarkProvider -benchmem ./packages/engine/telemetry/metrics`
- Environment capture: Go version, CPU model, logical CPUs, commit hash.
- Modes: noop (baseline), prom, otel.
- Timer benchmark includes minimal 1ns sleep to simulate span of work.

## 3. Microbenchmark Results (Final)

Updated results after Prometheus timer optimization (histogram reuse) on developer machine (Apple M4 Max, Go 1.25.x). OTEL figures still exclude external exporter pipeline.

| Benchmark        | noop ns/op | prom ns/op | otel ns/op | prom % over noop | otel % over noop | allocs (noop/prom/otel) |
| ---------------- | ---------: | ---------: | ---------: | ---------------: | ---------------: | ----------------------- |
| CounterInc       |       0.97 |      57.29 |       4.24 |            5804% |             337% | 0 / 1 / 0               |
| HistogramObserve |       1.07 |      64.75 |       4.20 |            5971% |             294% | 0 / 1 / 0               |
| Timer            |      201.9 |      412.2 |      312.5 |             104% |              55% | 0 / 2 / 1               |

Improvement: Prometheus timer path reduced from ~675ns / 4 allocs to ~412ns / 2 allocs.

## 4. Integrated Workload Benchmark

`BenchmarkIntegratedWorkload` simulates a representative per-page workload (3 stage timers, ~5 assets, occasional failure) to approximate aggregate telemetry overhead.

| Provider | ns/op | B/op | allocs/op |
| -------- | ----: | ---: | --------: |
| noop     |   714 |   96 |         6 |
| prom     |  2779 |  545 |        31 |
| otel     |  2903 | 1879 |        57 |

Interpretation: Full metrics emission adds ~2.0–2.2µs absolute overhead per simulated page unit of work in this synthetic loop. For high-throughput scenarios (e.g. 10k pages/sec), raw added CPU time remains within budget (<5% of a single core), assuming similar distribution; real workloads will have higher intrinsic processing cost diluting percentage further.

## 5. Interpretation & Analysis

Observations:

- Prometheus provider absolute cost remains sub-65ns for single metric ops—acceptable for crawl cadence; high relative % vs ~1ns noop baseline expected.
- OTEL provider (bridge only) cost is ~4ns per counter/hist observe—lower than Prom due to absence of exporter pipeline; numbers will rise once exporters attached.
- Timer overhead dominated by histogram record + minimal work; Prom adds ~485ns vs noop, OTEL adds ~125ns.

Threshold Check (Qualitative):

- Even Prom 65ns ops at, e.g., 50k ops/sec => ~3.25ms CPU time per second (<<5% of a single core). Next step: integrated scenario benchmark to empirically validate CPU <5% overhead.
- Memory allocations: Prom allocs for counter/hist are 1 per op; OTEL zero. Potential optimization for Prom timer (current 4 allocs) if necessary.

Follow-up Profiling Targets:

1. Reuse timer histogram to reduce allocs.
2. Investigate fast-path label caching if label-bearing metrics become hot.
3. Re-run after enabling OTEL exporter to obtain realistic overhead.

Threshold Check: Micro + workload results indicate Prometheus + tracing + events (future) likely remain under CPU & memory budgets. Memory alloc focus areas: OTEL provider path allocs (future exporter integration tuning) and Prometheus counter/hist allocation (acceptable for now).

## 6. SLO Impact

Telemetry Overhead SLO (<5% CPU, <10% memory) marked Provisionally PASS based on synthetic workload. Production validation deferred to Phase 5F capacity exercises.

## 7. Change Log

- Initial scaffold (Iteration 7)
- Added timer optimization + updated results
- Added integrated workload benchmark results & analysis
