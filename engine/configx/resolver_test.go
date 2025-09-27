package configx

import (
	"testing"
	"time"
)

func TestResolverBasicPrecedence(t *testing.T) {
	r := NewResolver()
	layers := map[int]*EngineConfigSpec{
		LayerGlobal: {
			Global: &GlobalConfigSection{MaxConcurrency: 5, LoggingLevel: "info"},
			Crawling: &CrawlingConfigSection{RateRules: &RateLimitConfig{DefaultDelay: 100 * time.Millisecond}},
		},
		LayerEnvironment: {
			Global: &GlobalConfigSection{MaxConcurrency: 10}, // overrides global
		},
		LayerSite: {
			Global: &GlobalConfigSection{LoggingLevel: "debug"}, // overrides earlier
			Crawling: &CrawlingConfigSection{RateRules: &RateLimitConfig{DefaultDelay: 50 * time.Millisecond}},
		},
	}
	final := r.Resolve(layers)
	if final.Global == nil || final.Crawling == nil || final.Crawling.RateRules == nil {
		t.Fatalf("expected merged sections to be non-nil")
	}
	if final.Global.MaxConcurrency != 10 { // env layer override
		t.Fatalf("expected MaxConcurrency=10 got %d", final.Global.MaxConcurrency)
	}
	if final.Global.LoggingLevel != "debug" { // site layer override
		t.Fatalf("expected LoggingLevel=debug got %s", final.Global.LoggingLevel)
	}
	if final.Crawling.RateRules.DefaultDelay != 50*time.Millisecond { // site layer overrides env/global
		t.Fatalf("expected DefaultDelay=50ms got %s", final.Crawling.RateRules.DefaultDelay)
	}
}

func TestResolverMapMerging(t *testing.T) {
	r := NewResolver()
	global := &EngineConfigSpec{Crawling: &CrawlingConfigSection{SiteRules: map[string]*SiteCrawlerRule{
		"example.com": {MaxDepth: 1},
	}}}
	domain := &EngineConfigSpec{Crawling: &CrawlingConfigSection{SiteRules: map[string]*SiteCrawlerRule{
		"example.com": {MaxDepth: 3}, // override
		"newsite.org": {MaxDepth: 2},
	}}}
	final := r.Resolve(map[int]*EngineConfigSpec{LayerGlobal: global, LayerDomain: domain})
	if got := final.Crawling.SiteRules["example.com"].MaxDepth; got != 3 {
		t.Fatalf("expected override depth 3 got %d", got)
	}
	if _, ok := final.Crawling.SiteRules["newsite.org"]; !ok {
		t.Fatalf("expected newsite.org to be present")
	}
	// Mutation safety: modifying source after resolve must not affect final.
	global.Crawling.SiteRules["example.com"].MaxDepth = 99
	if final.Crawling.SiteRules["example.com"].MaxDepth == 99 {
		t.Fatalf("final structure mutated after source change")
	}
}

func TestResolverSliceReplacement(t *testing.T) {
	r := NewResolver()
	specA := &EngineConfigSpec{Processing: &ProcessingConfigSection{ExtractionRules: []string{"a", "b"}}}
	specB := &EngineConfigSpec{Processing: &ProcessingConfigSection{ExtractionRules: []string{"x"}}}
	final := r.Resolve(map[int]*EngineConfigSpec{LayerGlobal: specA, LayerSite: specB})
	if len(final.Processing.ExtractionRules) != 1 || final.Processing.ExtractionRules[0] != "x" {
		t.Fatalf("expected slice replacement by higher layer")
	}
	// Ensure slice was cloned.
	specB.Processing.ExtractionRules[0] = "mutated"
	if final.Processing.ExtractionRules[0] == "mutated" {
		t.Fatalf("expected cloning of slice to prevent mutation propagation")
	}
}
