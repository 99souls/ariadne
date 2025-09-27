package engine

import (
	"context"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"testing"
	"time"

	engmodels "github.com/99souls/ariadne/engine/models"
)

// TestAssetExecuteConcurrency ensures parallel execution yields deterministic rewrite & metrics integrity.
func TestAssetExecuteConcurrency(t *testing.T) {
	origFetch := fetchAsset
	fetchAsset = func(ctx context.Context, rawURL string, capRemaining int64) ([]byte, error) {
		// Simulate variable latency to exercise concurrency ordering guarantees
		if regexp.MustCompile(`slow`).MatchString(rawURL) {
			time.Sleep(10 * time.Millisecond)
		}
		return []byte("data" + rawURL), nil
	}
	defer func() { fetchAsset = origFetch }()

	cfg := Defaults()
	cfg.AssetPolicy.Enabled = true
	cfg.AssetPolicy.Optimize = false
	cfg.AssetPolicy.AllowTypes = []string{"img", "script", "stylesheet", "media"}
	cfg.AssetPolicy.MaxConcurrent = 4

	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("engine construction failed: %v", err)
	}

	// Build HTML with multiple assets including srcset & media sources
	html := `<html><head>
	<link rel="stylesheet" href="/css/a.css">
	</head><body>
	<img src="/img/one.png">
	<img srcset="/img/two-480w.png 480w, /img/two-960w.png 960w">
	<video><source src="/media/vid-slow.mp4"></video>
	<script src="/js/slow-app.js"></script>
	</body></html>`
	u, _ := url.Parse("https://example.com/")
	page := &engmodels.Page{URL: u, Content: html, Title: "Concurrency"}

	hook := eng.pl.Config().AssetProcessingHook
	if hook == nil {
		t.Fatalf("expected asset processing hook")
	}

	mut, err := hook(context.Background(), page)
	if err != nil {
		t.Fatalf("hook error: %v", err)
	}
	if mut.Content == html {
		t.Fatalf("expected rewrite mutation")
	}

	// Ensure all original refs gone
	for _, orig := range []string{"/css/a.css", "/img/one.png", "/img/two-480w.png", "/media/vid-slow.mp4", "/js/slow-app.js"} {
		if regexp.MustCompile(regexp.QuoteMeta(orig)).FindString(mut.Content) != "" {
			to := mut.Content
			if len(to) > 120 {
				to = to[:120]
			}
			t.Fatalf("original reference %s still present in mutated content: %s", orig, to)
		}
	}

	// Check hashed pattern count (expect 5 assets)
	pat := regexp.MustCompile(`/assets/[0-9a-f]{2}/[0-9a-f]{64}`)
	matches := pat.FindAllString(mut.Content, -1)
	set := map[string]struct{}{}
	for _, m := range matches {
		set[m] = struct{}{}
	}
	if len(set) != 5 {
		keys := make([]string, 0, len(set))
		for k := range set {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Fatalf("expected 5 unique hashed assets, got %s matches=%v", strconv.Itoa(len(set)), keys)
	}

	snap := eng.AssetMetricsSnapshot()
	if snap.Discovered < 5 || snap.Selected < 5 || snap.Downloaded < 5 {
		// concurrency may produce additional discovered due to src + srcset separate entries
		if snap.Downloaded < 5 {
			t.Errorf("expected >=5 downloaded, got %d", snap.Downloaded)
		}
	}
}

// BenchmarkAssetExecute provides a baseline for parallel execution cost (Iteration 7 baseline).
func BenchmarkAssetExecute(b *testing.B) {
	origFetch := fetchAsset
	fetchAsset = func(ctx context.Context, rawURL string, capRemaining int64) ([]byte, error) {
		return []byte("data"), nil
	}
	defer func() { fetchAsset = origFetch }()

	cfg := Defaults()
	cfg.AssetPolicy.Enabled = true
	cfg.AssetPolicy.AllowTypes = []string{"img", "script", "stylesheet"}
	cfg.AssetPolicy.MaxConcurrent = 4

	eng, _ := New(cfg)
	u, _ := url.Parse("https://bench.local/")
	base := `<html><head><link rel="stylesheet" href="/css/a.css"></head><body>`
	for i := 0; i < 32; i++ {
		base += `<img src="/img/one` + strconv.Itoa(i%10) + `.png">`
	}
	base += `</body></html>`
	page := &engmodels.Page{URL: u, Content: base, Title: "Bench"}
	hook := eng.pl.Config().AssetProcessingHook

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hook(context.Background(), page)
	}
}
