package strategies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestExportAllowlist enforces an allowlist of exported identifiers while the
// package is Experimental. Adjust deliberately (Wave 3) and update list.
func TestExportAllowlist(t *testing.T) {
	// Allowlist of exported identifiers we intentionally expose today.
	allowed := map[string]struct{}{
		"FetchingStrategyType":   {},
		"ProcessingStrategyType": {},
		"OutputStrategyType":     {},
		"ParallelFetching":       {}, "SequentialFetching": {}, "FallbackFetching": {}, "AdaptiveFetching": {},
		"SequentialProcessing": {}, "ParallelProcessing": {}, "ConditionalProcessing": {}, "PipelineProcessing": {},
		"SimpleOutput": {}, "ConditionalRouting": {}, "MultiSinkOutput": {}, "BufferedOutput": {},
		"StrategyComposer":   {},
		"ComposedStrategies": {}, "ComposedFetchingStrategy": {}, "ComposedProcessingStrategy": {}, "ComposedOutputStrategy": {},
		"RetryConfiguration": {}, "AdaptiveConfiguration": {}, "ProcessingCondition": {}, "MultiSinkConfiguration": {},
		"StrategyMetadata": {},
		"StrategyExecutor": {}, "ExecutionPlan": {}, "FetchingExecutionPlan": {}, "ProcessingExecutionPlan": {}, "OutputExecutionPlan": {},
		"StrategyPerformanceMonitor": {}, "StrategyMetrics": {}, "PerformanceMetrics": {}, "PerformanceRecommendations": {},
		"StrategyOptimizer": {}, "OptimizationRule": {},
		"AdaptiveStrategyManager": {}, "StrategyAdjustment": {}, "PerformanceFeedback": {},
		"NewStrategyComposer": {}, "NewStrategyExecutor": {}, "NewStrategyPerformanceMonitor": {}, "NewStrategyOptimizer": {}, "NewAdaptiveStrategyManager": {},
	}

	// Parse current file (runtime.Caller locates this test file directory)
	_, fname, _, _ := runtime.Caller(0)
	dir := filepath.Dir(fname)
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		t.Fatalf("parse dir: %v", err)
	}
	for _, pkg := range pkgs {
		for fnameAst, f := range pkg.Files {
			isTestFile := strings.HasSuffix(fnameAst, "_test.go")
			ast.Inspect(f, func(n ast.Node) bool {
				// Track exported top-level declarations
				switch x := n.(type) {
				case *ast.TypeSpec:
					if x.Name.IsExported() {
						name := x.Name.Name
						if _, ok := allowed[name]; !ok {
							// Skip test files added types
							if strings.HasSuffix(f.Name.Name, "_test") {
								return false
							}
							t.Fatalf("unexpected exported type: %s (update allowlist or internalize)", name)
						}
					}
				case *ast.ValueSpec:
					for _, ident := range x.Names {
						if ident.IsExported() {
							if _, ok := allowed[ident.Name]; !ok {
								if strings.HasSuffix(f.Name.Name, "_test") {
									continue
								}
								t.Fatalf("unexpected exported value: %s (update allowlist or internalize)", ident.Name)
							}
						}
					}
				case *ast.FuncDecl:
					if isTestFile {
						return true
					}
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
