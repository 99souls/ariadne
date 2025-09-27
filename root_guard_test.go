package main_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoRootMain ensures no executable Go files live at repo root (post Phase 5F purge).
func TestNoRootMain(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil { t.Fatalf("read root: %v", err) }
	for _, e := range entries {
		if e.IsDir() { continue }
		name := e.Name()
		if !strings.HasSuffix(name, ".go") { continue }
		if strings.HasSuffix(name, "_test.go") { continue }
		// Exempt build scripts or tooling placeholders if any appear later (none now)
		p := filepath.Join(".", name)
		content, err := os.ReadFile(p)
		if err != nil { t.Fatalf("read %s: %v", p, err) }
		if strings.Contains(string(content), "package main") {
			 t.Fatalf("unexpected executable Go file at root: %s", name)
		}
	}
}
