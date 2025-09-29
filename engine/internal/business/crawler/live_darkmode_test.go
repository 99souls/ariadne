package crawler_test

import (
	"net/url"
	"testing"
	"time"

	engcrawler "github.com/99souls/ariadne/engine/internal/crawler"
	"github.com/99souls/ariadne/engine/internal/testutil/testsite"
	"github.com/99souls/ariadne/engine/models"
)

// TestLiveSiteDarkModeDeDup ensures that visiting a dark-mode variant using a theme query parameter
// (e.g. ?theme=dark) does not cause the crawler to treat it as a distinct logical page when the
// underlying path is identical. This requires the crawler's normalization layer to ignore known
// cosmetic query keys.
func TestLiveSiteDarkModeDeDup(t *testing.T) {
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
		cfg.StartURL = base + "/?theme=dark" // seed with dark mode variant
		cfg.MaxPages = 20
		cfg.Timeout = 3 * time.Second
		cfg.RequestDelay = 40 * time.Millisecond
		cfg.RespectRobots = true
		crawler := engcrawler.New(cfg)
		if err := crawler.Start(cfg.StartURL); err != nil {
			t.Fatalf("start crawler: %v", err)
		}

		canonical := base + "/"
		darkVariant := base + "/?theme=dark"
		seenCanonical := false
		seenVariant := false
		// Collect results until both forms would be seen (which would indicate failure) or timeout.
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) && (!seenCanonical || !seenVariant) {
			select {
			case r := <-crawler.Results():
				if r == nil {
					break
				}
				if r.Success && r.URL == canonical {
					seenCanonical = true
				}
				if r.Success && r.URL == darkVariant {
					seenVariant = true
				}
			case <-time.After(100 * time.Millisecond):
			}
		}
		crawler.Stop()
		// Either: (1) canonical only (preferred normalization) OR (2) both canonical+variant treated separately (fail) OR (3) only variant (fail).
		if seenCanonical && seenVariant {
			t.Fatalf("expected dark mode query to normalize; saw both canonical=%v and variant=%v", seenCanonical, seenVariant)
		}
		if !seenCanonical && !seenVariant {
			t.Fatalf("neither canonical nor variant discovered")
		}
		if !seenCanonical && seenVariant {
			t.Fatalf("only variant discovered; expected normalization to canonical URL")
		}
	})
}
