This directory houses non-public engine implementation packages migrated from the legacy root `internal/` tree.

Migration status:

- ratelimit (DONE: internalized; implementation moved here from former public package)
- resources (DONE: engine/resources provides public API; legacy shim removed)
- assets (INTERNALIZED: code relocated to engine/internal/assets; tests to be restored)
- crawler (INTERNALIZED: code relocated to engine/internal/crawler; tests to be restored)
- processor (INTERNALIZED: code relocated to engine/internal/processor; asset alias layer trimmed; tests to be restored)
- pipeline (INTERNALIZED: code & representative tests relocated; full original test suite trimmed for now)

Next steps:

1. Move internal/pipeline into engine/internal/pipeline preserving tests.
2. Relocate adapters (packages/adapters/telemetryhttp) under engine/adapters/.
3. Reconstitute and adapt tests for assets, processor, crawler under new paths.
4. Remove deprecated cmd/test-phase1 entirely (currently build-tag ignored).
5. Tighten root directory whitelist to reflect removals.
6. Begin API pruning / stability annotations for public engine surface.

Packages are moved verbatim first (minimal edits) before pruning/refactoring for the stable engine API phase.
