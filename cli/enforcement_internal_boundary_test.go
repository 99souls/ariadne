package cli_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoInternalImports ensures no CLI package imports engine internal packages.
func TestNoInternalImports(t *testing.T) {
	// Walk from module root (this test runs inside cli/ module)
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip vendor if ever present
			if d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip this guard test file itself; it necessarily references internal pattern strings.
		if strings.HasSuffix(path, "enforcement_internal_boundary_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(b)
		if strings.Contains(content, "github.com/99souls/ariadne/engine/internal/") {
			t.Fatalf("file %s imports engine/internal – CLI must depend only on public engine API", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk cli module: %v", err)
	}
}
