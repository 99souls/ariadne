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
    data, err := os.ReadFile("main.go")
    if err != nil {
        t.Fatalf("read main.go: %v", err)
    }
    content := string(data)
    if strings.Contains(content, "\"site-scraper/internal/") {
        t.Fatalf("main.go imports internal/*; migrate to engine facade only")
    }
}
