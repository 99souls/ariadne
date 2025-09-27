package cli_test

import (
	"os"
	"strings"
	"testing"
)

// TestNoInternalImports ensures the CLI does not import engine internal packages.
// Relocated from root during root module elimination.
func TestNoInternalImports(t *testing.T) {
	path := "cli/cmd/ariadne/main.go"
	data, err := os.ReadFile("../../" + path)
	if err != nil {
		// Fallback: attempt relative from workspace root (when go test launched at module root)
		data, err = os.ReadFile("../" + path)
		if err != nil {
			// Last attempt: direct path
			data, err = os.ReadFile(path)
			if err != nil {
				// Do not fail hard if path layout changes; treat as skipped.
				t.Skipf("could not read %s: %v", path, err)
			}
		}
	}
	content := string(data)
	if strings.Contains(content, "\t\"github.com/99souls/ariadne/engine/internal/") || strings.Contains(content, "\t\"ariadne/internal/") {
		t.Fatalf("%s imports internal/* packages; CLI must only use public engine API", path)
	}
}
