package telemetry

// Wave 4 (W4-05): Telemetry export allowlist guard.
// This test enforces a curated set of exported identifiers across the
// `engine/telemetry/*` public packages to prevent accidental surface growth.
// If you intentionally add or remove an export, update the allowlist here
// together with CHANGELOG, API report regeneration, and pruning candidates doc.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTelemetryExportAllowlist(t *testing.T) {
	// Aggregated allowlist: map package import path suffix -> allowed exported identifiers.
	allow := map[string]map[string]struct{}{
		"events": {
			// Core event bus contracts currently public (subject to future facade wrap)
			"Event": {}, "Subscription": {}, "Bus": {}, "BusStats": {},
			// Constructors/constants
			"NewBus":         {},
			"CategoryAssets": {}, "CategoryPipeline": {}, "CategoryRateLimit": {}, "CategoryResources": {}, "CategoryConfig": {}, "CategoryError": {}, "CategoryHealth": {},
		},
		"metrics": {
			// Provider interfaces & option structs (may be narrowed in future)
			"Provider": {}, "Counter": {}, "Gauge": {}, "Histogram": {}, "Timer": {},
			"CommonOpts": {}, "CounterOpts": {}, "GaugeOpts": {}, "HistogramOpts": {}, "PrometheusProvider": {}, "OTelProvider": {},
			// Legacy adapter types (subject to future internalization)
			"BusinessCollectorAdapter": {}, "NewBusinessCollectorAdapter": {},
			// Public constructors for built-in providers
			"NewPrometheusProvider": {}, "PrometheusProviderOptions": {}, "NewOTelProvider": {}, "OTelProviderOptions": {}, "NewNoopProvider": {},
		},
		"tracing": {
			// Minimal tracing interfaces
			"Tracer": {}, "Span": {}, "SpanContext": {},
			// Constructors
			"NewTracer": {}, "NewAdaptiveTracer": {},
			// Context helpers
			"SpanFromContext": {}, "ExtractIDs": {},
		},
		"policy": {
			// Telemetry policy surface (may be slimmed later behind engine facade)
			"TelemetryPolicy": {}, "HealthPolicy": {}, "TracingPolicy": {}, "EventBusPolicy": {},
			"Default": {},
		},
		"health": {
			// Health evaluator snapshot types (public for adapter consumption)
			"Snapshot": {}, "ProbeResult": {}, "Status": {}, "Probe": {}, "ProbeFunc": {}, "Evaluator": {},
			// Factory helpers
			"NewEvaluator": {},
			// Helper constructors
			"Healthy": {}, "Degraded": {}, "Unhealthy": {}, "Unknown": {},
			// Status constants
			"StatusUnknown": {}, "StatusHealthy": {}, "StatusDegraded": {}, "StatusUnhealthy": {},
			// Status helpers (consts or funcs) intentionally excluded: they are unexported or validated separately
		},
		"logging": {
			// Logging facade (if kept minimal)
			"Logger": {}, "NewLogger": {}, "New": {},
		},
	}

	// Determine telemetry root directory (this file's directory parent).
	_, thisFile, _, _ := runtime.Caller(0)
	telemetryDir := filepath.Dir(thisFile)

	// Walk immediate subdirectories (packages) and inspect exports.
	entries, err := filepath.Glob(filepath.Join(telemetryDir, "*"))
	if err != nil {
		t.Fatalf("glob telemetry subdirs: %v", err)
	}
	for _, pkgPath := range entries {
		info, err := os.Stat(pkgPath)
		if err != nil || !info.IsDir() {
			continue
		}
		sub := filepath.Base(pkgPath)
		allowed, ok := allow[sub]
		if !ok {
			// If a new telemetry subpackage appears, force explicit decision.
			t.Fatalf("unexpected telemetry subpackage: %s (add to allowlist or internalize)", sub)
		}
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, pkgPath, func(fi os.FileInfo) bool { return strings.HasSuffix(fi.Name(), ".go") }, 0)
		if err != nil {
			t.Fatalf("parse dir %s: %v", pkgPath, err)
		}
		for _, p := range pkgs {
			for filePath, f := range p.Files {
				if strings.HasSuffix(filePath, "_test.go") { // ignore test files
					continue
				}
				ast.Inspect(f, func(n ast.Node) bool {
					switch x := n.(type) {
					case *ast.TypeSpec:
						if x.Name.IsExported() {
							if _, ok := allowed[x.Name.Name]; !ok {
								t.Fatalf("unexpected exported type %s in telemetry/%s (update allowlist or internalize)", x.Name.Name, sub)
							}
						}
					case *ast.ValueSpec:
						for _, id := range x.Names {
							if id.IsExported() {
								if _, ok := allowed[id.Name]; !ok {
									t.Fatalf("unexpected exported value %s in telemetry/%s (update allowlist or internalize)", id.Name, sub)
								}
							}
						}
					case *ast.FuncDecl:
						if x.Recv == nil && x.Name.IsExported() { // top-level funcs only
							if _, ok := allowed[x.Name.Name]; !ok {
								t.Fatalf("unexpected exported function %s in telemetry/%s (update allowlist or internalize)", x.Name.Name, sub)
							}
						}
					}
					return true
				})
			}
		}
	}
}

// (No custom FS adapter required; using os.Stat directly above.)
