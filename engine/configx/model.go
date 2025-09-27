package configx

import "time"

// EngineConfigSpec is the canonical hierarchical configuration payload.
// Layers will merge and overlay partial specs to produce a final runtime config.
type EngineConfigSpec struct {
	Global     *GlobalConfigSection     `json:"global,omitempty"`
	Crawling   *CrawlingConfigSection   `json:"crawling,omitempty"`
	Processing *ProcessingConfigSection `json:"processing,omitempty"`
	Output     *OutputConfigSection     `json:"output,omitempty"`
	Policies   *PoliciesConfigSection   `json:"policies,omitempty"`
	Rollout    *RolloutSpec             `json:"rollout,omitempty"`
}

// GlobalConfigSection captures cross-cutting limits and behaviors applied to the entire engine.
type GlobalConfigSection struct {
	MaxConcurrency int              `json:"max_concurrency,omitempty"`
	Timeout        time.Duration    `json:"timeout,omitempty"`
	RetryPolicy    *RetryPolicySpec `json:"retry_policy,omitempty"`
	LoggingLevel   string           `json:"logging_level,omitempty"`
}

// RetryPolicySpec defines retry semantics for operations governed by the config system.
type RetryPolicySpec struct {
	MaxRetries    int           `json:"max_retries,omitempty"`
	InitialDelay  time.Duration `json:"initial_delay,omitempty"`
	BackoffFactor float64       `json:"backoff_factor,omitempty"`
}

// CrawlingConfigSection drives site fetching behaviors.
type CrawlingConfigSection struct {
	SiteRules map[string]*SiteCrawlerRule `json:"site_rules,omitempty"`
	LinkRules *LinkRuleConfig             `json:"link_rules,omitempty"`
	RateRules *RateLimitConfig            `json:"rate_rules,omitempty"`
}

// SiteCrawlerRule tailors crawling parameters for a specific domain or site group.
type SiteCrawlerRule struct {
	AllowedDomains []string      `json:"allowed_domains,omitempty"`
	MaxDepth       int           `json:"max_depth,omitempty"`
	Delay          time.Duration `json:"delay,omitempty"`
	Selectors      []string      `json:"selectors,omitempty"`
}

// LinkRuleConfig governs which links are traversed during crawling.
type LinkRuleConfig struct {
	FollowExternal bool `json:"follow_external,omitempty"`
	MaxDepth       int  `json:"max_depth,omitempty"`
}

// RateLimitConfig defines rate limiting characteristics.
type RateLimitConfig struct {
	DefaultDelay time.Duration            `json:"default_delay,omitempty"`
	SiteDelays   map[string]time.Duration `json:"site_delays,omitempty"`
}

// ProcessingConfigSection contains extraction and processing directives.
type ProcessingConfigSection struct {
	ExtractionRules    []string          `json:"extraction_rules,omitempty"`
	QualityThreshold   float64           `json:"quality_threshold,omitempty"`
	ProcessingSteps    []string          `json:"processing_steps,omitempty"`
	ConditionalActions map[string]string `json:"conditional_actions,omitempty"`
}

// OutputConfigSection configures output formatting and routing.
type OutputConfigSection struct {
	DefaultFormat string            `json:"default_format,omitempty"`
	Compression   bool              `json:"compression,omitempty"`
	RoutingRules  map[string]string `json:"routing_rules,omitempty"`
	QualityGates  []string          `json:"quality_gates,omitempty"`
}

// PoliciesConfigSection captures dynamic business rules tied to runtime configuration.
type PoliciesConfigSection struct {
	BusinessRules []*PolicyRuleSpec `json:"business_rules,omitempty"`
	EnabledFlags  map[string]bool   `json:"enabled_flags,omitempty"`
}

// PolicyRuleSpec represents a single dynamic rule.
type PolicyRuleSpec struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Priority  int       `json:"priority,omitempty"`
	Condition string    `json:"condition,omitempty"`
	Action    string    `json:"action,omitempty"`
	Enabled   bool      `json:"enabled,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// RolloutSpec declares how a configuration change is rolled out.
type RolloutSpec struct {
	Mode              string   `json:"mode"` // full|percentage|cohort
	Percentage        int      `json:"percentage,omitempty"`
	CohortDomains     []string `json:"cohort_domains,omitempty"`
	CohortDomainGlobs []string `json:"cohort_domain_globs,omitempty"`
}

// VersionedConfig records a committed configuration along with metadata.
type VersionedConfig struct {
	Version     int64             `json:"version"`
	Spec        *EngineConfigSpec `json:"spec"`
	Hash        string            `json:"hash"`
	AppliedAt   time.Time         `json:"applied_at"`
	Actor       string            `json:"actor"`
	Parent      int64             `json:"parent"`
	DiffSummary string            `json:"diff_summary,omitempty"`
}

// ApplyOptions control how a configuration change is processed.
type ApplyOptions struct {
	Actor        string `json:"actor"`
	DryRun       bool   `json:"dry_run"`
	Force        bool   `json:"force"`
	RolloutStage bool   `json:"rollout_stage"`
}
