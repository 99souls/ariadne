package models

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

// TestModelsExportAllowlist guards the curated exported surface of models.
// Adjust deliberately (Wave 3) and update CHANGELOG + API report if changed.
func TestModelsExportAllowlist(t *testing.T) {
    allowed := map[string]struct{}{
        "Page": {}, "PageMeta": {}, "OpenGraphMeta": {},
        "CrawlResult": {}, "CrawlStats": {}, "RateLimitConfig": {},
        "ScraperConfig": {}, "DefaultConfig": {},
        "ErrMissingStartURL": {}, "ErrMissingAllowedDomains": {}, "ErrInvalidMaxDepth": {},
        "ErrURLNotAllowed": {}, "ErrMaxDepthExceeded": {}, "ErrMaxPagesExceeded": {},
        "ErrContentNotFound": {}, "ErrHTTPError": {}, "ErrHTMLParsingFailed": {},
        "ErrMarkdownConversion": {}, "ErrAssetDownloadFailed": {}, "ErrOutputDirCreation": {},
        "ErrFileWriteFailed": {}, "ErrTemplateExecution": {},
        "CrawlError": {}, "NewCrawlError": {},
    }
    _, fname, _, _ := runtime.Caller(0)
    dir := filepath.Dir(fname)
    fset := token.NewFileSet()
    pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool { return strings.HasSuffix(fi.Name(), ".go") }, 0)
    if err != nil { t.Fatalf("parse dir: %v", err) }
    for _, pkg := range pkgs {
        for path, f := range pkg.Files {
            if strings.HasSuffix(path, "_test.go") { continue }
            ast.Inspect(f, func(n ast.Node) bool {
                switch x := n.(type) {
                case *ast.TypeSpec:
                    if x.Name.IsExported() { if _, ok := allowed[x.Name.Name]; !ok { t.Fatalf("unexpected exported type: %s", x.Name.Name) } }
                case *ast.ValueSpec:
                    for _, id := range x.Names { if id.IsExported() { if _, ok := allowed[id.Name]; !ok { t.Fatalf("unexpected exported value: %s", id.Name) } } }
                case *ast.FuncDecl:
                    if x.Recv == nil && x.Name.IsExported() { if _, ok := allowed[x.Name.Name]; !ok { t.Fatalf("unexpected exported function: %s", x.Name.Name) } }
                }
                return true
            })
        }
    }
}