# Outstanding Engine Work (Consolidated Backlog)

Status Date: 2025-09-29
Scope: Engine package (`engine/`) and directly related internal packages impacting crawl correctness, determinism, performance, and Phase 6 exit criteria. Phase 7 initiatives are explicitly paused until this list is either completed or transformed into GitHub issues tagged appropriately.

---
## 1. Determinism & Stability (Phase 6 Close-Out)

| Item | Description | Priority | Action | Exit Signal |
|------|-------------|----------|--------|-------------|
| Flake Baseline Run | Establish 20-iteration flake detector baseline post race + search index changes | P0 | Run `ITER=20 make flake-live` locally & in a temporary CI branch | Document 0/20 failures in issue comment | 
| Snapshot Acceptance Guide | Expand docs on reviewing HTML snapshot diffs (criteria, red flags, acceptance checklist) | P1 | Add `md/snapshot-acceptance.md` then link from README | Doc merged & referenced | 
| Sitemap Ingestion Decision | Decide: implement sitemap seeding vs defer | P1 | Create issue summarizing approach; implement or mark deferred | Issue link recorded here | 
| CI Reuse Validation Job | Confirm test site reuse across LiveSite tests in CI (single process) | P1 | Add dedicated job that runs two sequential focused tests & compares instance ID | Job green & fails if ID differs |
| Depth/Latency Metrics Instrumentation (Optional) | Lightweight timing histogram + depth counter for diagnostics (internal only) | P2 | Instrument internal pipeline / crawler; expose internal debug snapshot | Metrics visible in test assertion (optional) |

---
## 2. Correctness & Feature Parity

| Item | Description | Priority | Notes |
|------|-------------|----------|-------|
| Sitemap Seeding (if chosen) | Allow initial URL seeds to be augmented by sitemap.xml fetch | Depends (P1 if chosen) | Must preserve domain allowlist & robots honor |
| Enhanced Metadata Parsing | OG tags + canonical link capture into Page metadata | P2 | Improves future downstream output fidelity |
| Admonition / Footnote Structural Validation | Add integration assertions verifying role attributes + backlink anchors | P2 | Uses existing `/static/admonitions.html` fixture |
| Code Fence Language Preservation | Ensure extraction path preserves `language-` classes | P2 | Add targeted unit/integration test |
| Tag Page / Backlink Graph Sanity | Derive basic adjacency from discovered links + ensure consistency | P3 | Potential future graph export |

---
## 3. Performance & Resource Efficiency

| Item | Description | Priority | Target |
|------|-------------|----------|--------|
| Throughput Profiling Harness | Micro benchmark or integration measuring pages/sec under local loopback | P2 | Baseline before later pipeline changes |
| Memory Sampling Snapshot | Capture periodic RSS (approx) during crawl for >N pages (config flag) | P3 | Document in `metrics-reference.md` |
| Adaptive Delay Refinement | Revisit default `RequestDelay` vs rate limiter interplay | P3 | Derive from observed p95 latency windows |

---
## 4. Observability Foundations (Pre-Phase 7)

| Item | Description | Priority | Deliverable |
|------|-------------|----------|------------|
| Health Snapshot Adapter Design Note | Mini doc clarifying boundary (engine vs adapter package) | P1 | `md/outstanding/health-adapter-boundary.md` (or integrate into existing telemetry docs) |
| Minimal Metrics Adapter Stub | Compile-time guarded adapter that can export counters (no-op default) | P2 | Non-breaking; disabled by default |

---
## 5. API Surface Governance

| Item | Description | Priority | Action |
|------|-------------|----------|--------|
| Exported Symbol Diff Guard | Script comparing `api-report` diff vs allowlist label | P2 | Extend `tools/apireport` or add shell harness |
| Stability Annotation Coverage | Ensure all exported types/functions have stability comment tier | P2 | Gate in Phase 7, prep now |

---
## 6. Test Coverage Improvements (Engine Focused)

Target: Achieve ≥75% critical weighted coverage by Phase 6 exit. Current gaps:

| Area | Gap | Remediation | Priority |
|------|-----|------------|----------|
| internal/assets | Low negative-path & error tests | Add tests for missing image rewrite & fallback | P1 |
| internal/pipeline | Cancellation & timeout branch untested | Inject context cancellation mid-crawl | P1 |
| internal/processor | Malformed HTML branches skipped | Craft broken HTML fixture set | P2 |
| telemetry/health | State transition edges | Unit test degraded→healthy→degraded permutations | P2 |
| logging | Error context propagation | Force errors in collector callbacks | P3 |

---
## 7. CI Enhancements (Immediate)

| Item | Description | Priority | Implementation Notes |
|------|-------------|----------|----------------------|
| LiveSite Integration Job | Add job running `make integ-live` with Bun install & caching | P0 | Reuse Go version matrix=1.25.1; cache Bun ~/.bun |
| Flake Detector Job (Advisory) | Run `ITER=15 make flake-live` (non-blocking initially) | P1 | Mark as `continue-on-error: true` initially |
| Coverage Threshold Soft Gate | Use new `make coverage-check THRESHOLD=70` | P1 | Warning only (no fail) until threshold stable |

---
## 8. Deferred / Decision Log Candidates

| Candidate | Rationale to Defer | Revisit When |
|-----------|--------------------|--------------|
| Sitemap Seeding (if deferred) | Adds network variance & code complexity | After Phase 7 stabilization |
| Depth/Latency Metrics | Not required for determinism gate | When observability sprint begins |
| Tag Backlink Graph | Nice-to-have for future knowledge graph export | Post v0.2.0 |

---
## 9. Execution Ordering Proposal

1. CI LiveSite + coverage-check (soft) + flake baseline (advisory)
2. Snapshot acceptance doc
3. Decide sitemap (implement or defer)
4. Asset negative path tests (raise coverage)
5. Pipeline cancellation test
6. Metadata & structural extraction assertions (admonitions, code fences)
7. Health adapter design note
8. Optional metrics/depth instrumentation

---
## 10. Tracking & Issue Hygiene

Each item graduating from this doc MUST have a GitHub issue with:
- Label: `engine`, plus `stability`, `observability`, or `performance` as appropriate
- Link back to this document section (anchor recommended)
- Clear acceptance criteria & test plan

On completion: remove or mark item with ✅ here (keep historical row until Phase 7 start; then archive snapshot of this file).

---
## 11. Out of Scope (Phase 7+ Only)

- CLI UX polish (progress bars, interactive prompts)
- A11y lint integration & alt text coverage report
- PDF/HTML multi-format richer output adapter
- Plugin/adapter ecosystem elaboration (browser fetcher, etc.)

These remain paused to protect focus on determinism, correctness, and CI signal quality.

---
Generated collaboratively. Update date stamp on substantial edits.
