# Snapshot Acceptance & Review Guide

Status Date: 2025-09-29
Scope: Live test site HTML snapshot goldens in `engine/internal/testutil/testsite/testdata/snapshots/`.

---

## Purpose
Snapshots provide an early warning when the rendered HTML surface (used for crawler extraction correctness) drifts. They are NOT a substitute for targeted assertions; they complement them by surfacing unexpected structural or semantic changes.

## When a Snapshot Test Fails
You will see a unified diff with the first differing hunk. Treat the failure as a question: *"Did we intentionally change site content or extraction normalization?"*

### Triage Checklist
1. Intentional content change? (e.g., editing test-site route content)
2. Normalization regression? (missing trim / attribute strip)
3. Introduced nondeterminism? (timestamps, IDs, random ordering)
4. Encoding / whitespace artifact? (double spaces, stray CRLF)
5. External asset or network dependent content snuck in?

If ANY answer is unclear, **do not accept** yet—investigate.

## Accepting an Intentional Change
1. Ensure diff only includes intended structural/textual edits.
2. Run:
   ```bash
   UPDATE_SNAPSHOTS=1 go test ./engine/internal/testutil/testsite -run TestGenerateSnapshots -count=1
   ```
3. Re-run the full live integration suite:
   ```bash
   make integ-live
   ```
4. Commit updated goldens with concise message: `testsite: update snapshot for <reason>`.

## Red Flags (Investigate Before Accepting)
| Symptom | Possible Cause |
| ------- | -------------- |
| Reordered large blocks | Unstable DOM traversal or race in render pipeline |
| Added timestamps / numbers changing per run | Nondeterministic content source |
| Random attribute suffixes (e.g., `id="el-93812"`) | Client runtime generating ephemeral IDs |
| Large whitespace collapse/expansion | Normalization filter change or minifier behavior shift |
| Missing critical structural tags (h1, nav) | Regression in test site content or build step |

## Normalization Rules (Current)
- Strip cosmetic query params (`theme`, `utm_*`)
- Collapse multiple spaces
- Trim trailing whitespace lines
- Remove volatile attributes (data-* with variable tokens)

(If you add or change rules, update this doc and tests.)

## Adding a New Snapshot
1. Extend the test-site with deterministic content.
2. Add fetch & normalization logic (if needed) to snapshot generation test.
3. Run with `UPDATE_SNAPSHOTS=1` to create new golden.
4. Justify addition in PR (what invariant is the snapshot guarding?).

## Deleting a Snapshot
Avoid unless content route removed. Provide rationale in commit. Never delete to "silence" flakes—flaky snapshots indicate missing normalization.

## Flake Handling
If snapshot diffs appear intermittently:
1. Capture two failing diffs—compare variance.
2. Introduce explicit normalization (strip unstable attributes / reorder deterministic sort).
3. Only as last resort: reduce scope of snapshot to stable subtree.

## Review PR Checklist (Reviewer)
- [ ] Diff limited to intended semantic change.
- [ ] No obvious nondeterministic tokens remain.
- [ ] Normalization rules still applied (spot check known stripped attrs).
- [ ] LiveSite tests still pass.
- [ ] Doc updated if normalization rules changed.

## Future Enhancements (Backlog)
- Structured snapshot (DOM AST) to reduce noise from formatting.
- Per-section snapshotting (partition large pages for focused diffs).
- Tool to highlight only semantic tag/attribute changes ignoring pure text whitespace.

---
Generated collaboratively; refine as practices evolve.
