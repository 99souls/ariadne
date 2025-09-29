package crawler_test

import (
	"net/url"
	"testing"
	"time"

	engcrawler "github.com/99souls/ariadne/engine/internal/crawler"
	"github.com/99souls/ariadne/engine/internal/testutil/testsite"
	"github.com/99souls/ariadne/engine/models"
)

// TestLiveSiteLargeAssetThroughput ensures that fetching a large (~200KB) binary asset does not
// block discovery of normal HTML pages beyond an acceptable wall-clock threshold. The asset is
// linked both via a prerender <img> tag and a navigation anchor so the crawler enqueues it as an
// asset request.
func TestLiveSiteLargeAssetThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	testsite.WithLiveTestSite(t, func(base string) {
		u, err := url.Parse(base)
		if err != nil {
			t.Fatalf("parse base: %v", err)
		}
		cfg := models.DefaultConfig()
		cfg.AllowedDomains = []string{u.Host}
		cfg.StartURL = base + "/"
		cfg.MaxPages = 40
		cfg.Timeout = 5 * time.Second
		cfg.RequestDelay = 40 * time.Millisecond
		cfg.RespectRobots = true
		crawler := engcrawler.New(cfg)
		start := time.Now()
		if err := crawler.Start(cfg.StartURL); err != nil {
			t.Fatalf("start crawler: %v", err)
		}
		largeURL := base + "/static/large.bin"
		sawLarge := false
		sawAbout := false
		deadline := time.Now().Add(6 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case r := <-crawler.Results():
				if r == nil {
					break
				}
				if r.URL == largeURL && r.Stage == "asset" && r.StatusCode == 200 {
					sawLarge = true
				}
				if r.URL == base+"/about" && r.Success {
					sawAbout = true
				}
			case <-time.After(100 * time.Millisecond):
			}
			if sawLarge && sawAbout {
				break
			}
		}
		crawler.Stop()
		dur := time.Since(start)
		if !sawLarge {
			t.Fatalf("did not fetch large asset %s", largeURL)
		}
		if !sawAbout {
			t.Fatalf("did not discover normal page /about while large asset in flight")
		}
		if dur > 2500*time.Millisecond {
			t.Fatalf("crawl took too long with large asset present: %v", dur)
		}
	})
}
