package engine

import (
	"context"
	"net/url"
	"testing"

	engmodels "github.com/99souls/ariadne/engine/models"
)

// TestExtendedDiscoveryCoversPreloadSourceAndDocs validates new discovery coverage (Iteration 7 follow-up).
func TestExtendedDiscoveryCoversPreloadSourceAndDocs(t *testing.T) {
	orig := fetchAsset
	fetchAsset = func(ctx context.Context, rawURL string, capRemaining int64) ([]byte, error) { return []byte("x"), nil }
	defer func() { fetchAsset = orig }()

	html := `<html><head>
	<link rel="preload" as="image" href="/img/hero.png">
	<link rel="preload" as="script" href="/js/app.js">
	</head><body>
	<picture><source srcset="/img/pic-320w.png 320w, /img/pic-640w.png 640w"></picture>
	<a href="/docs/guide.pdf">Guide</a>
	<a href="/files/report.docx">Report</a>
	</body></html>`

	cfg := Defaults()
	cfg.AssetPolicy.Enabled = true
	cfg.AssetPolicy.AllowTypes = []string{"img", "script", "stylesheet", "doc"}
	cfg.AssetPolicy.MaxConcurrent = 4
	cfg.AssetPolicy.Optimize = false

	eng, err := New(cfg)
	if err != nil { t.Fatalf("engine construction failed: %v", err) }
	u, _ := url.Parse("https://example.com/")
	page := &engmodels.Page{URL: u, Content: html, Title: "Extended"}
	if eng.pl.Config().AssetProcessingHook == nil { t.Fatalf("missing hook") }

	_, err = eng.pl.Config().AssetProcessingHook(context.Background(), page)
	if err != nil { t.Fatalf("hook error: %v", err) }
	m := eng.AssetMetricsSnapshot()
	// Expect at least image preload, script preload, first source[srcset] candidate, and two doc links -> >=5 selected
	if m.Selected < 5 { t.Fatalf("expected >=5 selected, got %d (snapshot=%+v)", m.Selected, m) }
}
