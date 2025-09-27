//go:build ignore
// +build ignore

package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/99souls/ariadne/engine/models"
)

func main() {
	log.Println("🚀 Phase 1 Integration Test - Site Scraper")

	// Create test server to validate our crawler
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`<html><head><title>Home</title></head><body><h1>Home</h1><p>Home content</p><a href="/page1">Page 1</a><a href="/page2">Page 2</a></body></html>`))
		case "/page1":
			_, _ = w.Write([]byte(`<html><head><title>Page 1</title></head><body><h1>Page 1</h1><p>Page 1 content</p></body></html>`))
		case "/page2":
			_, _ = w.Write([]byte(`<html><head><title>Page 2</title></head><body><h1>Page 2</h1><p>Page 2 content</p></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Configure crawler for test
	config := models.DefaultConfig()
	config.StartURL = server.URL
	serverURL, _ := url.Parse(server.URL)
	config.AllowedDomains = []string{serverURL.Host}
	config.MaxPages = 3
	config.RequestDelay = 200 * time.Millisecond // Slow down for testing

	log.Printf("Testing with server: %s", server.URL)

	log.Println("Deprecated phase1 test binary ignored in build (crawler internalized).")
}
