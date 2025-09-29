package crawler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	engcrawler "github.com/99souls/ariadne/engine/internal/crawler"
	"github.com/99souls/ariadne/engine/internal/testutil/testsite"
	"github.com/99souls/ariadne/engine/models"
)

// TestLiveSiteSearchIndexIgnored ensures that a future /api/search.json endpoint (non-HTML JSON index)
// is ignored / not treated as a discoverable HTML page result. Currently acts as a placeholder
// asserting that no page containing "/api/search.json" appears; once the endpoint is added to
// the test site this test will be extended to fetch it directly and validate it is excluded.
func TestLiveSiteSearchIndexIgnored(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	testsite.WithLiveTestSite(t, func(base string) {
		searchURL := base + "/api/search.json"
		// First, fetch directly to ensure endpoint works and returns JSON index.
		resp, err := http.Get(searchURL)
		if err != nil {
			t.Fatalf("fetch search index: %v", err)
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil {
				t.Fatalf("close body: %v", cerr)
			}
		}()
		if resp.StatusCode != 200 {
			t.Fatalf("unexpected status %d for search index", resp.StatusCode)
		}
		var payload struct {
			Version     int `json:"version"`
			Entries     []struct{ URL, Title string }
			GeneratedAt string `json:"generatedAt"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode search index: %v", err)
		}
		if payload.Version != 1 || len(payload.Entries) == 0 {
			t.Fatalf("unexpected payload: %#v", payload)
		}

		// Now perform a crawl and assert that /api/search.json never appears as a Page result URL.
		u, _ := url.Parse(base)
		cfg := models.DefaultConfig()
		cfg.AllowedDomains = []string{u.Host}
		cfg.StartURL = base + "/"
		cfg.MaxPages = 30
		cfg.Timeout = 5 * time.Second
		cfg.RequestDelay = 25 * time.Millisecond
		cfg.RespectRobots = true
		c := engcrawler.New(cfg)
		if err := c.Start(cfg.StartURL); err != nil {
			t.Fatalf("start crawler: %v", err)
		}
		sawSearch := false
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case r, ok := <-c.Results():
				if !ok {
					deadline = time.Now() // channel closed
					break
				}
				if r.URL == searchURL {
					sawSearch = true
					deadline = time.Now() // can early exit
				}
			case <-time.After(75 * time.Millisecond):
			}
		}
		c.Stop()
		if sawSearch {
			t.Fatalf("search index endpoint appeared as a page result (should be ignored): %s", searchURL)
		}
	})
}
