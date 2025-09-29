package crawler_test

import (
	"net/url"
	"testing"
	"time"

	engcrawler "github.com/99souls/ariadne/engine/internal/crawler"
	"github.com/99souls/ariadne/engine/internal/testutil/testsite"
	"github.com/99souls/ariadne/engine/models"
)

// TestLiveSiteDiscovery performs an end-to-end crawl against the Bun live test site
// exercising link discovery over several routes. This replaces a portion of prior
// synthetic discovery tests by using realistic HTML + assets.
func TestLiveSiteDiscovery(t *testing.T) {
	// Requires prerender anchors in test site index.html (added 2025-09-29).
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
		cfg.MaxPages = 20
		cfg.Timeout = 2 * time.Second
		cfg.RequestDelay = 100 * time.Millisecond
		// Enable robots respect to exercise allow-mode (default) behavior; site serves Allow: / variant
		cfg.RespectRobots = true
		crawler := engcrawler.New(cfg)
		if err := crawler.Start(cfg.StartURL); err != nil {
			t.Fatalf("start crawler: %v", err)
		}

		// Collect results with polling until expected routes discovered or timeout.
		found := make(map[string]struct{})
		deadline := time.Now().Add(6 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case r := <-crawler.Results():
				if r == nil { // channel closed
					break
				}
				if r.Success && r.URL != "" {
					found[r.URL] = struct{}{}
				}
			case <-time.After(100 * time.Millisecond):
				// continue
			}
			if hasAll(found, base+"/about", base+"/docs/getting-started", base+"/blog", base+"/tags") {
				break
			}
		}
		if !hasAll(found, base+"/about", base+"/docs/getting-started", base+"/blog", base+"/tags") {
			crawler.Stop()
			// Provide diagnostic output
			t.Fatalf("did not discover expected core pages; got %d: %#v", len(found), found)
		}
		crawler.Stop()

		// Assertions
		if len(found) < 4 {
			t.Fatalf("expected >=4 pages, got %d: %#v", len(found), found)
		}
		mustContain(t, found, base+"/about")
		mustContain(t, found, base+"/docs/getting-started")
		mustContain(t, found, base+"/blog")
		mustContain(t, found, base+"/tags")
	})
}

// TestLiveSiteRobotsDeny verifies that when the live test site is started in robots deny mode
// and RespectRobots is enabled, the crawler does not fetch any content pages beyond attempting
// the start URL (which is itself blocked after fetching robots.txt).
func TestLiveSiteRobotsDeny(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	// Force deny robots mode for the live site instance started by the harness.
	t.Setenv("TESTSITE_ROBOTS", "deny")
	testsite.WithLiveTestSite(t, func(base string) {
		u, err := url.Parse(base)
		if err != nil {
			t.Fatalf("parse base: %v", err)
		}
		cfg := models.DefaultConfig()
		cfg.AllowedDomains = []string{u.Host}
		cfg.StartURL = base + "/"
		cfg.MaxPages = 10
		cfg.Timeout = 2 * time.Second
		cfg.RequestDelay = 50 * time.Millisecond
		cfg.RespectRobots = true
		crawler := engcrawler.New(cfg)
		if err := crawler.Start(cfg.StartURL); err != nil {
			t.Fatalf("start crawler: %v", err)
		}
		// Collect for a bounded interval; expect zero successful page results.
		found := 0
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case r := <-crawler.Results():
				if r == nil { // channel closed
					break
				}
				if r.Success && r.URL != "" {
					found++
				}
			case <-time.After(100 * time.Millisecond):
				// continue polling
			}
			if found > 0 {
				break
			}
		}
		crawler.Stop()
		if found != 0 {
			t.Fatalf("expected 0 pages due to robots deny-all; got %d", found)
		}
		stats := crawler.Stats()
		if stats.ProcessedPages != 0 {
			t.Fatalf("expected 0 processed pages; got %d", stats.ProcessedPages)
		}
	})
}

// TestLiveSiteDepthLimit verifies that MaxDepth prevents the crawler from reaching
// the deeply nested labs leaf route (depth 5) when configured below that level.
func TestLiveSiteDepthLimit(t *testing.T) {
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
		cfg.MaxPages = 50
		cfg.Timeout = 3 * time.Second
		cfg.RequestDelay = 50 * time.Millisecond
		cfg.RespectRobots = true
		// Set MaxDepth to 4 so that depth-5 leaf is excluded.
		cfg.MaxDepth = 4
		crawler := engcrawler.New(cfg)
		if err := crawler.Start(cfg.StartURL); err != nil {
			t.Fatalf("start crawler: %v", err)
		}
		leaf := base + "/labs/depth/depth2/depth3/leaf"
		found := make(map[string]struct{})
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case r := <-crawler.Results():
				if r == nil {
					break
				}
				if r.Success && r.URL != "" {
					found[r.URL] = struct{}{}
					if r.URL == leaf { // early exit if (unexpectedly) found
						deadline = time.Now() // break outer loop
					}
				}
			case <-time.After(100 * time.Millisecond):
				// continue
			}
		}
		crawler.Stop()
		if _, ok := found[leaf]; ok {
			t.Fatalf("leaf page should not be discovered with MaxDepth=4; discovered set=%#v", found)
		}
		// Sanity: should still have discovered at least one docs/blog page.
		if !hasAny(found, base+"/about", base+"/docs/getting-started", base+"/blog") {
			t.Fatalf("expected to discover some core pages; got %#v", found)
		}
	})
}

func hasAny(set map[string]struct{}, keys ...string) bool {
	for _, k := range keys {
		if _, ok := set[k]; ok {
			return true
		}
	}
	return false
}

func mustContain(t *testing.T, set map[string]struct{}, key string) {
	t.Helper()
	if _, ok := set[key]; !ok {
		t.Fatalf("expected to discover %s; set=%#v", key, set)
	}
}

func hasAll(set map[string]struct{}, keys ...string) bool {
	for _, k := range keys {
		if _, ok := set[k]; !ok {
			return false
		}
	}
	return true
}
