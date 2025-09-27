package engine

import (
	"context"
	"errors"
	"net/url"
	"testing"

	engmodels "github.com/99souls/ariadne/engine/models"
)

// TestAssetFailedDownloadMetrics ensures failed counter increments and bytes not counted.
func TestAssetFailedDownloadMetrics(t *testing.T) {
	orig := fetchAsset
	fetchAsset = func(ctx context.Context, rawURL string, capRemaining int64) ([]byte, error) {
		return nil, errors.New("boom")
	}
	defer func() { fetchAsset = orig }()

	cfg := Defaults()
	cfg.AssetPolicy.Enabled = true
	cfg.AssetPolicy.AllowTypes = []string{"img"}

	eng, err := New(cfg)
	if err != nil { t.Fatalf("engine construction failed: %v", err) }

	hook := eng.pl.Config().AssetProcessingHook
	if hook == nil { t.Fatalf("expected asset processing hook") }

	u, _ := url.Parse("https://example.com/")
	page := &engmodels.Page{URL:u, Content:"<img src=\"/a.png\">"}
	_, _ = hook(context.Background(), page)

	snap := eng.AssetMetricsSnapshot()
	if snap.Discovered != 1 || snap.Selected != 1 { t.Fatalf("unexpected discovered/selected %+v", snap) }
	if snap.Downloaded != 0 { t.Errorf("expected 0 downloaded, got %d", snap.Downloaded) }
	if snap.Failed != 1 { t.Errorf("expected 1 failed, got %d", snap.Failed) }
	if snap.BytesIn != 0 || snap.BytesOut != 0 { t.Errorf("expected zero bytes accounted, got in=%d out=%d", snap.BytesIn, snap.BytesOut) }
}
