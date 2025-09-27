package crawler

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSitePolicyManager(t *testing.T) {
	t.Run("create site policy manager", func(t *testing.T) {
		manager := NewSitePolicyManager()
		assert.NotNil(t, manager)
		assert.Empty(t, manager.GetAllSites())
	})
	
	t.Run("add and retrieve site policies", func(t *testing.T) {
		manager := NewSitePolicyManager()
		
		policy := &SiteSpecificPolicy{
			Domain: "example.com",
			Crawling: SitePolicy{
				Allowed:  true,
				MaxDepth: 5,
			},
			Content: ContentExtractionRules{
				Selectors:    []string{".main-content", ".article"},
				ExcludeRules: []string{".advertisement", ".sidebar"},
			},
			RateLimit: SiteRateLimitRules{
				RequestDelay: 2 * time.Second,
				MaxConcurrent: 2,
			},
		}
		
		err := manager.AddSitePolicy("example.com", policy)
		require.NoError(t, err)
		
		retrieved := manager.GetSitePolicy("example.com")
		require.NotNil(t, retrieved)
		assert.Equal(t, "example.com", retrieved.Domain)
		assert.True(t, retrieved.Crawling.Allowed)
		assert.Equal(t, 5, retrieved.Crawling.MaxDepth)
	})
	
	t.Run("update existing site policy", func(t *testing.T) {
		manager := NewSitePolicyManager()
		
		// Add initial policy
		policy1 := &SiteSpecificPolicy{
			Domain: "example.com",
			Crawling: SitePolicy{
				Allowed:  true,
				MaxDepth: 3,
			},
		}
		err := manager.AddSitePolicy("example.com", policy1)
		require.NoError(t, err)
		
		// Update policy
		policy2 := &SiteSpecificPolicy{
			Domain: "example.com",
			Crawling: SitePolicy{
				Allowed:  true,
				MaxDepth: 10,
			},
		}
		err = manager.UpdateSitePolicy("example.com", policy2)
		require.NoError(t, err)
		
		// Verify update
		retrieved := manager.GetSitePolicy("example.com")
		assert.Equal(t, 10, retrieved.Crawling.MaxDepth)
	})
	
	t.Run("remove site policy", func(t *testing.T) {
		manager := NewSitePolicyManager()
		
		policy := &SiteSpecificPolicy{
			Domain: "example.com",
			Crawling: SitePolicy{Allowed: true},
		}
		err := manager.AddSitePolicy("example.com", policy)
		require.NoError(t, err)
		
		// Verify it exists
		assert.NotNil(t, manager.GetSitePolicy("example.com"))
		
		// Remove it
		manager.RemoveSitePolicy("example.com")
		
		// Verify it's gone
		assert.Nil(t, manager.GetSitePolicy("example.com"))
	})
}

func TestSiteRuleEvaluator(t *testing.T) {
	manager := NewSitePolicyManager()
	
	// Add policies for different sites
	newsPolicy := &SiteSpecificPolicy{
		Domain: "news.example.com",
		Content: ContentExtractionRules{
			Selectors:    []string{".article-body", ".post-content"},
			ExcludeRules: []string{".ads", ".comments"},
		},
		RateLimit: SiteRateLimitRules{
			RequestDelay:  1 * time.Second,
			MaxConcurrent: 1,
		},
	}
	
	blogPolicy := &SiteSpecificPolicy{
		Domain: "blog.example.com",
		Content: ContentExtractionRules{
			Selectors:    []string{".blog-post", ".entry-content"},
			ExcludeRules: []string{".sidebar", ".footer"},
		},
		RateLimit: SiteRateLimitRules{
			RequestDelay:  500 * time.Millisecond,
			MaxConcurrent: 3,
		},
	}
	
	err := manager.AddSitePolicy("news.example.com", newsPolicy)
	require.NoError(t, err)
	err = manager.AddSitePolicy("blog.example.com", blogPolicy)
	require.NoError(t, err)
	
	evaluator := NewSiteRuleEvaluator(manager)
	
	t.Run("evaluate content extraction rules", func(t *testing.T) {
		newsURL, _ := url.Parse("https://news.example.com/article/123")
		rules := evaluator.GetContentExtractionRules(newsURL)
		
		assert.Equal(t, []string{".article-body", ".post-content"}, rules.Selectors)
		assert.Equal(t, []string{".ads", ".comments"}, rules.ExcludeRules)
	})
	
	t.Run("evaluate rate limit rules", func(t *testing.T) {
		blogURL, _ := url.Parse("https://blog.example.com/post/456")
		rules := evaluator.GetRateLimitRules(blogURL)
		
		assert.Equal(t, 500*time.Millisecond, rules.RequestDelay)
		assert.Equal(t, 3, rules.MaxConcurrent)
	})
	
	t.Run("fallback for unknown site", func(t *testing.T) {
		unknownURL, _ := url.Parse("https://unknown.com/page")
		rules := evaluator.GetContentExtractionRules(unknownURL)
		
		// Should return default/empty rules
		assert.Empty(t, rules.Selectors)
		assert.Empty(t, rules.ExcludeRules)
	})
}

func TestSitePatternMatching(t *testing.T) {
	manager := NewSitePolicyManager()
	
	// Add a policy for a parent domain
	policy := &SiteSpecificPolicy{
		Domain: "example.com",
		Crawling: SitePolicy{
			Allowed:  true,
			MaxDepth: 5,
		},
	}
	err := manager.AddSitePolicy("example.com", policy)
	require.NoError(t, err)
	
	evaluator := NewSiteRuleEvaluator(manager)
	
	tests := []struct {
		name        string
		url         string
		shouldMatch bool
	}{
		{
			name:        "exact domain match",
			url:         "https://example.com/page",
			shouldMatch: true,
		},
		{
			name:        "subdomain match",
			url:         "https://blog.example.com/post",
			shouldMatch: true,
		},
		{
			name:        "different domain",
			url:         "https://different.com/page",
			shouldMatch: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testURL, _ := url.Parse(tt.url)
			policy := evaluator.GetApplicablePolicy(testURL)
			
			if tt.shouldMatch {
				assert.NotNil(t, policy)
				assert.Equal(t, "example.com", policy.Domain)
			} else {
				assert.Nil(t, policy)
			}
		})
	}
}

func TestSitePolicyMerging(t *testing.T) {
	manager := NewSitePolicyManager()
	
	// Add base policy
	basePolicy := &SiteSpecificPolicy{
		Domain: "example.com",
		Crawling: SitePolicy{
			Allowed:  true,
			MaxDepth: 3,
		},
		Content: ContentExtractionRules{
			Selectors: []string{".content"},
		},
		RateLimit: SiteRateLimitRules{
			RequestDelay: 1 * time.Second,
		},
	}
	
	// Add more specific policy
	specificPolicy := &SiteSpecificPolicy{
		Domain: "api.example.com",
		Crawling: SitePolicy{
			Allowed:  true,
			MaxDepth: 10, // Override
		},
		RateLimit: SiteRateLimitRules{
			RequestDelay:  2 * time.Second, // Override
			MaxConcurrent: 5,               // Add new
		},
		// Don't override content rules
	}
	
	err := manager.AddSitePolicy("example.com", basePolicy)
	require.NoError(t, err)
	err = manager.AddSitePolicy("api.example.com", specificPolicy)
	require.NoError(t, err)
	
	evaluator := NewSiteRuleEvaluator(manager)
	
	t.Run("specific policy takes precedence", func(t *testing.T) {
		apiURL, _ := url.Parse("https://api.example.com/data")
		policy := evaluator.GetApplicablePolicy(apiURL)
		
		require.NotNil(t, policy)
		assert.Equal(t, "api.example.com", policy.Domain)
		assert.Equal(t, 10, policy.Crawling.MaxDepth) // Override
		assert.Equal(t, 2*time.Second, policy.RateLimit.RequestDelay) // Override
		assert.Equal(t, 5, policy.RateLimit.MaxConcurrent) // New field
	})
}