package engine

// Wave 3: Engine export allowlist guard.
// This test enforces a curated set of exported identifiers in the root engine
// package while the public surface is being aggressively pruned and annotated.
// If you intentionally add or remove an export, update the allowlist below
// together with CHANGELOG, API report regeneration, and pruning candidates doc.

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

func TestEngineExportAllowlist(t *testing.T) {
	allowed := map[string]struct{}{
		// Core types
		"Engine": {}, "Config": {}, "ResourcesConfig": {}, "Snapshot": {}, "ResourceSnapshot": {}, "ResumeSnapshot": {},
		// Construction & strategies placeholder
		"New": {}, "NewWithStrategies": {}, "EngineStrategies": {},
		// Consolidated strategy interfaces
		"Fetcher": {}, "Processor": {}, "OutputSink": {}, "AssetStrategy": {},
		// Asset subsystem extension points / metrics snapshots
		"AssetPolicy": {}, "AssetRef": {}, "AssetMode": {}, "AssetAction": {}, "MaterializedAsset": {},
		"AssetEvent": {}, "AssetEventPublisher": {}, "AssetMetrics": {}, "AssetMetricsSnapshot": {}, "DefaultAssetStrategy": {},
		// Public enums / consts (asset modes)
		"AssetModeDownload": {}, "AssetModeSkip": {}, "AssetModeInline": {}, "AssetModeRewrite": {},
		// Config helpers
		"Defaults": {},
		// Constructor for default asset strategy (still public while subsystem experimental)
		"NewDefaultAssetStrategy": {},
	}

	// Parse current package directory (this test's directory)
	_, fname, _, _ := runtime.Caller(0)
	dir := filepath.Dir(fname)
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool { return strings.HasSuffix(fi.Name(), ".go") }, 0)
	if err != nil {
		t.Fatalf("parse dir: %v", err)
	}
	for _, pkg := range pkgs {
		for path, f := range pkg.Files {
			if strings.HasSuffix(path, "_test.go") { // skip test files' own exports
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.TypeSpec:
					if x.Name.IsExported() {
						if _, ok := allowed[x.Name.Name]; !ok {
							t.Fatalf("unexpected exported type: %s (update allowlist or internalize)", x.Name.Name)
						}
					}
				case *ast.ValueSpec:
					for _, id := range x.Names {
						if id.IsExported() {
							if _, ok := allowed[id.Name]; !ok {
								t.Fatalf("unexpected exported value: %s (update allowlist or internalize)", id.Name)
							}
						}
					}
				case *ast.FuncDecl:
					if x.Recv == nil && x.Name.IsExported() {
						if _, ok := allowed[x.Name.Name]; !ok {
							t.Fatalf("unexpected exported function: %s (update allowlist or internalize)", x.Name.Name)
						}
					}
				}
				return true
			})
		}
	}
}
