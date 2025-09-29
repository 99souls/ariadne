package crawler_test

import (
	"net/url"
	"net/http"
	"io"
	"encoding/json"
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

// TestLiveSiteBrokenAsset ensures that a missing image (404) is surfaced as an asset result
// without preventing normal HTML page discovery.
func TestLiveSiteBrokenAsset(t *testing.T) {
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
		cfg.Timeout = 3 * time.Second
		cfg.RequestDelay = 50 * time.Millisecond
		cfg.RespectRobots = true
		crawler := engcrawler.New(cfg)
		if err := crawler.Start(cfg.StartURL); err != nil {
			t.Fatalf("start crawler: %v", err)
		}
		brokenPath := base + "/static/img/missing.png"
		foundPages := make(map[string]struct{})
		sawBroken := false
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case r := <-crawler.Results():
				if r == nil {
					break
				}
				if r.Stage == "asset" && r.URL == brokenPath && r.StatusCode >= 400 {
					sawBroken = true
				}
				if r.Success && r.Stage == "crawl" && r.URL != "" {
					foundPages[r.URL] = struct{}{}
				}
			case <-time.After(100 * time.Millisecond):
			}
			if sawBroken && hasAll(foundPages, base+"/about") {
				break
			}
		}
		crawler.Stop()
		if !sawBroken {
			t.Fatalf("did not observe failing status (>=400) for broken asset %s", brokenPath)
		}
		if !hasAll(foundPages, base+"/about") {
			t.Fatalf("expected to still discover normal pages; got %#v", foundPages)
		}
	})
}

// TestLiveSiteSlowEndpoint verifies that the presence of the slow API endpoint
// (linked via hidden prerender anchor) does not cause the total crawl to exceed
// a reasonable wall-clock duration. The endpoint adds ~400-600ms latency; we
// assert overall crawl completes within ~2s while still discovering normal pages.
func TestLiveSiteSlowEndpoint(t *testing.T) {
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
		cfg.MaxPages = 30
		cfg.Timeout = 4 * time.Second
		cfg.RequestDelay = 50 * time.Millisecond
		cfg.RespectRobots = true
		crawler := engcrawler.New(cfg)
		start := time.Now()
		if err := crawler.Start(cfg.StartURL); err != nil {
			t.Fatalf("start crawler: %v", err)
		}
		slowURL := base + "/api/slow"
		sawSlow := false
		foundCore := make(map[string]struct{})
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
				if r.Success && r.Stage == "crawl" && r.URL != "" {
					foundCore[r.URL] = struct{}{}
				}
			case <-time.After(100 * time.Millisecond):
			}
			if sawSlow && hasAll(foundCore, base+"/about") {
				break
			}
		}
		crawler.Stop()
		duration := time.Since(start)
		if !sawSlow {
			t.Fatalf("did not fetch slow endpoint %s", slowURL)
		}
		if !hasAll(foundCore, base+"/about") {
			t.Fatalf("expected to discover core page /about; discovered=%#v", foundCore)
		}
		// Expect total wall time to stay under 2 seconds + small headroom (slow endpoint ~600ms max + request delays).
		if duration > 2*time.Second+500*time.Millisecond {
			t.Fatalf("slow endpoint caused excessive total crawl time: %v", duration)
		}
	})
}

// TestLiveSiteReuseSingleInstance verifies that when TESTSITE_REUSE=1 and a fixed TESTSITE_PORT are set,
// multiple invocations of the harness within a single test process observe the same underlying Bun process
// (by comparing the /api/instance id). This guards against accidental respawns that would add test latency.
func TestLiveSiteReuseSingleInstance(t *testing.T) {
	if testing.Short() { t.Skip("short mode") }
	// Pick a deterministic port unlikely to collide in local dev; could be overridden by env if needed.
	t.Setenv("TESTSITE_PORT", "5179")
	t.Setenv("TESTSITE_REUSE", "1")
	fetchID := func(base string) string {
		resp, err := http.Get(base + "/api/instance")
		if err != nil { t.Fatalf("fetch instance id: %v", err) }
		defer func(){ _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		var obj struct{ ID string `json:"id"`; StartedAt string `json:"startedAt"` }
		if err := json.Unmarshal(b, &obj); err != nil { t.Fatalf("unmarshal instance id: %v body=%s", err, string(b)) }
		if obj.ID == "" { t.Fatalf("empty instance id; body=%s", string(b)) }
		return obj.ID
	}
	var firstID string
	// First invocation starts (or reuses) instance.
	testsite.WithLiveTestSite(t, func(base string){
		firstID = fetchID(base)
	})
	if firstID == "" { t.Fatalf("first instance id empty") }
	// Second invocation should reuse same process & ID.
	testsite.WithLiveTestSite(t, func(base string){
		secondID := fetchID(base)
		if secondID != firstID { t.Fatalf("expected reuse of instance; first=%s second=%s", firstID, secondID) }
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
