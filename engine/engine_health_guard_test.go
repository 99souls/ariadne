package engine

import (
	"reflect"
	"testing"
)

// TestNoHealthEvaluatorForTest ensures the legacy test-only mutator is not reintroduced.
func TestNoHealthEvaluatorForTest(t *testing.T) {
	mset := map[string]struct{}{}
	et := reflect.TypeOf(&Engine{})
	for i := 0; i < et.NumMethod(); i++ {
		mset[et.Method(i).Name] = struct{}{}
	}
	if _, ok := mset["HealthEvaluatorForTest"]; ok {
		// Intentional assertion: this method was internalized during Wave 3 API pruning.
		// If this fires, remove the method again and update tests using a stub implementing HealthSnapshot instead.
		// See phase5f-plan Wave 3 tasks.
		// Fail fast to prevent accidental expansion of public API surface.
		// (Guard added 2025-09-27)
		// NOTE: If a future controlled injection point is required, prefer an interface adapter.
			
		// Provide method list to aid debugging.
		methods := make([]string,0,len(mset))
		for k := range mset { methods = append(methods,k) }
		
		//nolint:goerr113 // test failure formatting
		 t.Fatalf("unexpected public method HealthEvaluatorForTest present; methods=%v", methods)
	}
}
