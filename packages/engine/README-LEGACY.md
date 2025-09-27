# Legacy Engine Tree (Frozen)

Status: FROZEN after Phase 5F big-bang merge.

This directory is the pre-extraction copy of the engine that existed under `packages/engine` prior to migration to the standalone module at `engine/` (module path `github.com/99souls/ariadne/engine`).

Policy:

- Do NOT modify implementation code here.
- Only permitted changes: removal, addition of forwarders, or mechanical rewrites that redirect to the canonical engine module.
- New development MUST occur in `engine/`.
- Tests here will be removed progressively (Wave 2B) in favor of the `engine/` module test suite.

Migration Waves Reference:

| Wave | Action                                                      |
| ---- | ----------------------------------------------------------- |
| 2A   | Normalize imports inside `engine/` (in progress)            |
| 2B   | Replace this tree with thin forwarders / remove duplicates  |
| 3    | API pruning & internalization (performed only in `engine/`) |
| 5    | Forwarder removal (this directory likely deleted)           |

Enforcement:

A Make target `legacy-imports` plus a future CI check counts occurrences of the old import root `ariadne/packages/engine`. This count must never increase and will trend toward zero until removal.

If you believe you need to touch this directory for a feature/fix, open an architecture discussion first.

— Migration Guardians
