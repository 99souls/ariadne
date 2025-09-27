package engine

import (
	engmodels "ariadne/packages/engine/models"
	"context"
	"net/url"
	"strings"
	"testing"
)

// TestAssetStrategyInterfacePresence ensures Iteration 1 scaffolding exists and defaults are sane.
func TestAssetStrategyInterfacePresence(t *testing.T) {
	cfg := Defaults()
	if cfg.AssetPolicy.Enabled != false {
		t.Fatalf("expected default AssetPolicy.Enabled=false, got true")
	}

	var s AssetStrategy = &DefaultAssetStrategy{}
	if s.Name() != "noop" {
		t.Fatalf("expected noop strategy name, got %s", s.Name())
	}

	// Ensure empty returns don't panic and obey contract shapes.
	refs, err := s.Discover(context.TODO(), nil)
	if err != nil {
		t.Fatalf("unexpected error in Discover: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected no refs from noop discover, got %d", len(refs))
	}

	actions, err := s.Decide(context.TODO(), refs, cfg.AssetPolicy)
	if err != nil {
		t.Fatalf("unexpected error in Decide: %v", err)
	}
	if len(actions) != 0 {
		t.Fatalf("expected no actions from noop decide, got %d", len(actions))
	}

	mats, err := s.Execute(context.TODO(), actions, cfg.AssetPolicy)
	if err != nil {
		t.Fatalf("unexpected error in Execute: %v", err)
	}
	if len(mats) != 0 {
		t.Fatalf("expected no materialized assets from noop execute, got %d", len(mats))
	}
}

// TestDefaultAssetStrategyBasicFlow exercises Iteration 2 functionality: discovery -> decide -> execute -> rewrite.
func TestDefaultAssetStrategyBasicFlow(t *testing.T) {
	s := &DefaultAssetStrategy{}
	base, _ := url.Parse("https://example.com/page.html")
	html := `<html><head><link rel="stylesheet" href="/css/site.css"><script src="/js/app.js"></script></head><body><img src="/images/logo.png"/></body></html>`
	page := &engmodels.Page{URL: base, Content: html}
	refs, err := s.Discover(context.TODO(), page)
	if err != nil {
		t.Fatalf("discover error: %v", err)
	}
	if len(refs) != 3 {
		t.Fatalf("expected 3 refs (css, js, img); got %d", len(refs))
	}
	policy := AssetPolicy{Enabled: true, MaxPerPage: 10, MaxBytes: 1024 * 1024, RewritePrefix: "/assets/", AllowTypes: []string{"img", "stylesheet", "script"}}
	actions, err := s.Decide(context.TODO(), refs, policy)
	if err != nil {
		t.Fatalf("decide error: %v", err)
	}
	if len(actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(actions))
	}
	// We cannot reliably execute HTTP downloads here (external), so simulate by short-circuiting: Expect zero materialized (no accessible network assets)
	mats, err := s.Execute(context.TODO(), actions, policy)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	// Likely zero because example.com assets won't be fetched; allow either 0 or 3 depending on network availability.
	if len(mats) != 0 && len(mats) != 3 {
		t.Fatalf("unexpected materialized count: %d", len(mats))
	}
	updated, err := s.Rewrite(context.TODO(), page, mats, policy)
	if err != nil {
		t.Fatalf("rewrite error: %v", err)
	}
	if len(mats) == 0 && updated.Content != html {
		t.Fatalf("expected unchanged HTML when no assets; diff present")
	}
	if len(mats) > 0 {
		if !strings.Contains(updated.Content, policy.RewritePrefix) {
			t.Fatalf("expected rewritten paths to contain prefix %s", policy.RewritePrefix)
		}
	}
}

func TestAssetStrategyDecisionInlineAndBlock(t *testing.T) {
	s := &DefaultAssetStrategy{}
	base, _ := url.Parse("https://example.com/")
	html := `<html><body><img src="/images/logo.svg"><img src="/images/photo.png"><script src="/js/app.js"></script></body></html>`
	page := &engmodels.Page{URL: base, Content: html}
	refs, err := s.Discover(context.TODO(), page)
	if err != nil { t.Fatalf("discover: %v", err) }
	// Expect 3 refs
	if len(refs) != 3 { t.Fatalf("expected 3 refs got %d", len(refs)) }
	policy := AssetPolicy{Enabled: true, AllowTypes: []string{"img","script"}, BlockTypes: []string{"script"}, InlineMaxBytes: 4096, RewritePrefix: "/assets/"}
	actions, err := s.Decide(context.TODO(), refs, policy)
	if err != nil { t.Fatalf("decide: %v", err) }
	// script should be blocked -> only 2 image actions
	if len(actions) != 2 { t.Fatalf("expected 2 actions got %d", len(actions)) }
	// One of them (logo.svg) should be inline
	var sawInline bool
	for _, a := range actions {
		if strings.HasSuffix(a.Ref.URL, "logo.svg") && a.Mode == AssetModeInline { sawInline = true }
	}
	if !sawInline { t.Fatalf("expected svg asset to be inline candidate") }
}
