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
		cfg.RespectRobots = false
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
