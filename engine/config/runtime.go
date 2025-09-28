package config

// NOTE: Runtime subsystem internalized.
// This file intentionally left as a lightweight placeholder after removal of the
// experimental runtime & A/B testing API that previously lived here.
// The authoritative implementation now resides under internal/runtime and is
// not part of the public Engine configuration surface.
//
// Do NOT add exported types or functions here without updating:
//   - config_allowlist_guard_test.go
//   - API_PRUNING_CANDIDATES.md (regression note)
//   - API_REPORT.md (via `make api-report`)
//
// Having this stub prevents accidental recreation of the original large file
// during merges or manual experimentation.
