package config

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestConfigExportAllowlist guards the curated exported surface of the config package.
// Update this map intentionally when promoting/dropping symbols; avoid accidental drift.
func TestConfigExportAllowlist(t *testing.T) {
	allowed := map[string]struct{}{
		// Core unified config types & constructors
		"UnifiedBusinessConfig": {}, "GlobalSettings": {},
		"NewUnifiedBusinessConfig": {}, "DefaultBusinessConfig": {}, "ComposeBusinessConfig": {}, "FromLegacyConfig": {}, "DefaultGlobalSettings": {},
		// NOTE: Runtime / dynamic configuration system has been internalized (W4-03); removed from public allowlist intentionally.
	}

	_, fname, _, _ := runtime.Caller(0)
	dir := filepath.Dir(fname)
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool { return strings.HasSuffix(fi.Name(), ".go") }, 0)
	if err != nil {
		t.Fatalf("parse dir: %v", err)
	}
	for _, pkg := range pkgs {
		for path, f := range pkg.Files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.TypeSpec:
					if x.Name.IsExported() {
						if _, ok := allowed[x.Name.Name]; !ok {
							t.Fatalf("unexpected exported type: %s", x.Name.Name)
						}
					}
				case *ast.FuncDecl:
					if x.Recv == nil && x.Name.IsExported() {
						if _, ok := allowed[x.Name.Name]; !ok {
							t.Fatalf("unexpected exported function: %s", x.Name.Name)
						}
					}
				case *ast.ValueSpec:
					for _, id := range x.Names {
						if id.IsExported() {
							if _, ok := allowed[id.Name]; !ok {
								t.Fatalf("unexpected exported value: %s", id.Name)
							}
						}
					}
				}
				return true
			})
		}
	}
}
