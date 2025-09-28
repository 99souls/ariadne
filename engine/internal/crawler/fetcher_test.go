package crawler

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestFetcherInterface ensures the Fetcher interface contract is properly defined
func TestFetcherInterface(t *testing.T) {
	t.Run("should define basic fetch contract", func(t *testing.T) {
		// This test validates the interface definition exists and has correct method signatures
		var fetcher Fetcher
		_ = fetcher // Interface should compile; zero value of an interface is nil. No runtime assertion needed.
	})
}

// TestFetchResult validates the fetch result structure
func TestFetchResult(t *testing.T) {
	t.Run("should contain required fields", func(t *testing.T) {
		testURL, _ := url.Parse("https://example.com/test")

		result := &FetchResult{
			URL:      testURL,
			Content:  []byte("<html><body>Test</body></html>"),
			Headers:  map[string]string{"Content-Type": "text/html"},
			Status:   200,
			Links:    []*url.URL{testURL},
			Metadata: map[string]interface{}{"title": "Test Page"},
		}

		if result.URL.String() != "https://example.com/test" {
			t.Errorf("Expected URL to be preserved, got %v", result.URL)
		}

		if result.Status != 200 {
			t.Errorf("Expected status 200, got %d", result.Status)
		}

		if string(result.Content) != "<html><body>Test</body></html>" {
			t.Errorf("Expected content to be preserved")
		}

		if result.Headers["Content-Type"] != "text/html" {
			t.Errorf("Expected headers to be preserved")
		}

		if len(result.Links) != 1 {
			t.Errorf("Expected 1 link, got %d", len(result.Links))
		}

		if result.Metadata["title"] != "Test Page" {
			t.Errorf("Expected metadata to be preserved")
		}
	})
}

// TestFetchPolicy validates the fetch policy configuration
func TestFetchPolicy(t *testing.T) {
	t.Run("should define comprehensive fetch policies", func(t *testing.T) {
		policy := FetchPolicy{
			UserAgent:       "Ariadne/1.0",
			RequestDelay:    time.Second,
			Timeout:         30 * time.Second,
			MaxRetries:      3,
			RespectRobots:   true,
			FollowRedirects: true,
			AllowedDomains:  []string{"example.com", "test.com"},
			MaxDepth:        10,
		}

		if policy.UserAgent != "Ariadne/1.0" {
			t.Errorf("Expected UserAgent to be preserved")
		}

		if policy.RequestDelay != time.Second {
			t.Errorf("Expected RequestDelay to be preserved")
		}

		if policy.Timeout != 30*time.Second {
			t.Errorf("Expected Timeout to be preserved")
		}

		if policy.MaxRetries != 3 {
			t.Errorf("Expected MaxRetries to be preserved")
		}

		if !policy.RespectRobots {
			t.Errorf("Expected RespectRobots to be true")
		}

		if !policy.FollowRedirects {
			t.Errorf("Expected FollowRedirects to be true")
		}

		if len(policy.AllowedDomains) != 2 {
			t.Errorf("Expected 2 allowed domains, got %d", len(policy.AllowedDomains))
		}

		if policy.MaxDepth != 10 {
			t.Errorf("Expected MaxDepth to be preserved")
		}
	})
}

// TestCollyFetcher tests the concrete Colly-based implementation
func TestCollyFetcher(t *testing.T) {
	t.Run("should implement Fetcher interface", func(t *testing.T) {
		policy := FetchPolicy{
			UserAgent:    "Test Agent",
			RequestDelay: 100 * time.Millisecond,
			Timeout:      5 * time.Second,
			MaxRetries:   1,
		}

		fetcher, err := NewCollyFetcher(policy)
		if err != nil {
			t.Fatalf("Failed to create CollyFetcher: %v", err)
		}

		// Verify interface compliance
		var _ Fetcher = fetcher

		if fetcher == nil {
			t.Error("Expected non-nil fetcher")
		}
	})

	t.Run("should validate policy configuration", func(t *testing.T) {
		// Test invalid policy
		invalidPolicy := FetchPolicy{
			Timeout:    0,  // Invalid timeout
			MaxRetries: -1, // Invalid retry count
		}

		_, err := NewCollyFetcher(invalidPolicy)
		if err == nil {
			t.Error("Expected error for invalid policy")
		}

		if !strings.Contains(err.Error(), "timeout") {
			t.Error("Expected timeout validation error")
		}
	})

	t.Run("should fetch content successfully", func(t *testing.T) {
		// This will be a mock test initially, then integration test
		policy := FetchPolicy{
			UserAgent:    "Test Agent",
			RequestDelay: 10 * time.Millisecond,
			Timeout:      5 * time.Second,
			MaxRetries:   1,
		}

		fetcher, err := NewCollyFetcher(policy)
		if err != nil {
			t.Fatalf("Failed to create fetcher: %v", err)
		}

		ctx := context.Background()

		// For now, we'll test with a mock URL that should fail gracefully
		// TODO: Add proper mock server for integration testing
		_, err = fetcher.Fetch(ctx, "http://invalid-test-url-12345.example")

		// We expect this to fail for now, but it should be a controlled failure
		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
	})

	t.Run("should discover links from content", func(t *testing.T) {
		policy := FetchPolicy{
			UserAgent: "Test Agent",
			Timeout:   5 * time.Second,
		}

		fetcher, err := NewCollyFetcher(policy)
		if err != nil {
			t.Fatalf("Failed to create fetcher: %v", err)
		}

		ctx := context.Background()
		baseURL, _ := url.Parse("https://example.com/page")

		htmlContent := []byte(`
			<html>
				<body>
					<a href="/test1">Link 1</a>
					<a href="https://example.com/test2">Link 2</a>
					<a href="mailto:test@example.com">Email</a>
					<a href="javascript:void(0)">JS Link</a>
				</body>
			</html>
		`)

		links, err := fetcher.Discover(ctx, htmlContent, baseURL)
		if err != nil {
			t.Fatalf("Failed to discover links: %v", err)
		}

		// Should find valid HTTP links and resolve relative URLs
		if len(links) < 2 {
			t.Errorf("Expected at least 2 links, got %d", len(links))
		}

		// Verify relative URL resolution
		found := false
		for _, link := range links {
			if link.String() == "https://example.com/test1" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected relative URL to be resolved to absolute URL")
		}
	})

	t.Run("should provide stats", func(t *testing.T) {
		policy := FetchPolicy{
			UserAgent: "Test Agent",
			Timeout:   5 * time.Second,
		}

		fetcher, err := NewCollyFetcher(policy)
		if err != nil {
			t.Fatalf("Failed to create fetcher: %v", err)
		}

		stats := fetcher.Stats()

		// Stats should be initialized
		if stats.RequestsCompleted < 0 {
			t.Error("Expected non-negative requests completed")
		}

		if stats.RequestsFailed < 0 {
			t.Error("Expected non-negative requests failed")
		}

		if stats.LinksDiscovered < 0 {
			t.Error("Expected non-negative links discovered")
		}
	})

	t.Run("should allow policy updates", func(t *testing.T) {
		initialPolicy := FetchPolicy{
			UserAgent: "Initial Agent",
			Timeout:   5 * time.Second,
		}

		fetcher, err := NewCollyFetcher(initialPolicy)
		if err != nil {
			t.Fatalf("Failed to create fetcher: %v", err)
		}

		newPolicy := FetchPolicy{
			UserAgent:    "Updated Agent",
			RequestDelay: 2 * time.Second,
			Timeout:      10 * time.Second,
		}

		err = fetcher.Configure(newPolicy)
		if err != nil {
			t.Fatalf("Failed to update policy: %v", err)
		}

		// Verify policy was updated (this will require getter or internal validation)
		// For now, just ensure no error occurred
	})
}

// TestFetcherStats validates the stats structure
func TestFetcherStats(t *testing.T) {
	t.Run("should track comprehensive statistics", func(t *testing.T) {
		stats := FetcherStats{
			RequestsCompleted: 10,
			RequestsFailed:    2,
			LinksDiscovered:   50,
			BytesDownloaded:   1024,
			AverageLatency:    200 * time.Millisecond,
		}

		if stats.RequestsCompleted != 10 {
			t.Errorf("Expected 10 completed requests, got %d", stats.RequestsCompleted)
		}

		if stats.RequestsFailed != 2 {
			t.Errorf("Expected 2 failed requests, got %d", stats.RequestsFailed)
		}

		if stats.LinksDiscovered != 50 {
			t.Errorf("Expected 50 discovered links, got %d", stats.LinksDiscovered)
		}

		if stats.BytesDownloaded != 1024 {
			t.Errorf("Expected 1024 bytes downloaded, got %d", stats.BytesDownloaded)
		}

		if stats.AverageLatency != 200*time.Millisecond {
			t.Errorf("Expected 200ms average latency, got %v", stats.AverageLatency)
		}
	})
}
