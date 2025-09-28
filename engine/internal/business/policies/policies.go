package policies

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	crawlerPolicies "github.com/99souls/ariadne/engine/internal/business/crawler"
	outputPolicies "github.com/99souls/ariadne/engine/internal/business/output"
	processorPolicies "github.com/99souls/ariadne/engine/internal/business/processor"
)

// BusinessPolicies represents the complete set of business policies for the engine
type BusinessPolicies struct {
	CrawlingPolicy   *CrawlingBusinessPolicy   `json:"crawling_policy"`
	ProcessingPolicy *ProcessingBusinessPolicy `json:"processing_policy"`
	OutputPolicy     *OutputBusinessPolicy     `json:"output_policy"`
	GlobalPolicy     *GlobalBusinessPolicy     `json:"global_policy"`
}

// CrawlingBusinessPolicy defines crawling-specific business rules
type CrawlingBusinessPolicy struct {
	SiteRules    map[string]*SitePolicy  `json:"site_rules"`
	LinkRules    *LinkFollowingPolicy    `json:"link_rules"`
	ContentRules *ContentSelectionPolicy `json:"content_rules"`
	RateRules    *RateLimitingPolicy     `json:"rate_rules"`
}

// ProcessingBusinessPolicy defines processing-specific business rules
type ProcessingBusinessPolicy struct {
	ContentExtractionRules []string `json:"content_extraction_rules"`
	QualityThreshold       float64  `json:"quality_threshold"`
	ProcessingSteps        []string `json:"processing_steps"`
}

// OutputBusinessPolicy defines output-specific business rules
type OutputBusinessPolicy struct {
	DefaultFormat string            `json:"default_format"`
	Compression   bool              `json:"compression"`
	RoutingRules  map[string]string `json:"routing_rules"`
	QualityGates  []string          `json:"quality_gates"`
}

// GlobalBusinessPolicy defines global business rules that apply across all components
type GlobalBusinessPolicy struct {
	MaxConcurrency int           `json:"max_concurrency"`
	Timeout        time.Duration `json:"timeout"`
	RetryPolicy    *RetryPolicy  `json:"retry_policy"`
	LoggingLevel   string        `json:"logging_level"`
}

// SitePolicy defines site-specific policies
type SitePolicy struct {
	AllowedDomains []string      `json:"allowed_domains"`
	MaxDepth       int           `json:"max_depth"`
	Delay          time.Duration `json:"delay"`
	Selectors      []string      `json:"selectors"`
}

// LinkFollowingPolicy defines rules for following links
type LinkFollowingPolicy struct {
	FollowExternalLinks bool `json:"follow_external_links"`
	MaxDepth            int  `json:"max_depth"`
}

// ContentSelectionPolicy defines rules for content selection
type ContentSelectionPolicy struct {
	DefaultSelectors []string            `json:"default_selectors"`
	SiteSelectors    map[string][]string `json:"site_selectors"`
}

// RateLimitingPolicy defines rate limiting rules
type RateLimitingPolicy struct {
	DefaultDelay time.Duration            `json:"default_delay"`
	SiteDelays   map[string]time.Duration `json:"site_delays"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// BusinessRule represents a dynamic business rule
type BusinessRule struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Condition RuleCondition `json:"condition"`
	Action    RuleAction    `json:"action"`
	Priority  int           `json:"priority"`
	Enabled   bool          `json:"enabled"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// RuleCondition defines when a rule should apply
type RuleCondition struct {
	URLPattern  string `json:"url_pattern"`
	ContentType string `json:"content_type"`
	Domain      string `json:"domain"`
	PathPattern string `json:"path_pattern"`
}

// RuleAction defines what action to take when rule matches
type RuleAction struct {
	SetMaxDepth    int           `json:"set_max_depth"`
	SetDelay       time.Duration `json:"set_delay"`
	SetSelectors   []string      `json:"set_selectors"`
	SetFormat      string        `json:"set_format"`
	EnableFeature  string        `json:"enable_feature"`
	DisableFeature string        `json:"disable_feature"`
}

// EvaluationContext provides context for rule evaluation
type EvaluationContext struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Domain      string `json:"domain"`
	Path        string `json:"path"`
}

// PolicyManager manages business policies with thread-safe operations
type PolicyManager struct {
	mutex           sync.RWMutex
	currentPolicies *BusinessPolicies
}

// DynamicRuleEngine manages dynamic business rules
type DynamicRuleEngine struct {
	mutex sync.RWMutex
	rules map[string]*BusinessRule
}

// PolicyConfigurationLoader loads policies from various sources
type PolicyConfigurationLoader struct {
	// Configuration loading utilities
}

// NewPolicyManager creates a new policy manager
func NewPolicyManager() *PolicyManager {
	return &PolicyManager{
		currentPolicies: &BusinessPolicies{},
	}
}

// NewDynamicRuleEngine creates a new dynamic rule engine
func NewDynamicRuleEngine() *DynamicRuleEngine {
	return &DynamicRuleEngine{
		rules: make(map[string]*BusinessRule),
	}
}

// NewPolicyConfigurationLoader creates a new policy configuration loader
func NewPolicyConfigurationLoader() *PolicyConfigurationLoader {
	return &PolicyConfigurationLoader{}
}

// PolicyManager methods

// ConfigurePolicies sets the current policies
func (pm *PolicyManager) ConfigurePolicies(policies *BusinessPolicies) error {
	if err := pm.ValidatePolicies(policies); err != nil {
		return fmt.Errorf("invalid policies: %w", err)
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.currentPolicies = policies
	return nil
}

// GetCurrentPolicies returns the current policies
func (pm *PolicyManager) GetCurrentPolicies() *BusinessPolicies {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return pm.currentPolicies
}

// GetPolicyForURL returns site-specific policy for a URL
func (pm *PolicyManager) GetPolicyForURL(targetURL string) *SitePolicy {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if pm.currentPolicies.CrawlingPolicy == nil || pm.currentPolicies.CrawlingPolicy.SiteRules == nil {
		return nil
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil
	}

	domain := parsedURL.Host

	// Check for exact domain match first
	if policy, exists := pm.currentPolicies.CrawlingPolicy.SiteRules[domain]; exists {
		return policy
	}

	// Check for wildcard matches
	for pattern, policy := range pm.currentPolicies.CrawlingPolicy.SiteRules {
		if pm.matchesDomainPattern(domain, pattern) {
			return policy
		}
	}

	return nil
}

// ValidatePolicies validates business policies
func (pm *PolicyManager) ValidatePolicies(policies *BusinessPolicies) error {
	if policies == nil {
		return fmt.Errorf("policies cannot be nil")
	}

	// Validate global policies
	if policies.GlobalPolicy != nil {
		if policies.GlobalPolicy.MaxConcurrency <= 0 {
			return fmt.Errorf("maxConcurrency must be positive, got %d", policies.GlobalPolicy.MaxConcurrency)
		}
		if policies.GlobalPolicy.Timeout <= 0 {
			return fmt.Errorf("timeout must be positive, got %v", policies.GlobalPolicy.Timeout)
		}
	}

	// Validate crawling policies
	if policies.CrawlingPolicy != nil {
		if policies.CrawlingPolicy.LinkRules != nil {
			if policies.CrawlingPolicy.LinkRules.MaxDepth < 0 {
				return fmt.Errorf("maxDepth must be non-negative, got %d", policies.CrawlingPolicy.LinkRules.MaxDepth)
			}
		}
	}

	// Validate processing policies
	if policies.ProcessingPolicy != nil {
		if policies.ProcessingPolicy.QualityThreshold < 0 || policies.ProcessingPolicy.QualityThreshold > 1 {
			return fmt.Errorf("qualityThreshold must be between 0 and 1, got %f", policies.ProcessingPolicy.QualityThreshold)
		}
	}

	return nil
}

// matchesDomainPattern checks if domain matches a pattern
func (pm *PolicyManager) matchesDomainPattern(domain, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Simple wildcard matching
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*.")
		return strings.HasSuffix(domain, suffix)
	}

	return domain == pattern
}

// DynamicRuleEngine methods

// AddRule adds a new business rule
func (dre *DynamicRuleEngine) AddRule(rule *BusinessRule) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}
	if rule.ID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	dre.mutex.Lock()
	defer dre.mutex.Unlock()

	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	dre.rules[rule.ID] = rule
	return nil
}

// RemoveRule removes a business rule
func (dre *DynamicRuleEngine) RemoveRule(ruleID string) error {
	dre.mutex.Lock()
	defer dre.mutex.Unlock()

	if _, exists := dre.rules[ruleID]; !exists {
		return fmt.Errorf("rule with ID %s not found", ruleID)
	}

	delete(dre.rules, ruleID)
	return nil
}

// EnableRule enables a business rule
func (dre *DynamicRuleEngine) EnableRule(ruleID string) error {
	dre.mutex.Lock()
	defer dre.mutex.Unlock()

	rule, exists := dre.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule with ID %s not found", ruleID)
	}

	rule.Enabled = true
	rule.UpdatedAt = time.Now()
	return nil
}

// DisableRule disables a business rule
func (dre *DynamicRuleEngine) DisableRule(ruleID string) error {
	dre.mutex.Lock()
	defer dre.mutex.Unlock()

	rule, exists := dre.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule with ID %s not found", ruleID)
	}

	rule.Enabled = false
	rule.UpdatedAt = time.Now()
	return nil
}

// GetAllRules returns all business rules
func (dre *DynamicRuleEngine) GetAllRules() []*BusinessRule {
	dre.mutex.RLock()
	defer dre.mutex.RUnlock()

	rules := make([]*BusinessRule, 0, len(dre.rules))
	for _, rule := range dre.rules {
		rules = append(rules, rule)
	}

	return rules
}

// EvaluateRules evaluates rules against the given context
func (dre *DynamicRuleEngine) EvaluateRules(ctx *EvaluationContext) []*BusinessRule {
	dre.mutex.RLock()
	defer dre.mutex.RUnlock()

	var matchedRules []*BusinessRule

	for _, rule := range dre.rules {
		if !rule.Enabled {
			continue
		}

		if dre.matchesCondition(rule.Condition, ctx) {
			matchedRules = append(matchedRules, rule)
		}
	}

	// Sort by priority (higher priority first)
	sort.Slice(matchedRules, func(i, j int) bool {
		return matchedRules[i].Priority > matchedRules[j].Priority
	})

	return matchedRules
}

// matchesCondition checks if a rule condition matches the context
func (dre *DynamicRuleEngine) matchesCondition(condition RuleCondition, ctx *EvaluationContext) bool {
	// URL pattern matching
	if condition.URLPattern != "" {
		if !dre.matchesURLPattern(condition.URLPattern, ctx.URL) {
			return false
		}
	}

	// Content type matching
	if condition.ContentType != "" && condition.ContentType != ctx.ContentType {
		return false
	}

	// Domain matching
	if condition.Domain != "" && condition.Domain != ctx.Domain {
		return false
	}

	// Path pattern matching
	if condition.PathPattern != "" {
		matched, err := filepath.Match(condition.PathPattern, ctx.Path)
		if err != nil || !matched {
			return false
		}
	}

	return true
}

// matchesURLPattern checks if URL matches a pattern with wildcard support
func (dre *DynamicRuleEngine) matchesURLPattern(pattern, targetURL string) bool {
	// Handle special case of "*" which matches everything
	if pattern == "*" {
		return true
	}

	// Parse the target URL to get components
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	// Simple wildcard pattern matching for domains
	if strings.Contains(pattern, "*") {
		// Handle patterns like "*.news.com" or "*.blog.*"
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*.")
			if strings.Contains(suffix, "*") {
				// Pattern like "*.blog.*" - check if domain contains the middle part
				parts := strings.Split(suffix, "*")
				if len(parts) >= 2 {
					return strings.Contains(parsedURL.Host, parts[0])
				}
			} else {
				// Pattern like "*.news.com" - check if domain ends with suffix
				return strings.HasSuffix(parsedURL.Host, suffix)
			}
		}
		// Pattern matching for full URL
		matched, err := filepath.Match(pattern, targetURL)
		if err != nil {
			return false
		}
		return matched
	}

	// Exact pattern matching
	return pattern == targetURL
}

// PolicyConfigurationLoader methods

// LoadFromMap loads policies from a configuration map
func (pcl *PolicyConfigurationLoader) LoadFromMap(config map[string]interface{}) (*BusinessPolicies, error) {
	policies := &BusinessPolicies{}

	// Load crawling policies
	if crawlingConfig, ok := config["crawling"].(map[string]interface{}); ok {
		crawlingPolicy := &CrawlingBusinessPolicy{}

		if linkConfig, exists := crawlingConfig["max_depth"].(int); exists {
			crawlingPolicy.LinkRules = &LinkFollowingPolicy{
				MaxDepth: linkConfig,
			}
		}

		if followExternal, exists := crawlingConfig["follow_external_links"].(bool); exists {
			if crawlingPolicy.LinkRules == nil {
				crawlingPolicy.LinkRules = &LinkFollowingPolicy{}
			}
			crawlingPolicy.LinkRules.FollowExternalLinks = followExternal
		}

		policies.CrawlingPolicy = crawlingPolicy
	}

	// Load processing policies
	if processingConfig, ok := config["processing"].(map[string]interface{}); ok {
		processingPolicy := &ProcessingBusinessPolicy{}

		if threshold, exists := processingConfig["quality_threshold"].(float64); exists {
			processingPolicy.QualityThreshold = threshold
		}

		policies.ProcessingPolicy = processingPolicy
	}

	// Load output policies
	if outputConfig, ok := config["output"].(map[string]interface{}); ok {
		outputPolicy := &OutputBusinessPolicy{}

		if format, exists := outputConfig["default_format"].(string); exists {
			outputPolicy.DefaultFormat = format
		}

		if compression, exists := outputConfig["compression"].(bool); exists {
			outputPolicy.Compression = compression
		}

		policies.OutputPolicy = outputPolicy
	}

	// Load global policies
	if globalConfig, ok := config["global"].(map[string]interface{}); ok {
		globalPolicy := &GlobalBusinessPolicy{}

		if concurrency, exists := globalConfig["max_concurrency"].(int); exists {
			if concurrency <= 0 {
				return nil, fmt.Errorf("max_concurrency must be positive, got %d", concurrency)
			}
			globalPolicy.MaxConcurrency = concurrency
		}

		if timeoutStr, exists := globalConfig["timeout"].(string); exists {
			timeout, err := time.ParseDuration(timeoutStr)
			if err != nil {
				return nil, fmt.Errorf("invalid timeout duration: %w", err)
			}
			globalPolicy.Timeout = timeout
		}

		policies.GlobalPolicy = globalPolicy
	}

	// Validate the loaded policies
	manager := NewPolicyManager()
	if err := manager.ValidatePolicies(policies); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return policies, nil
}

// Integration interfaces to connect with existing business logic

// ToCrawlerPolicies converts to crawler business policies
func (bp *BusinessPolicies) ToCrawlerPolicies() crawlerPolicies.CrawlingBusinessPolicy {
	if bp.CrawlingPolicy == nil {
		return crawlerPolicies.CrawlingBusinessPolicy{}
	}

	policy := crawlerPolicies.CrawlingBusinessPolicy{}

	// Convert site rules to crawler SitePolicy format
	if bp.CrawlingPolicy.SiteRules != nil {
		policy.SiteRules = make(map[string]*crawlerPolicies.SitePolicy)
		for domain, sitePolicy := range bp.CrawlingPolicy.SiteRules {
			policy.SiteRules[domain] = &crawlerPolicies.SitePolicy{
				Allowed:  len(sitePolicy.AllowedDomains) > 0,
				MaxDepth: sitePolicy.MaxDepth,
			}
		}
	}

	// Convert link following policy
	if bp.CrawlingPolicy.LinkRules != nil {
		policy.LinkRules = &crawlerPolicies.LinkFollowingPolicy{
			MaxDepth:       bp.CrawlingPolicy.LinkRules.MaxDepth,
			FollowExternal: bp.CrawlingPolicy.LinkRules.FollowExternalLinks,
		}
	}

	// Convert content selection policy
	if bp.CrawlingPolicy.ContentRules != nil {
		policy.ContentRules = &crawlerPolicies.ContentSelectionPolicy{
			DefaultSelectors: bp.CrawlingPolicy.ContentRules.DefaultSelectors,
			SiteSelectors:    bp.CrawlingPolicy.ContentRules.SiteSelectors,
		}
	}

	// Convert rate limiting policy
	if bp.CrawlingPolicy.RateRules != nil {
		policy.RateRules = &crawlerPolicies.RateLimitingPolicy{
			DefaultDelay: bp.CrawlingPolicy.RateRules.DefaultDelay,
			SiteDelays:   bp.CrawlingPolicy.RateRules.SiteDelays,
		}
	}

	return policy
}

// ToProcessorPolicies converts to processor business policies
func (bp *BusinessPolicies) ToProcessorPolicies() processorPolicies.ProcessingBusinessPolicy {
	if bp.ProcessingPolicy == nil {
		return processorPolicies.ProcessingBusinessPolicy{}
	}

	return processorPolicies.ProcessingBusinessPolicy{
		ContentPolicy: processorPolicies.ContentProcessingPolicy{
			ContentSelectors:   []string{"main", "article"}, // Default selectors
			MetadataExtraction: true,
			ImageExtraction:    true,
		},
		QualityPolicy: processorPolicies.ContentQualityPolicy{
			MinWordCount: int(bp.ProcessingPolicy.QualityThreshold * 100), // Convert threshold to word count
		},
	}
}

// ToOutputPolicies converts to output business policies
func (bp *BusinessPolicies) ToOutputPolicies() outputPolicies.OutputBusinessPolicy {
	if bp.OutputPolicy == nil {
		return outputPolicies.OutputBusinessPolicy{}
	}

	return outputPolicies.OutputBusinessPolicy{
		ProcessingPolicy: outputPolicies.OutputProcessingPolicy{
			OutputFormat:       bp.OutputPolicy.DefaultFormat,
			CompressionEnabled: bp.OutputPolicy.Compression,
		},
		QualityPolicy: outputPolicies.OutputQualityPolicy{
			RequireValidation: true,
			MaxOutputSize:     1048576, // 1MB default
		},
	}
}
