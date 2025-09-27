package main_test

import (
	"os"
	"testing"
)

// TestRootDirectoryWhitelist enforces the atomic root layout objective: only
// allowed top-level code directories are `engine` and `cli`. All other legacy
// code directories must be removed or migrated.
func TestRootDirectoryWhitelist(t *testing.T) {
	allowed := map[string]struct{}{
		"engine": {}, "cli": {}, "md": {},
		// Non-code / metadata directories permitted:
		".git": {}, ".github": {},
	}

	// Transitional legacy directories still pending full migration. List has shrunk
	// after removal of packages/adapters (telemetryhttp) and legacy pipeline stubs.
	transitional := map[string]struct{}{
		"internal": {}, // pipeline tests & historical refs pending final prune
		"cmd":      {}, // old experimental commands scheduled for deletion
		"packages": {}, // legacy directory still present (adapter files pending physical removal)
		"test":     {}, // root test harness utilities (may relocate under engine/)
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read root: %v", err)
	}
	unexpected := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, ok := allowed[name]; ok {
			continue
		}
		if _, ok := transitional[name]; ok {
			continue
		}
		unexpected = append(unexpected, name)
	}
	if len(unexpected) > 0 {
		t.Fatalf("unexpected root directories present (violates atomic root policy): %v", unexpected)
	}
}
