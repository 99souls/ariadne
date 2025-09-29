package crawler_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	engcrawler "github.com/99souls/ariadne/engine/internal/crawler"
	"github.com/99souls/ariadne/engine/internal/testutil/testsite"
	"github.com/99souls/ariadne/engine/models"
)

// TestLiveSiteLatencyDistribution exercises the /api/slow endpoint multiple times to build a
// latency distribution sample and asserts tight variance around the deterministic 500ms delay.
// This establishes a baseline harness for future introduction of controlled jitter while
// protecting current deterministic expectations.
func TestLiveSiteLatencyDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	testsite.WithLiveTestSite(t, func(base string) {
		u, err := url.Parse(base)
		if err != nil {
			t.Fatalf("parse base: %v", err)
		}
		// We'll directly issue HTTP requests to isolate pure endpoint latency measurement,
		// then perform one crawler run to ensure integration still within wall-clock bounds.
		const samples = 5
		delays := make([]time.Duration, 0, samples)
		client := &http.Client{Timeout: 2 * time.Second}
		for i := 0; i < samples; i++ {
			start := time.Now()
			resp, err := client.Get(base + "/api/slow")
			if err != nil {
				t.Fatalf("slow endpoint request %d: %v", i, err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("unexpected status %d", resp.StatusCode)
			}
			delays = append(delays, time.Since(start))
		}
		// Compute min, max.
		var min, max time.Duration
		for i, d := range delays {
			if i == 0 || d < min {
				min = d
			}
			if i == 0 || d > max {
				max = d
			}
		}
		// Expect tight envelope: 500ms +/- ~120ms headroom (process + scheduling).
		if min < 400*time.Millisecond || max > 650*time.Millisecond {
			t.Fatalf("unexpected latency envelope min=%v max=%v delays=%v", min, max, delays)
		}

		// Integration crawl to ensure presence does not exceed 2.5s total.
		cfg := models.DefaultConfig()
		cfg.AllowedDomains = []string{u.Host}
		cfg.StartURL = base + "/"
		cfg.MaxPages = 25
		cfg.Timeout = 4 * time.Second
		cfg.RequestDelay = 50 * time.Millisecond
		cfg.RespectRobots = true
		crawler := engcrawler.New(cfg)
		startCrawl := time.Now()
		if err := crawler.Start(cfg.StartURL); err != nil {
			t.Fatalf("start crawler: %v", err)
		}
		slowURL := base + "/api/slow"
		sawSlow := false
		deadline := time.Now().Add(6 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case r := <-crawler.Results():
				if r == nil {
					break
				}
				if r.URL == slowURL {
					sawSlow = true
				}
			case <-time.After(100 * time.Millisecond):
			}
			if sawSlow {
				break
			}
		}
		crawler.Stop()
		crawlDur := time.Since(startCrawl)
		if !sawSlow {
			t.Fatalf("crawler did not hit slow endpoint")
		}
		if crawlDur > 2500*time.Millisecond {
			t.Fatalf("crawl duration too high: %v", crawlDur)
		}
	})
}
