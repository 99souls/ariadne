package crawler

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrawlingDecisionMaker(t *testing.T) {
	policy := &CrawlingBusinessPolicy{
		SiteRules: map[string]*SitePolicy{
			"example.com": {
				Allowed:  true,
				MaxDepth: 3,
			},
		},
		LinkRules: &LinkFollowingPolicy{
			MaxDepth:       3,
			FollowExternal: false,
		},
		ContentRules: &ContentSelectionPolicy{
			DefaultSelectors: []string{"main", "article"},
			SiteSelectors: map[string][]string{
				"example.com": {".content", ".main-article"},
			},
		},
		RateRules: &RateLimitingPolicy{
			DefaultDelay: 500 * time.Millisecond,
			SiteDelays: map[string]time.Duration{
				"example.com": 1 * time.Second,
			},
		},
	}

	decisionMaker := NewCrawlingDecisionMaker(policy)
	
	t.Run("should crawl decision", func(t *testing.T) {
		testURL, _ := url.Parse("https://example.com/page")
		ctx := context.Background()
		
		decision, err := decisionMaker.ShouldCrawl(ctx, testURL, 2)
		require.NoError(t, err)
		
		assert.True(t, decision.ShouldCrawl)
		assert.Equal(t, []string{".content", ".main-article"}, decision.ContentSelectors)
		assert.Equal(t, 1*time.Second, decision.RequestDelay)
		assert.Equal(t, "example.com", decision.Domain)
		assert.Equal(t, 2, decision.CurrentDepth)
	})

	t.Run("should not crawl blocked domain", func(t *testing.T) {
		testURL, _ := url.Parse("https://blocked.com/page")
		ctx := context.Background()
		
		decision, err := decisionMaker.ShouldCrawl(ctx, testURL, 1)
		require.NoError(t, err)
		
		assert.False(t, decision.ShouldCrawl)
		assert.Equal(t, "domain not allowed", decision.Reason)
	})

	t.Run("should not crawl excessive depth", func(t *testing.T) {
		testURL, _ := url.Parse("https://example.com/deep/page")
		ctx := context.Background()
		
		decision, err := decisionMaker.ShouldCrawl(ctx, testURL, 5)
		require.NoError(t, err)
		
		assert.False(t, decision.ShouldCrawl)
		assert.Equal(t, "exceeds maximum depth", decision.Reason)
	})
}

func TestLinkDecisionMaking(t *testing.T) {
	policy := &CrawlingBusinessPolicy{
		SiteRules: map[string]*SitePolicy{
			"example.com": {
				Allowed:  true,
				MaxDepth: 3,
			},
		},
		LinkRules: &LinkFollowingPolicy{
			MaxDepth:       3,
			FollowExternal: false,
		},
	}

	decisionMaker := NewCrawlingDecisionMaker(policy)
	
	t.Run("should follow internal link", func(t *testing.T) {
		baseURL, _ := url.Parse("https://example.com/base")
		linkURL, _ := url.Parse("https://example.com/linked-page")
		ctx := context.Background()
		
		decision, err := decisionMaker.ShouldFollowLink(ctx, baseURL, linkURL, 1)
		require.NoError(t, err)
		
		assert.True(t, decision.ShouldFollow)
		assert.Equal(t, 2, decision.NextDepth)
	})

	t.Run("should not follow external link when disabled", func(t *testing.T) {
		baseURL, _ := url.Parse("https://example.com/base")
		linkURL, _ := url.Parse("https://external.com/page")
		ctx := context.Background()
		
		decision, err := decisionMaker.ShouldFollowLink(ctx, baseURL, linkURL, 1)
		require.NoError(t, err)
		
		assert.False(t, decision.ShouldFollow)
		assert.Equal(t, "external links disabled", decision.Reason)
	})
}

func TestBatchDecisionMaking(t *testing.T) {
	policy := &CrawlingBusinessPolicy{
		SiteRules: map[string]*SitePolicy{
			"example.com": {
				Allowed:  true,
				MaxDepth: 3,
			},
		},
		LinkRules: &LinkFollowingPolicy{
			MaxDepth:       3,
			FollowExternal: false,
		},
	}

	decisionMaker := NewCrawlingDecisionMaker(policy)
	
	t.Run("batch crawl decisions", func(t *testing.T) {
		urls := []string{
			"https://example.com/page1",
			"https://example.com/page2", 
			"https://blocked.com/page",
		}
		
		ctx := context.Background()
		decisions, err := decisionMaker.BatchShouldCrawl(ctx, urls, 1)
		require.NoError(t, err)
		
		assert.Len(t, decisions, 3)
		assert.True(t, decisions[0].ShouldCrawl)
		assert.True(t, decisions[1].ShouldCrawl)
		assert.False(t, decisions[2].ShouldCrawl)
	})
}

func TestCrawlingContext(t *testing.T) {
	policy := &CrawlingBusinessPolicy{
		RateRules: &RateLimitingPolicy{
			DefaultDelay: 500 * time.Millisecond,
		},
	}

	decisionMaker := NewCrawlingDecisionMaker(policy)
	
	t.Run("creates crawling context", func(t *testing.T) {
		testURL, _ := url.Parse("https://example.com/page")
		
		ctx := decisionMaker.CreateCrawlingContext(context.Background(), testURL)
		
		// Verify context contains expected values
		assert.NotNil(t, ctx)
		
		// Check if domain is stored in context
		domain, ok := ctx.Value(CrawlingContextKey("domain")).(string)
		require.True(t, ok)
		assert.Equal(t, "example.com", domain)
	})
}