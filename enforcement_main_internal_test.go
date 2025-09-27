package main_test

import (
	"os"
	"strings"
	"testing"
)

// TestNoInternalImports enforces that the root CLI does not directly import
// any internal implementation packages. This guards the architectural boundary
// established in P5 and formalized in P6.
func TestNoInternalImports(t *testing.T) {
    path := "cli/cmd/ariadne/main.go"
    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("read %s: %v", path, err)
    }
    content := string(data)
    if strings.Contains(content, "\t\"github.com/99souls/ariadne/engine/internal/") || strings.Contains(content, "\t\"ariadne/internal/") {
        t.Fatalf("%s imports internal/* packages; CLI must only use public engine API", path)
    }
}
