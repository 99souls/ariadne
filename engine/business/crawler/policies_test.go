package crawler

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrawlingPolicyEvaluation(t *testing.T) {
	tests := []struct {
		name     string
		policy   *CrawlingBusinessPolicy
		url      string
		expected bool
	}{
		{
			name: "allowed domain exact match",
			policy: &CrawlingBusinessPolicy{
				SiteRules: map[string]*SitePolicy{
					"example.com": {
						Allowed:  true,
						MaxDepth: 3,
					},
				},
			},
			url:      "https://example.com/page1",
			expected: true,
		},
		{
			name: "allowed domain subdomain match",
			policy: &CrawlingBusinessPolicy{
				SiteRules: map[string]*SitePolicy{
					"example.com": {
						Allowed:  true,
						MaxDepth: 3,
					},
				},
			},
			url:      "https://blog.example.com/post",
			expected: true,
		},
		{
			name: "disallowed domain",
			policy: &CrawlingBusinessPolicy{
				SiteRules: map[string]*SitePolicy{
					"allowed.com": {
						Allowed:  true,
						MaxDepth: 3,
					},
				},
			},
			url:      "https://blocked.com/page",
			expected: false,
		},
		{
			name: "explicit blocked site",
			policy: &CrawlingBusinessPolicy{
				SiteRules: map[string]*SitePolicy{
					"blocked.com": {
						Allowed: false,
					},
				},
			},
			url:      "https://blocked.com/page",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewCrawlingPolicyEvaluator(tt.policy)

			parsedURL, err := url.Parse(tt.url)
			require.NoError(t, err)

			result := evaluator.IsURLAllowed(parsedURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLinkFollowingPolicyEvaluation(t *testing.T) {
	tests := []struct {
		name     string
		policy   *LinkFollowingPolicy
		url      string
		depth    int
		expected bool
	}{
		{
			name: "within depth limit",
			policy: &LinkFollowingPolicy{
				MaxDepth:       3,
				FollowExternal: false,
			},
			url:      "https://example.com/page",
			depth:    2,
			expected: true,
		},
		{
			name: "exceeds depth limit",
			policy: &LinkFollowingPolicy{
				MaxDepth:       3,
				FollowExternal: false,
			},
			url:      "https://example.com/page",
			depth:    4,
			expected: false,
		},
		{
			name: "external link allowed",
			policy: &LinkFollowingPolicy{
				MaxDepth:       5,
				FollowExternal: true,
			},
			url:      "https://external.com/page",
			depth:    2,
			expected: true,
		},
		{
			name: "external link blocked",
			policy: &LinkFollowingPolicy{
				MaxDepth:       5,
				FollowExternal: false,
			},
			url:      "https://external.com/page",
			depth:    2,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := &CrawlingPolicyEvaluator{
				policy: &CrawlingBusinessPolicy{
					LinkRules: tt.policy,
					SiteRules: map[string]*SitePolicy{
						"example.com": {Allowed: true, MaxDepth: 5},
					},
				},
			}

			parsedURL, err := url.Parse(tt.url)
			require.NoError(t, err)

			result := evaluator.ShouldFollowLink(parsedURL, tt.depth)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContentSelectionPolicyEvaluation(t *testing.T) {
	tests := []struct {
		name     string
		policy   *ContentSelectionPolicy
		url      string
		expected []string
	}{
		{
			name: "site-specific selectors",
			policy: &ContentSelectionPolicy{
				DefaultSelectors: []string{"main", "article"},
				SiteSelectors: map[string][]string{
					"news.example.com": {".article-content", ".post-body"},
				},
			},
			url:      "https://news.example.com/article",
			expected: []string{".article-content", ".post-body"},
		},
		{
			name: "fallback to default selectors",
			policy: &ContentSelectionPolicy{
				DefaultSelectors: []string{"main", "article"},
				SiteSelectors: map[string][]string{
					"news.example.com": {".article-content", ".post-body"},
				},
			},
			url:      "https://blog.example.com/post",
			expected: []string{"main", "article"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := &CrawlingPolicyEvaluator{
				policy: &CrawlingBusinessPolicy{
					ContentRules: tt.policy,
				},
			}

			parsedURL, err := url.Parse(tt.url)
			require.NoError(t, err)

			result := evaluator.GetContentSelectors(parsedURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRateLimitingPolicyEvaluation(t *testing.T) {
	tests := []struct {
		name     string
		policy   *RateLimitingPolicy
		domain   string
		expected time.Duration
	}{
		{
			name: "site-specific delay",
			policy: &RateLimitingPolicy{
				DefaultDelay: 500 * time.Millisecond,
				SiteDelays: map[string]time.Duration{
					"slow.example.com": 2 * time.Second,
				},
			},
			domain:   "slow.example.com",
			expected: 2 * time.Second,
		},
		{
			name: "fallback to default delay",
			policy: &RateLimitingPolicy{
				DefaultDelay: 500 * time.Millisecond,
				SiteDelays: map[string]time.Duration{
					"slow.example.com": 2 * time.Second,
				},
			},
			domain:   "fast.example.com",
			expected: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := &CrawlingPolicyEvaluator{
				policy: &CrawlingBusinessPolicy{
					RateRules: tt.policy,
				},
			}

			result := evaluator.GetRequestDelay(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBusinessPoliciesIntegration(t *testing.T) {
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

	evaluator := NewCrawlingPolicyEvaluator(policy)
	testURL, _ := url.Parse("https://example.com/test-page")

	t.Run("integrated policy evaluation", func(t *testing.T) {
		// Test URL allowed
		assert.True(t, evaluator.IsURLAllowed(testURL))

		// Test link following
		assert.True(t, evaluator.ShouldFollowLink(testURL, 2))
		assert.False(t, evaluator.ShouldFollowLink(testURL, 4))

		// Test content selectors
		selectors := evaluator.GetContentSelectors(testURL)
		assert.Equal(t, []string{".content", ".main-article"}, selectors)

		// Test rate limiting
		delay := evaluator.GetRequestDelay("example.com")
		assert.Equal(t, 1*time.Second, delay)
	})
}
