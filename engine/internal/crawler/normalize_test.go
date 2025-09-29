package crawler

import (
	"net/url"
	"testing"

	"github.com/99souls/ariadne/engine/models"
)

// TestNormalizeURLCosmeticParams ensures cosmetic / tracking query parameters are stripped
// while meaningful parameters are preserved. It also validates fragment removal.
func TestNormalizeURLCosmeticParams(t *testing.T) {
	cfg := defaultTestConfig()
	c := New(&cfg) // minimal valid config

	cases := []struct {
		in   string
		out  string
		name string
	}{
		{in: "http://example.com/?theme=dark", out: "http://example.com/", name: "theme only"},
		{in: "http://example.com/?theme=dark&utm_source=foo", out: "http://example.com/", name: "theme + utm both removed"},
		{in: "http://example.com/?page=2&theme=dark", out: "http://example.com/?page=2", name: "preserve non-cosmetic param"},
		{in: "http://example.com/?utm_campaign=abc&page=3", out: "http://example.com/?page=3", name: "remove utm keep page"},
		{in: "http://example.com/?utm_source=foo&utm_medium=bar", out: "http://example.com/", name: "all tracking removed"},
		{in: "http://example.com/?q=go+lang", out: "http://example.com/?q=go+lang", name: "unrelated query preserved"},
		{in: "http://example.com/docs/getting-started?theme=light", out: "http://example.com/docs/getting-started", name: "path with theme"},
		{in: "http://example.com/about#section", out: "http://example.com/about", name: "fragment removed"},
	}

	for _, tc := range cases {
		parsed, err := url.Parse(tc.in)
		if err != nil {
			t.Fatalf("parse %s: %v", tc.in, err)
		}
		got := c.normalizeURL(parsed)
		if got != tc.out {
			// Show difference once per case
			gotURL, _ := url.Parse(got)
			wantURL, _ := url.Parse(tc.out)
			_ = gotURL
			_ = wantURL
			// (If later we need ordering insensitive compare, we could map queries.)
			t.Errorf("%s: normalizeURL(%q) = %q; want %q", tc.name, tc.in, got, tc.out)
		}
	}
}

// defaultTestConfig returns a minimal valid scraper config for constructing a crawler.
// We keep values small to avoid side effects (no real network calls in this unit test).
func defaultTestConfig() (cfg models.ScraperConfig) {
	cfg.StartURL = "http://example.com/"
	cfg.AllowedDomains = []string{"example.com"}
	cfg.MaxDepth = 2
	cfg.MaxPages = 10
	cfg.RequestDelay = 0
	return
}
