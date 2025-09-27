package crawler

import (
	"net/url"
	"strings"
	"sync"
	"time"
)

// ContentExtractionRules defines rules for extracting content from a site
type ContentExtractionRules struct {
	Selectors    []string
	ExcludeRules []string
}

// SiteRateLimitRules defines site-specific rate limiting rules
type SiteRateLimitRules struct {
	RequestDelay  time.Duration
	MaxConcurrent int
}

// SiteSpecificPolicy contains all site-specific rules and policies
type SiteSpecificPolicy struct {
	Domain    string
	Crawling  SitePolicy
	Content   ContentExtractionRules
	RateLimit SiteRateLimitRules
}

// SitePolicyManager manages site-specific policies
type SitePolicyManager struct {
	policies map[string]*SiteSpecificPolicy
	mu       sync.RWMutex
}

// NewSitePolicyManager creates a new site policy manager
func NewSitePolicyManager() *SitePolicyManager {
	return &SitePolicyManager{
		policies: make(map[string]*SiteSpecificPolicy),
	}
}

// AddSitePolicy adds or replaces a site policy
func (spm *SitePolicyManager) AddSitePolicy(domain string, policy *SiteSpecificPolicy) error {
	spm.mu.Lock()
	defer spm.mu.Unlock()
	
	spm.policies[domain] = policy
	return nil
}

// UpdateSitePolicy updates an existing site policy
func (spm *SitePolicyManager) UpdateSitePolicy(domain string, policy *SiteSpecificPolicy) error {
	spm.mu.Lock()
	defer spm.mu.Unlock()
	
	spm.policies[domain] = policy
	return nil
}

// RemoveSitePolicy removes a site policy
func (spm *SitePolicyManager) RemoveSitePolicy(domain string) {
	spm.mu.Lock()
	defer spm.mu.Unlock()
	
	delete(spm.policies, domain)
}

// GetSitePolicy retrieves a site policy by domain
func (spm *SitePolicyManager) GetSitePolicy(domain string) *SiteSpecificPolicy {
	spm.mu.RLock()
	defer spm.mu.RUnlock()
	
	return spm.policies[domain]
}

// GetAllSites returns all managed site domains
func (spm *SitePolicyManager) GetAllSites() []string {
	spm.mu.RLock()
	defer spm.mu.RUnlock()
	
	domains := make([]string, 0, len(spm.policies))
	for domain := range spm.policies {
		domains = append(domains, domain)
	}
	return domains
}

// SiteRuleEvaluator evaluates site-specific rules
type SiteRuleEvaluator struct {
	manager *SitePolicyManager
}

// NewSiteRuleEvaluator creates a new site rule evaluator
func NewSiteRuleEvaluator(manager *SitePolicyManager) *SiteRuleEvaluator {
	return &SiteRuleEvaluator{
		manager: manager,
	}
}

// GetApplicablePolicy finds the most specific policy for a URL
func (sre *SiteRuleEvaluator) GetApplicablePolicy(u *url.URL) *SiteSpecificPolicy {
	domain := u.Host
	
	// First try exact match
	if policy := sre.manager.GetSitePolicy(domain); policy != nil {
		return policy
	}
	
	// Try parent domains (for subdomain matching)
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		parentDomain := strings.Join(parts[i:], ".")
		if policy := sre.manager.GetSitePolicy(parentDomain); policy != nil {
			return policy
		}
	}
	
	return nil
}

// GetContentExtractionRules returns content extraction rules for a URL
func (sre *SiteRuleEvaluator) GetContentExtractionRules(u *url.URL) ContentExtractionRules {
	policy := sre.GetApplicablePolicy(u)
	if policy != nil {
		return policy.Content
	}
	
	// Return empty rules as default
	return ContentExtractionRules{}
}

// GetRateLimitRules returns rate limiting rules for a URL
func (sre *SiteRuleEvaluator) GetRateLimitRules(u *url.URL) SiteRateLimitRules {
	policy := sre.GetApplicablePolicy(u)
	if policy != nil {
		return policy.RateLimit
	}
	
	// Return empty rules as default
	return SiteRateLimitRules{}
}