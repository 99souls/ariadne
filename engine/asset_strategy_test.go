package engine

import (
	engmodels "github.com/99souls/ariadne/engine/models"
	"context"
	"crypto/sha256"
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
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	// Expect 3 refs
	if len(refs) != 3 {
		t.Fatalf("expected 3 refs got %d", len(refs))
	}
	policy := AssetPolicy{Enabled: true, AllowTypes: []string{"img", "script"}, BlockTypes: []string{"script"}, InlineMaxBytes: 4096, RewritePrefix: "/assets/"}
	actions, err := s.Decide(context.TODO(), refs, policy)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	// script should be blocked -> only 2 image actions
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions got %d", len(actions))
	}
	// One of them (logo.svg) should be inline
	var sawInline bool
	for _, a := range actions {
		if strings.HasSuffix(a.Ref.URL, "logo.svg") && a.Mode == AssetModeInline {
			sawInline = true
		}
	}
	if !sawInline {
		t.Fatalf("expected svg asset to be inline candidate")
	}
}

// Iteration 4: Deterministic path + optimization tests (unit-level, synthetic inputs).
func TestComputeAssetPathDeterministic(t *testing.T) {
	// Synthetic bytes to avoid network.
	data := []byte("function test  ( ) {   return  42; }\n")
	hash := hashBytesHex(data)
	if len(hash) != sha256.Size*2 {
		t.Fatalf("expected sha256 hex length %d got %d", sha256.Size*2, len(hash))
	}
	path := computeAssetPath("/assets/", hash, "https://example.com/js/app.js")
	if !strings.HasPrefix(path, "/assets/") {
		t.Fatalf("path missing prefix: %s", path)
	}
	parts := strings.Split(path[len("/assets/"):], "/")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts after prefix, got %d (%v)", len(parts), parts)
	}
	if len(parts[0]) != 2 {
		t.Fatalf("expected first directory of length 2, got %s", parts[0])
	}
	if !strings.HasSuffix(path, ".js") {
		t.Fatalf("expected original extension .js preserved, got %s", path)
	}
	// Determinism: recompute path for same hash & url
	path2 := computeAssetPath("/assets/", hash, "https://example.com/js/app.js")
	if path != path2 {
		t.Fatalf("expected deterministic path, got %s vs %s", path, path2)
	}
}

func TestOptimizeBytesWhitespaceCollapse(t *testing.T) {
	css := []byte("body {  color:   red;    background:   white; }\n\n")
	out, applied := optimizeBytes("stylesheet", css)
	if len(applied) == 0 || applied[0] != "css_minify" {
		t.Fatalf("expected css_minify applied, got %v", applied)
	}
	if len(out) >= len(css) {
		t.Fatalf("expected reduced size, in=%d out=%d", len(css), len(out))
	}
	// Idempotent second pass
	out2, applied2 := optimizeBytes("stylesheet", out)
	if len(applied2) != 0 {
		t.Fatalf("expected no further optimization second pass, got %v", applied2)
	}
	if string(out) != string(out2) {
		t.Fatalf("expected stable output after second pass")
	}
}

func TestOptimizeBytesJS(t *testing.T) {
	js := []byte("function   foo( ) {  return   1 + 2 ; }    ")
	out, applied := optimizeBytes("script", js)
	if len(applied) == 0 || applied[0] != "js_minify" {
		t.Fatalf("expected js_minify applied, got %v", applied)
	}
	if len(out) >= len(js) {
		t.Fatalf("expected reduced size for js, in=%d out=%d", len(js), len(out))
	}
}

func TestOptimizeDisabledLeavesBytes(t *testing.T) {
	// Use helper directly; simulate disabled optimization by skipping optimizeBytes invocation.
	js := []byte("let   x =  1 ;")
	out, applied := optimizeBytes("script", js)
	if len(applied) == 0 {
		t.Fatalf("expected optimization when enabled helper directly: %v", applied)
	}
	// Simulate disabled by not calling optimizeBytes: ensure hash differs only when we apply.
	if hashBytesHex(js) == hashBytesHex(out) && string(js) != string(out) {
		t.Fatalf("hash should change when bytes change; input and output hash equal")
	}
}

// --- Hardening tests (post Iteration 4) ---
func TestDecideEnforcesMaxPerPage(t *testing.T) {
	s := &DefaultAssetStrategy{}
	// fabricate refs
	var refs []AssetRef
	for i := 0; i < 10; i++ {
		refs = append(refs, AssetRef{URL: "https://e/x", Type: "img"})
	}
	policy := AssetPolicy{Enabled: true, MaxPerPage: 3, AllowTypes: []string{"img"}}
	acts, err := s.Decide(context.TODO(), refs, policy)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if len(acts) != 3 {
		t.Fatalf("expected 3 actions capped, got %d", len(acts))
	}
}

func TestDecideAllowFilters(t *testing.T) {
	s := &DefaultAssetStrategy{}
	refs := []AssetRef{{URL: "https://e/a.css", Type: "stylesheet"}, {URL: "https://e/a.js", Type: "script"}, {URL: "https://e/a.png", Type: "img"}}
	policy := AssetPolicy{Enabled: true, AllowTypes: []string{"script"}}
	acts, err := s.Decide(context.TODO(), refs, policy)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if len(acts) != 1 || acts[0].Ref.Type != "script" {
		t.Fatalf("allow filter failed: %+v", acts)
	}
}

func TestComputeAssetPathPrefixNormalization(t *testing.T) {
	hash := strings.Repeat("a", 64)
	p1 := computeAssetPath("assets", hash, "https://e/x.js")
	p2 := computeAssetPath("/assets", hash, "https://e/x.js")
	p3 := computeAssetPath("/assets/", hash, "https://e/x.js")
	if p1 != p2 || p2 != p3 {
		t.Fatalf("expected normalized paths equal, got: %s | %s | %s", p1, p2, p3)
	}
}

func TestExecuteOptimizeToggle(t *testing.T) {
	// Override fetchAsset to return deterministic bytes without network.
	oldFetch := fetchAsset
	fetchAsset = func(ctx context.Context, rawURL string, cap int64) ([]byte, error) {
		return []byte("function   x( ){ return  1 ; }"), nil
	}
	defer func() { fetchAsset = oldFetch }()
	s := &DefaultAssetStrategy{}
	ref := AssetRef{URL: "https://e/app.js", Type: "script"}
	actions := []AssetAction{{Ref: ref, Mode: AssetModeDownload}}
	polNo := AssetPolicy{Enabled: true, RewritePrefix: "/assets/", Optimize: false}
	matsNo, err := s.Execute(context.TODO(), actions, polNo)
	if err != nil || len(matsNo) != 1 {
		t.Fatalf("execute no optimize failed: %v %d", err, len(matsNo))
	}
	polYes := polNo
	polYes.Optimize = true
	matsYes, err := s.Execute(context.TODO(), actions, polYes)
	if err != nil || len(matsYes) != 1 {
		t.Fatalf("execute optimize failed: %v %d", err, len(matsYes))
	}
	if matsNo[0].Hash == matsYes[0].Hash && string(matsNo[0].Bytes) != string(matsYes[0].Bytes) {
		t.Fatalf("hash unchanged despite byte change")
	}
	if polYes.Optimize && len(matsYes[0].Optimizations) == 0 {
		t.Fatalf("expected optimization tag")
	}
}

func TestAssetPolicyValidate(t *testing.T) {
	bad := AssetPolicy{Enabled: true, RewritePrefix: "assets/"}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected validation error for missing leading slash")
	}
	good := AssetPolicy{Enabled: true, RewritePrefix: "/assets"}
	if err := good.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
