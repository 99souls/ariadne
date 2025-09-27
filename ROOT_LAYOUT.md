# Root Layout Invariants

The repository root is intentionally non-modular (no `go.mod`). All buildable Go code lives exclusively inside explicit modules:

- `engine/` – library / embedding API
- `cli/` – user-facing binary module
- `tools/apireport/` – maintenance tool (API surface report)

## Prohibited at Root

- Any `go.mod` file
- Executable or library `.go` files (besides historical archived sources under `archive/` with `//go:build ignore`)
- New code directories other than the whitelisted module directories and documentation (`md/`)

## Enforcement

- Directory whitelist test (enforcement test relocated into a module) ensures root contains only approved entries.
- CI workflow runs `go work sync` and per-module tests/builds; failure if a root `go.mod` is introduced.

## Rationale

Removing the root module eliminates dependency duplication and accidental API leakage, reinforcing the Atomic Root Layout goal: top-level clarity and strict module boundaries.
