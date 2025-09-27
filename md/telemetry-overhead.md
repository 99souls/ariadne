# Telemetry Overhead Report (Iteration 7)

Status: In Progress
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

## 3. Benchmark Results (Initial Run)

Initial run on developer machine (see environment captured in benchmark log output). Treat OTEL figures as lower bound (no exporter pipeline hooked yet).

| Benchmark        | noop ns/op | prom ns/op | otel ns/op | prom % over noop | otel % over noop | allocs (noop/prom/otel) |
| ---------------- | ---------: | ---------: | ---------: | ---------------: | ---------------: | ----------------------- |
| CounterInc       |       0.97 |      57.29 |       4.24 |            5804% |             337% | 0 / 1 / 0               |
| HistogramObserve |       1.07 |      64.75 |       4.20 |            5971% |             294% | 0 / 1 / 0               |
| Timer            |      190.1 |      674.9 |      314.8 |             255% |              66% | 0 / 4 / 1               |

## 4. Interpretation & Preliminary Analysis

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

## 5. Next Steps

1. Run integrated end-to-end crawl benchmark (future script) for holistic CPU delta.
2. Update `slo-baselines.md` with overhead actuals (Telemetry Overhead line: PASS / within target preliminarily).
3. Append summarized overhead table to `phase5e-plan.md` Iteration 7 completion snapshot.

## 6. Change Log

- Iteration 7: Initial scaffold with methodology and table.
