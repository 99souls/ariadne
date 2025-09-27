package metrics

import (
	legacy "ariadne/packages/engine/monitoring"
	"net/http/httptest"
	"testing"
)

func TestNoopProviderBasic(t *testing.T) {
	p := NewNoopProvider()
	c := p.NewCounter(CounterOpts{CommonOpts: CommonOpts{Name: "test_counter"}})
	g := p.NewGauge(GaugeOpts{CommonOpts: CommonOpts{Name: "test_gauge"}})
	h := p.NewHistogram(HistogramOpts{CommonOpts: CommonOpts{Name: "test_hist"}})
	timerCtor := p.NewTimer(HistogramOpts{CommonOpts: CommonOpts{Name: "test_timer_seconds"}})

	c.Inc(5)
	g.Set(10)
	g.Add(-3)
	h.Observe(123)
	timer := timerCtor()
	timer.ObserveDuration()
}

func TestPrometheusProviderRegistration(t *testing.T) {
	p := NewPrometheusProvider(PrometheusProviderOptions{})
	c := p.NewCounter(CounterOpts{CommonOpts: CommonOpts{Name: "events_total", Help: "total events", Labels: []string{"type"}}})
	c.Inc(1, "test")

	// ensure handler renders without panic
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	p.MetricsHandler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if len(rr.Body.Bytes()) == 0 {
		t.Fatal("expected some metrics output")
	}
}

func TestBusinessCollectorAdapter(t *testing.T) {
	legacyCollector := legacy.NewBusinessMetricsCollector()
	// Simulate some legacy metrics
	legacyCollector.RecordRuleEvaluation("p1", "r1", 10, true)
	legacyCollector.RecordRuleEvaluation("p1", "r1", 5, false)
	legacyCollector.RecordStrategyExecution("s1", 5, 10, 8)
	legacyCollector.RecordBusinessOutcome("o1", 3, map[string]interface{}{})

	prov := NewPrometheusProvider(PrometheusProviderOptions{})
	adapter := NewBusinessCollectorAdapter(legacyCollector, prov)
	if adapter == nil {
		t.Fatal("adapter nil")
	}
	adapter.SyncOnce()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	prov.MetricsHandler().ServeHTTP(rr, req)
	body := rr.Body.String()
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !contains(body, "ariadne_business_rule_evaluations_total") {
		t.Fatalf("expected rule evaluations metric, got body=%s", body)
	}
	if !contains(body, "ariadne_business_strategy_executions_total") {
		t.Fatalf("expected strategy executions metric")
	}
	if !contains(body, "ariadne_business_business_outcomes_total") {
		t.Fatalf("expected business outcomes metric")
	}
}

// contains helper (avoid pulling strings package multiple times test)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && ((len(substr) == 0) || (Index(s, substr) >= 0))
}

// Index is a minimal substring search to avoid additional imports; naive O(n*m) suffices for tests.
func Index(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
