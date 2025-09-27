This directory houses non-public engine implementation packages migrated from the legacy root `internal/` tree.

Migration status:

- ratelimit (DONE: moved to public engine/ratelimit API surface)
- resources (DONE: engine/resources provides public API; legacy shim removed)
- assets (pending move)
- crawler (pending move)
- processor (pending move)
- pipeline (pending move)

Packages will be moved verbatim first, then pruned/refactored.
