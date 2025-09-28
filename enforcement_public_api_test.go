package main_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoInternalImportsOutsideEngine ensures no packages outside engine reference engine/internal/*.
func TestNoInternalImportsOutsideEngine(t *testing.T) {
	// Walk all dirs except engine/internal
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.Contains(path, "engine/internal") {
				return filepath.SkipDir
			}
			// skip VCS & vendor
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".git") || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(src), "github.com/99souls/ariadne/engine/internal/") {
			// Allow imports within engine/ itself (facade wiring). Everything else forbidden.
			if !strings.HasPrefix(path, "engine/") {
				t.Fatalf("file %s imports engine/internal/* which is forbidden", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

// TestNoNewTopLevelDirs ensures no new top-level code directories beyond whitelist (strict version; transitional dirs allowed in existing guard test).
func TestNoNewTopLevelDirs(t *testing.T) {
	allowed := map[string]struct{}{"engine": {}, "cli": {}, "md": {}, "cmd": {}, ".github": {}, ".git": {}}
	transitional := map[string]struct{}{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read root: %v", err)
	}
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
		t.Fatalf("unexpected new top-level directory: %s", name)
	}
}
