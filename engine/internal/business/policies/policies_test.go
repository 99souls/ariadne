package policies

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusinessPolicies(t *testing.T) {
	policies := &BusinessPolicies{
		CrawlingPolicy: &CrawlingBusinessPolicy{
			SiteRules: map[string]*SitePolicy{
				"example.com": {
					AllowedDomains: []string{"example.com"},
					MaxDepth:       3,
				},
			},
			LinkRules: &LinkFollowingPolicy{
				FollowExternalLinks: false,
				MaxDepth:            5,
			},
			ContentRules: &ContentSelectionPolicy{
				DefaultSelectors: []string{"main", "article"},
			},
			RateRules: &RateLimitingPolicy{
				DefaultDelay: 1 * time.Second,
			},
		},
		ProcessingPolicy: &ProcessingBusinessPolicy{
			ContentExtractionRules: []string{"remove_scripts", "clean_html"},
			QualityThreshold:       0.7,
		},
		OutputPolicy: &OutputBusinessPolicy{
			DefaultFormat: "json",
			Compression:   true,
		},
		GlobalPolicy: &GlobalBusinessPolicy{
			MaxConcurrency: 10,
			Timeout:        30 * time.Second,
		},
	}

	t.Run("business_policies_structure", func(t *testing.T) {
		require.NotNil(t, policies.CrawlingPolicy)
		require.NotNil(t, policies.ProcessingPolicy)
		require.NotNil(t, policies.OutputPolicy)
		require.NotNil(t, policies.GlobalPolicy)

		assert.NotEmpty(t, policies.CrawlingPolicy.SiteRules)
		assert.Equal(t, 5, policies.CrawlingPolicy.LinkRules.MaxDepth)
		assert.Equal(t, "json", policies.OutputPolicy.DefaultFormat)
		assert.Equal(t, 10, policies.GlobalPolicy.MaxConcurrency)
	})

	t.Run("site_specific_rules", func(t *testing.T) {
		sitePolicy := policies.CrawlingPolicy.SiteRules["example.com"]
		require.NotNil(t, sitePolicy)

		assert.Contains(t, sitePolicy.AllowedDomains, "example.com")
		assert.Equal(t, 3, sitePolicy.MaxDepth)
	})
}

func TestPolicyManager(t *testing.T) {
	manager := NewPolicyManager()
	require.NotNil(t, manager)

	// Test policy configuration
	policies := &BusinessPolicies{
		CrawlingPolicy: &CrawlingBusinessPolicy{
			LinkRules: &LinkFollowingPolicy{
				FollowExternalLinks: true,
				MaxDepth:            10,
			},
		},
		GlobalPolicy: &GlobalBusinessPolicy{
			MaxConcurrency: 5,
			Timeout:        60 * time.Second,
		},
	}

	t.Run("configure_policies", func(t *testing.T) {
		err := manager.ConfigurePolicies(policies)
		assert.NoError(t, err)

		currentPolicies := manager.GetCurrentPolicies()
		assert.Equal(t, policies, currentPolicies)
	})

	t.Run("validate_policies", func(t *testing.T) {
		// Valid policies
		validPolicies := &BusinessPolicies{
			GlobalPolicy: &GlobalBusinessPolicy{
				MaxConcurrency: 5,
				Timeout:        30 * time.Second,
			},
		}

		err := manager.ValidatePolicies(validPolicies)
		assert.NoError(t, err)

		// Invalid policies (negative concurrency)
		invalidPolicies := &BusinessPolicies{
			GlobalPolicy: &GlobalBusinessPolicy{
				MaxConcurrency: -1,
				Timeout:        30 * time.Second,
			},
		}

		err = manager.ValidatePolicies(invalidPolicies)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maxConcurrency must be positive")
	})

	t.Run("get_policy_for_url", func(t *testing.T) {
		// Configure manager with site-specific policies
		policies := &BusinessPolicies{
			CrawlingPolicy: &CrawlingBusinessPolicy{
				SiteRules: map[string]*SitePolicy{
					"news.example.com": {
						AllowedDomains: []string{"news.example.com"},
						MaxDepth:       5,
					},
					"blog.example.com": {
						AllowedDomains: []string{"blog.example.com"},
						MaxDepth:       3,
					},
				},
			},
		}

		err := manager.ConfigurePolicies(policies)
		require.NoError(t, err)

		// Test site-specific policy retrieval
		newsPolicy := manager.GetPolicyForURL("https://news.example.com/article")
		assert.NotNil(t, newsPolicy)
		assert.Equal(t, 5, newsPolicy.MaxDepth)

		blogPolicy := manager.GetPolicyForURL("https://blog.example.com/post")
		assert.NotNil(t, blogPolicy)
		assert.Equal(t, 3, blogPolicy.MaxDepth)

		// Test fallback for unknown site
		unknownPolicy := manager.GetPolicyForURL("https://unknown.com/page")
		assert.Nil(t, unknownPolicy) // No default policy configured
	})
}

func TestDynamicRuleEngine(t *testing.T) {
	engine := NewDynamicRuleEngine()
	require.NotNil(t, engine)

	// Create test rules
	rule1 := &BusinessRule{
		ID:   "test-rule-1",
		Name: "News Site Rule",
		Condition: RuleCondition{
			URLPattern:  "*.news.com",
			ContentType: "article",
		},
		Action: RuleAction{
			SetMaxDepth:  5,
			SetDelay:     500 * time.Millisecond,
			SetSelectors: []string{".article-content", ".story"},
		},
		Priority: 10,
		Enabled:  true,
	}

	rule2 := &BusinessRule{
		ID:   "test-rule-2",
		Name: "Blog Site Rule",
		Condition: RuleCondition{
			URLPattern:  "*.blog.*",
			ContentType: "blog",
		},
		Action: RuleAction{
			SetMaxDepth:  3,
			SetDelay:     1 * time.Second,
			SetSelectors: []string{".post-content", ".blog-entry"},
		},
		Priority: 5,
		Enabled:  true,
	}

	t.Run("add_business_rules", func(t *testing.T) {
		err := engine.AddRule(rule1)
		assert.NoError(t, err)

		err = engine.AddRule(rule2)
		assert.NoError(t, err)

		rules := engine.GetAllRules()
		assert.Len(t, rules, 2)
	})

	t.Run("evaluate_rules_for_url", func(t *testing.T) {
		// Test news site matching
		context := &EvaluationContext{
			URL:         "https://tech.news.com/story/123",
			ContentType: "article",
			Domain:      "tech.news.com",
			Path:        "/story/123",
		}

		matchedRules := engine.EvaluateRules(context)
		assert.Len(t, matchedRules, 1)
		assert.Equal(t, "test-rule-1", matchedRules[0].ID)

		// Test blog site matching
		context = &EvaluationContext{
			URL:         "https://my.blog.org/post/456",
			ContentType: "blog",
			Domain:      "my.blog.org",
			Path:        "/post/456",
		}

		matchedRules = engine.EvaluateRules(context)
		assert.Len(t, matchedRules, 1)
		assert.Equal(t, "test-rule-2", matchedRules[0].ID)

		// Test no match
		context = &EvaluationContext{
			URL:         "https://example.com/page",
			ContentType: "generic",
			Domain:      "example.com",
			Path:        "/page",
		}

		matchedRules = engine.EvaluateRules(context)
		assert.Len(t, matchedRules, 0)
	})

	t.Run("rule_priority_ordering", func(t *testing.T) {
		// Add a third rule with higher priority that matches both patterns
		rule3 := &BusinessRule{
			ID:   "test-rule-3",
			Name: "High Priority Rule",
			Condition: RuleCondition{
				URLPattern: "*", // Matches all URLs
			},
			Action: RuleAction{
				SetMaxDepth: 1,
				SetDelay:    2 * time.Second,
			},
			Priority: 20, // Higher than rule1 (10) and rule2 (5)
			Enabled:  true,
		}

		err := engine.AddRule(rule3)
		assert.NoError(t, err)

		context := &EvaluationContext{
			URL:         "https://tech.news.com/story/123",
			ContentType: "article",
			Domain:      "tech.news.com",
			Path:        "/story/123",
		}

		matchedRules := engine.EvaluateRules(context)
		// Should match both rule1 and rule3, but rule3 should come first (higher priority)
		assert.Len(t, matchedRules, 2)
		assert.Equal(t, "test-rule-3", matchedRules[0].ID) // Higher priority first
		assert.Equal(t, "test-rule-1", matchedRules[1].ID)
	})

	t.Run("disable_enable_rules", func(t *testing.T) {
		// Disable rule1
		err := engine.DisableRule("test-rule-1")
		assert.NoError(t, err)

		context := &EvaluationContext{
			URL:         "https://tech.news.com/story/123",
			ContentType: "article",
			Domain:      "tech.news.com",
			Path:        "/story/123",
		}

		matchedRules := engine.EvaluateRules(context)
		// Should only match rule3 now (rule1 is disabled)
		assert.Len(t, matchedRules, 1)
		assert.Equal(t, "test-rule-3", matchedRules[0].ID)

		// Re-enable rule1
		err = engine.EnableRule("test-rule-1")
		assert.NoError(t, err)

		matchedRules = engine.EvaluateRules(context)
		// Should match both rules again
		assert.Len(t, matchedRules, 2)
	})

	t.Run("remove_rules", func(t *testing.T) {
		err := engine.RemoveRule("test-rule-3")
		assert.NoError(t, err)

		rules := engine.GetAllRules()
		assert.Len(t, rules, 2) // Only rule1 and rule2 remain

		// Verify rule3 no longer matches
		context := &EvaluationContext{
			URL:         "https://example.com/generic-page",
			ContentType: "generic",
		}

		matchedRules := engine.EvaluateRules(context)
		assert.Len(t, matchedRules, 0) // No matches without the wildcard rule
	})
}

func TestPolicyConfigurationLoader(t *testing.T) {
	loader := NewPolicyConfigurationLoader()
	require.NotNil(t, loader)

	t.Run("load_from_map", func(t *testing.T) {
		config := map[string]interface{}{
			"crawling": map[string]interface{}{
				"max_depth":             5,
				"follow_external_links": true,
			},
			"processing": map[string]interface{}{
				"quality_threshold": 0.8,
			},
			"output": map[string]interface{}{
				"default_format": "markdown",
				"compression":    false,
			},
			"global": map[string]interface{}{
				"max_concurrency": 15,
				"timeout":         "45s",
			},
		}

		policies, err := loader.LoadFromMap(config)
		assert.NoError(t, err)
		assert.NotNil(t, policies)

		assert.Equal(t, 5, policies.CrawlingPolicy.LinkRules.MaxDepth)
		assert.True(t, policies.CrawlingPolicy.LinkRules.FollowExternalLinks)
		assert.Equal(t, 0.8, policies.ProcessingPolicy.QualityThreshold)
		assert.Equal(t, "markdown", policies.OutputPolicy.DefaultFormat)
		assert.False(t, policies.OutputPolicy.Compression)
		assert.Equal(t, 15, policies.GlobalPolicy.MaxConcurrency)
		assert.Equal(t, 45*time.Second, policies.GlobalPolicy.Timeout)
	})

	t.Run("validation_errors", func(t *testing.T) {
		invalidConfig := map[string]interface{}{
			"global": map[string]interface{}{
				"max_concurrency": -5, // Invalid negative value
				"timeout":         "invalid_duration",
			},
		}

		policies, err := loader.LoadFromMap(invalidConfig)
		assert.Error(t, err)
		assert.Nil(t, policies)
		assert.Contains(t, err.Error(), "max_concurrency")
	})
}
