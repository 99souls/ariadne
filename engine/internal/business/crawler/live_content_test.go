package crawler_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	engcrawler "github.com/99souls/ariadne/engine/internal/crawler"
	"github.com/99souls/ariadne/engine/internal/testutil/testsite"
	"github.com/99souls/ariadne/engine/models"
)

// TestLiveSiteAdvancedContent verifies the crawler retrieves the static advanced content
// fixture (admonitions, code fences, footnotes) and that key semantic markers are present
// in the captured Page.Content. This guards against regressions in HTML capture and ensures
// the test site provides rich fixture coverage beyond trivial text.
func TestLiveSiteAdvancedContent(t *testing.T) {
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
		cfg.MaxPages = 60
		cfg.Timeout = 4 * time.Second
		cfg.RequestDelay = 40 * time.Millisecond
		cfg.RespectRobots = true
		crawler := engcrawler.New(cfg)
		if err := crawler.Start(cfg.StartURL); err != nil {
			t.Fatalf("start crawler: %v", err)
		}

		targetURL := base + "/static/admonitions.html"
		var content string
		deadline := time.Now().Add(6 * time.Second)
		for time.Now().Before(deadline) && content == "" {
			select {
			case r := <-crawler.Results():
				if r == nil {
					break
				}
				if r.Success && r.URL == targetURL && r.Page != nil {
					content = r.Page.Content
				}
			case <-time.After(100 * time.Millisecond):
			}
		}
		crawler.Stop()
		if content == "" {
			t.Fatalf("did not capture content for %s", targetURL)
		}

		// Assertions on semantic markers.
		// 1. Multiple admonitions present.
		if count := strings.Count(content, "class=\"admonition"); count < 2 {
			t.Fatalf("expected >=2 admonition blocks; found %d", count)
		}
		mustContainSubstring(t, content, "class=\"admonition note\"")
		mustContainSubstring(t, content, "class=\"admonition warning\"")
		// 2. Code fences with language classes.
		mustContainSubstring(t, content, "class=\"language-go\"")
		mustContainSubstring(t, content, "class=\"language-ts\"")
		// 3. Footnote reference and backlink anchors.
		mustContainSubstring(t, content, "id=\"fnref1\"")
		mustContainSubstring(t, content, "id=\"fn1\"")
	})
}

func mustContainSubstring(t *testing.T, body, sub string) {
	t.Helper()
	if !strings.Contains(body, sub) {
		t.Fatalf("expected substring %q not found", sub)
	}
}
