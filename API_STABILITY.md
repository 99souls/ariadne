# API Stability & Versioning Guide

Status: Draft (Post-P4, Pre-P6)
Date: September 26, 2025
Scope: Engine facade (`packages/engine`) and CLI entrypoint

## Stability Levels

| Component / Field                      | Stability     | Notes |
| -------------------------------------- | ------------- | ----- |
| `engine.New`, `Engine.Start/Stop`      | Stable (v0.1) | Behavior guaranteed; error shapes may expand |
| `engine.Config` core worker fields     | Stable        | Names & types fixed; future fields additive |
| `engine.Config.RateLimit`              | Experimental  | Tunables may change; enabling/disabling stable |
| `engine.Config.Resources`              | Experimental  | May gain eviction / sizing parameters |
| `engine.Config.Resume` + `CheckpointPath` | Experimental | May evolve into strategy enum |
| `Engine.Snapshot()` struct fields      | Evolving      | New fields additive; existing names stable; no removal before v1.0 |
| Internal packages (`internal/*`)       | Internal Only | No compatibility guarantees; do not import directly |

## Backward Compatibility Policy

- Additive changes (new fields / snapshot metrics) are allowed anytime.
- Breaking changes to stable APIs require a major version bump (pre-1.0: minor bump signals breakage).
- Experimental fields can change or be removed with a deprecation notice across one minor version.

## Deprecation Process

1. Mark symbol/comment with `DEPRECATED:` and guidance.
2. Provide facade-based alternative.
3. Maintain for >= one minor version before removal.

## Snapshot Evolution Contract

- JSON tags are part of the public contract.
- Duration fields expressed as Go `time.Duration` string values when marshaled.
- Adding nested sections (e.g., domain rate summaries) is non-breaking.

## Versioning Strategy

Current: Unversioned (pre-release). Git tags will begin at `v0.1.0` upon completion of P7.

Semantic Versioning (SemVer) will guide post-P7 releases:
- MAJOR: incompatible API changes
- MINOR: backward-compatible features
- PATCH: backward-compatible bug fixes

## Future Considerations

- Pluggable output sinks (Prometheus, JSON file, stdout) for snapshots
- Strategy-driven resume (checkpoint, hash-index, external DB)
- Stable event bus for progress / error notifications

---

This document will be updated at the completion of phases P6 and P7 to lock initial stability guarantees.
