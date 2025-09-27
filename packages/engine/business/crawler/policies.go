package crawler

import (
	"net/url"
	"strings"
	"time"
)

// SitePolicy defines site-specific crawling rules
type SitePolicy struct {
	Allowed  bool
	MaxDepth int
}

// LinkFollowingPolicy defines rules for following links
type LinkFollowingPolicy struct {
	MaxDepth       int
	FollowExternal bool
}

// ContentSelectionPolicy defines rules for content extraction
type ContentSelectionPolicy struct {
	DefaultSelectors []string
	SiteSelectors    map[string][]string
}

// RateLimitingPolicy defines rate limiting rules
type RateLimitingPolicy struct {
	DefaultDelay time.Duration
	SiteDelays   map[string]time.Duration
}

// CrawlingBusinessPolicy consolidates all crawling business rules
type CrawlingBusinessPolicy struct {
	SiteRules    map[string]*SitePolicy
	LinkRules    *LinkFollowingPolicy
	ContentRules *ContentSelectionPolicy
	RateRules    *RateLimitingPolicy
}

// CrawlingPolicyEvaluator evaluates crawling policies against URLs and contexts
type CrawlingPolicyEvaluator struct {
	policy *CrawlingBusinessPolicy
}

// NewCrawlingPolicyEvaluator creates a new policy evaluator
func NewCrawlingPolicyEvaluator(policy *CrawlingBusinessPolicy) *CrawlingPolicyEvaluator {
	return &CrawlingPolicyEvaluator{
		policy: policy,
	}
}

// IsURLAllowed determines if a URL is allowed to be crawled
func (e *CrawlingPolicyEvaluator) IsURLAllowed(u *url.URL) bool {
	if e.policy.SiteRules == nil {
		return false
	}

	domain := u.Host

	// Check for exact domain match
	if sitePolicy, exists := e.policy.SiteRules[domain]; exists {
		return sitePolicy.Allowed
	}

	// Check for parent domain match (subdomain handling)
	for ruleDomain, sitePolicy := range e.policy.SiteRules {
		if strings.HasSuffix(domain, "."+ruleDomain) {
			return sitePolicy.Allowed
		}
	}

	// Default deny if no matching rule found
	return false
}

// ShouldFollowLink determines if a link should be followed based on depth and external rules
func (e *CrawlingPolicyEvaluator) ShouldFollowLink(u *url.URL, currentDepth int) bool {
	if e.policy.LinkRules == nil {
		return false
	}

	// Check depth limit
	if currentDepth >= e.policy.LinkRules.MaxDepth {
		return false
	}

	// For external links, check if external following is enabled
	// We need a way to determine if this is external - for now, assume external if not in site rules
	isExternal := true
	if e.policy.SiteRules != nil {
		domain := u.Host
		if _, exists := e.policy.SiteRules[domain]; exists {
			isExternal = false
		} else {
			// Check for parent domain match
			for ruleDomain := range e.policy.SiteRules {
				if strings.HasSuffix(domain, "."+ruleDomain) {
					isExternal = false
					break
				}
			}
		}
	}

	// If it's external and external following is disabled, don't follow
	if isExternal && !e.policy.LinkRules.FollowExternal {
		return false
	}

	return true
}

// GetContentSelectors returns the appropriate content selectors for a URL
func (e *CrawlingPolicyEvaluator) GetContentSelectors(u *url.URL) []string {
	if e.policy.ContentRules == nil {
		return []string{}
	}

	domain := u.Host

	// Check for site-specific selectors
	if selectors, exists := e.policy.ContentRules.SiteSelectors[domain]; exists {
		return selectors
	}

	// Return default selectors
	return e.policy.ContentRules.DefaultSelectors
}

// GetRequestDelay returns the appropriate request delay for a domain
func (e *CrawlingPolicyEvaluator) GetRequestDelay(domain string) time.Duration {
	if e.policy.RateRules == nil {
		return 0
	}

	// Check for site-specific delay
	if delay, exists := e.policy.RateRules.SiteDelays[domain]; exists {
		return delay
	}

	// Return default delay
	return e.policy.RateRules.DefaultDelay
}
