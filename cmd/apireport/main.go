package main

// Command apireport generates a simple API_REPORT.md enumerating exported symbols
// in the public engine packages. It intentionally ignores any packages whose import
// path contains "/internal/". The output groups by package and lists exported types,
// functions, variables, and methods. This is a lightweight inventory – stability tiers
// are preserved if the source file includes lines containing "Stable:", "Experimental:",
// "Deprecated:" or "Internal:" on the preceding doc comments.
//
// Usage:
//   go run ./cmd/apireport > API_REPORT.md
// or via Makefile target: make api-report
//
// The tool prefers the go/packages API for accuracy over naive parsing.

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
)

var (
	modulePrefix = "github.com/99souls/ariadne/engine"
	includePkgs  = []string{
		"github.com/99souls/ariadne/engine",
		"github.com/99souls/ariadne/engine/config",
		"github.com/99souls/ariadne/engine/models",
		"github.com/99souls/ariadne/engine/ratelimit",
		"github.com/99souls/ariadne/engine/resources",
	}
)

func main() {
	outPath := flag.String("out", "API_REPORT.md", "output path (default API_REPORT.md)")
	flag.Parse()
	ctx := context.Background()
	fset := token.NewFileSet()
	cfg := &packages.Config{Fset: fset, Mode: packages.LoadSyntax, Context: ctx}
	pkgs, err := packages.Load(cfg, includePkgs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load packages: %v\n", err)
		os.Exit(1)
	}
	var buf bytes.Buffer
	buf.WriteString("# API Report\n\n")
	buf.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))
	for _, p := range pkgs {
		if strings.Contains(p.PkgPath, "/internal/") { // safety
			continue
		}
		docPkg := &ast.Package{Name: p.Name, Files: map[string]*ast.File{}}
		for i, f := range p.Syntax {
			key := fmt.Sprintf("file%d", i)
			docPkg.Files[key] = f
		}
		d := doc.New(docPkg, p.PkgPath, 0)
		buf.WriteString(fmt.Sprintf("## Package `%s`\n\n", trimModule(p.PkgPath)))
		if d.Doc != "" {
			buf.WriteString(strings.TrimSpace(d.Doc) + "\n\n")
		}
		// Collect exported items
		type entry struct{ name, kind, stability, summary string }
		var entries []entry
		stabilityFrom := func(comment string) string {
			lc := strings.ToLower(comment)
			switch {
			case strings.Contains(lc, "deprecated:"):
				return "Deprecated"
			case strings.Contains(lc, "stable:"):
				return "Stable"
			case strings.Contains(lc, "experimental:"):
				return "Experimental"
			case strings.Contains(lc, "internal:"):
				return "Internal"
			}
			return "" // unspecified
		}
		for _, t := range d.Types {
			if !ast.IsExported(t.Name) {
				continue
			}
			entries = append(entries, entry{t.Name, "type", stabilityFrom(t.Doc), firstLine(t.Doc)})
			for _, m := range t.Methods {
				if ast.IsExported(m.Name) {
					entries = append(entries, entry{fmt.Sprintf("%s.%s", t.Name, m.Name), "method", stabilityFrom(m.Doc), firstLine(m.Doc)})
				}
			}
		}
		for _, f := range d.Funcs {
			if ast.IsExported(f.Name) {
				entries = append(entries, entry{f.Name, "func", stabilityFrom(f.Doc), firstLine(f.Doc)})
			}
		}
		for _, v := range d.Vars {
			for _, n := range v.Names {
				if ast.IsExported(n) {
					entries = append(entries, entry{n, "var", stabilityFrom(v.Doc), firstLine(v.Doc)})
				}
			}
		}
		for _, c := range d.Consts {
			for _, n := range c.Names {
				if ast.IsExported(n) {
					entries = append(entries, entry{n, "const", stabilityFrom(c.Doc), firstLine(c.Doc)})
				}
			}
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
		if len(entries) == 0 {
			buf.WriteString("(no exported symbols)\n\n")
			continue
		}
		buf.WriteString("Name | Kind | Stability | Summary\n")
		buf.WriteString("-----|------|-----------|--------\n")
		for _, e := range entries {
			buf.WriteString(fmt.Sprintf("%s | %s | %s | %s\n", e.name, e.kind, e.stability, escapePipes(e.summary)))
		}
		buf.WriteString("\n")
	}
	if err := os.WriteFile(*outPath, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", err)
		os.Exit(1)
	}
}

func trimModule(path string) string {
	// produce relative path like engine, engine/config etc
	idx := strings.Index(path, modulePrefix)
	if idx == -1 {
		return path
	}
	rel := strings.TrimPrefix(path[idx+len(modulePrefix):], "/")
	if rel == "" {
		return "engine"
	}
	return filepath.ToSlash(rel)
}

func firstLine(doc string) string {
	if doc == "" {
		return ""
	}
	lines := strings.Split(doc, "\n")
	for _, l := range lines {
		s := strings.TrimSpace(l)
		if s != "" {
			return s
		}
	}
	return ""
}

func escapePipes(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
