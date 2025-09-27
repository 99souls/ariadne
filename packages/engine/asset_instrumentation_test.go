package engine

import (
	"context"
	"net/url"
	"regexp"
	"testing"

	engmodels "ariadne/packages/engine/models"
)

// TestAssetInstrumentationAndDeterminism validates metrics/events and deterministic rewrite output.
func TestAssetInstrumentationAndDeterminism(t *testing.T) {
	// Save original fetch and restore after test
	origFetch := fetchAsset
	fetchAsset = func(ctx context.Context, rawURL string, capRemaining int64) ([]byte, error) {
		switch {
		case regexp.MustCompile(`site\.css$`).MatchString(rawURL):
			return []byte("body {  color: red;   }"), nil
		case regexp.MustCompile(`app\.js$`).MatchString(rawURL):
			return []byte("function test ( ) {  return  42 ; }"), nil
		case regexp.MustCompile(`logo\.svg$`).MatchString(rawURL):
			return []byte("<svg>   </svg>"), nil
		default:
			return []byte("data"), nil
		}
	}
	defer func() { fetchAsset = origFetch }()

	cfg := Defaults()
	cfg.AssetPolicy.Enabled = true
	cfg.AssetPolicy.Optimize = true
	cfg.AssetPolicy.AllowTypes = []string{"img", "stylesheet", "script"}

	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("engine construction failed: %v", err)
	}
	if eng.assetStrategy == nil {
		t.Fatalf("expected asset strategy to be initialized")
	}

	html := `<html><head><link rel="stylesheet" href="/styles/site.css"></head><body><img src="/images/logo.svg"><script src="/js/app.js"></script></body></html>`
	u, _ := url.Parse("https://example.com/")
	page := &engmodels.Page{URL: u, Content: html, Title: "Test"}

	hook := eng.pl.Config().AssetProcessingHook
	if hook == nil {
		t.Fatalf("expected asset processing hook")
	}

	mutated1, err := hook(context.Background(), page)
	if err != nil {
		t.Fatalf("hook returned error: %v", err)
	}
	if mutated1 == nil || mutated1.Content == html {
		t.Fatalf("expected rewritten content different from original")
	}
	// Ensure original references are gone
	for _, orig := range []string{"/styles/site.css", "/images/logo.svg", "/js/app.js"} {
		if regexp.MustCompile(regexp.QuoteMeta(orig)).MatchString(mutated1.Content) {
			t.Errorf("original reference %s still present after rewrite", orig)
		}
	}
	// Ensure hashed asset paths present
	reCSS := regexp.MustCompile(`/assets/[0-9a-f]{2}/[0-9a-f]{64}\.css`)
	reJS := regexp.MustCompile(`/assets/[0-9a-f]{2}/[0-9a-f]{64}\.js`)
	reSVG := regexp.MustCompile(`/assets/[0-9a-f]{2}/[0-9a-f]{64}\.svg`)
	if !reCSS.MatchString(mutated1.Content) {
		t.Errorf("missing rewritten css path")
	}
	if !reJS.MatchString(mutated1.Content) {
		t.Errorf("missing rewritten js path")
	}
	if !reSVG.MatchString(mutated1.Content) {
		t.Errorf("missing rewritten svg path")
	}

	snap1 := eng.AssetMetricsSnapshot()
	if snap1.Discovered != 3 || snap1.Selected != 3 || snap1.Downloaded != 3 {
		t.Errorf("unexpected metrics snapshot: %+v", snap1)
	}
	if snap1.Optimized != 3 { // css/js + svg meta tag
		t.Errorf("expected 3 optimized assets, got %d", snap1.Optimized)
	}

	// Collect events
	events := eng.AssetEvents()
	downloads := 0
	for _, ev := range events {
		if ev.Type == "asset_download" {
			downloads++
		}
	}
	if downloads != 3 {
		t.Errorf("expected 3 download events, got %d", downloads)
	}
	// Optimization events are now coalesced into download events (Optimizations field present)

	// Determinism: run again on a fresh clone of original page
	page2 := &engmodels.Page{URL: u, Content: html, Title: "Test"}
	mutated2, err := hook(context.Background(), page2)
	if err != nil {
		t.Fatalf("second hook run error: %v", err)
	}
	if mutated2.Content != mutated1.Content {
		t.Errorf("expected deterministic rewrite; contents differ")
	}
}
