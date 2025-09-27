package configx

import "time"

// Resolver performs layered configuration resolution.
// It merges EngineConfigSpec fragments provided per layer into a single effective spec.
// Merge semantics:
//   * Precedence: later layers in LayerPrecedenceOrder() override earlier ones.
//   * Section pointers: nil means "no contribution"; non-nil overlays field-wise.
//   * Scalars: higher layer non-zero or zero values overwrite lower (explicit override model).
//   * Slices: if higher layer slice is non-empty it replaces lower slice entirely.
//   * Maps: merged by key; higher layer entries overwrite conflicting keys.
//   * Nested structs inside maps (e.g., SiteRules) are deep-copied to avoid mutation.
// NOTE: Because scalar fields are not pointer types we cannot distinguish intentional zero
// from omission; future iteration may introduce pointer-based optional scalars if needed.
// The resolver never mutates the input specs.
//
// Public surface kept small to allow evolution before CLI / external exposure.

type Resolver struct{}

// NewResolver constructs a new Resolver.
func NewResolver() *Resolver { return &Resolver{} }

// Resolve merges the provided specs (indexed by layer constant) into a final EngineConfigSpec.
// Missing entries are skipped. Returns a deep copy.
func (r *Resolver) Resolve(layerSpecs map[int]*EngineConfigSpec) *EngineConfigSpec {
	final := &EngineConfigSpec{}
	for _, layer := range LayerPrecedenceOrder() {
		spec := layerSpecs[layer]
		if spec == nil {
			continue
		}
		mergeSpecs(final, spec)
	}
	return final
}

// mergeSpecs overlays src onto dst (in-place) according to merge semantics.
func mergeSpecs(dst, src *EngineConfigSpec) {
	if src.Global != nil {
		if dst.Global == nil { dst.Global = &GlobalConfigSection{} }
		mergeGlobal(dst.Global, src.Global)
	}
	if src.Crawling != nil {
		if dst.Crawling == nil { dst.Crawling = &CrawlingConfigSection{} }
		mergeCrawling(dst.Crawling, src.Crawling)
	}
	if src.Processing != nil {
		if dst.Processing == nil { dst.Processing = &ProcessingConfigSection{} }
		mergeProcessing(dst.Processing, src.Processing)
	}
	if src.Output != nil {
		if dst.Output == nil { dst.Output = &OutputConfigSection{} }
		mergeOutput(dst.Output, src.Output)
	}
	if src.Policies != nil {
		if dst.Policies == nil { dst.Policies = &PoliciesConfigSection{} }
		mergePolicies(dst.Policies, src.Policies)
	}
	if src.Rollout != nil {
		// Rollout is replaced as a unit (higher layer fully controls strategy)
		dst.Rollout = cloneRollout(src.Rollout)
	}
}

func mergeGlobal(dst, src *GlobalConfigSection) {
	if src.MaxConcurrency != 0 || dst.MaxConcurrency == 0 { dst.MaxConcurrency = src.MaxConcurrency }
	if src.Timeout != 0 || dst.Timeout == 0 { dst.Timeout = src.Timeout }
	if src.LoggingLevel != "" { dst.LoggingLevel = src.LoggingLevel }
	if src.RetryPolicy != nil {
		if dst.RetryPolicy == nil { dst.RetryPolicy = &RetryPolicySpec{} }
		dst.RetryPolicy.MaxRetries = src.RetryPolicy.MaxRetries
		dst.RetryPolicy.InitialDelay = src.RetryPolicy.InitialDelay
		dst.RetryPolicy.BackoffFactor = src.RetryPolicy.BackoffFactor
	}
}

func mergeCrawling(dst, src *CrawlingConfigSection) {
	if src.SiteRules != nil {
		if dst.SiteRules == nil { dst.SiteRules = make(map[string]*SiteCrawlerRule, len(src.SiteRules)) }
		for k, v := range src.SiteRules {
			if v == nil { continue }
			dst.SiteRules[k] = cloneSiteRule(v)
		}
	}
	if src.LinkRules != nil {
		if dst.LinkRules == nil { dst.LinkRules = &LinkRuleConfig{} }
		dst.LinkRules.FollowExternal = src.LinkRules.FollowExternal
		dst.LinkRules.MaxDepth = src.LinkRules.MaxDepth
	}
	if src.RateRules != nil {
		if dst.RateRules == nil { dst.RateRules = &RateLimitConfig{} }
		if src.RateRules.DefaultDelay != 0 || dst.RateRules.DefaultDelay == 0 { dst.RateRules.DefaultDelay = src.RateRules.DefaultDelay }
		if src.RateRules.SiteDelays != nil {
			if dst.RateRules.SiteDelays == nil { dst.RateRules.SiteDelays = make(map[string]time.Duration, len(src.RateRules.SiteDelays)) }
			for k, v := range src.RateRules.SiteDelays {
				// direct copy (time.Duration)
				dst.RateRules.SiteDelays[k] = v
			}
		}
	}
}

func mergeProcessing(dst, src *ProcessingConfigSection) {
	if len(src.ExtractionRules) > 0 { dst.ExtractionRules = cloneStringSlice(src.ExtractionRules) }
	if src.QualityThreshold != 0 || dst.QualityThreshold == 0 { dst.QualityThreshold = src.QualityThreshold }
	if len(src.ProcessingSteps) > 0 { dst.ProcessingSteps = cloneStringSlice(src.ProcessingSteps) }
	if src.ConditionalActions != nil {
		if dst.ConditionalActions == nil { dst.ConditionalActions = make(map[string]string, len(src.ConditionalActions)) }
		for k, v := range src.ConditionalActions { dst.ConditionalActions[k] = v }
	}
}

func mergeOutput(dst, src *OutputConfigSection) {
	if src.DefaultFormat != "" { dst.DefaultFormat = src.DefaultFormat }
	// bool override: always take higher layer explicit value
	dst.Compression = src.Compression
	if src.RoutingRules != nil {
		if dst.RoutingRules == nil { dst.RoutingRules = make(map[string]string, len(src.RoutingRules)) }
		for k, v := range src.RoutingRules { dst.RoutingRules[k] = v }
	}
	if len(src.QualityGates) > 0 { dst.QualityGates = cloneStringSlice(src.QualityGates) }
}

func mergePolicies(dst, src *PoliciesConfigSection) {
	if src.BusinessRules != nil {
		// Replace entire slice (higher layer authoritative for ordering/Priority context)
		cloned := make([]*PolicyRuleSpec, 0, len(src.BusinessRules))
		for _, r := range src.BusinessRules {
			if r == nil { continue }
			cr := *r
			cloned = append(cloned, &cr)
		}
		dst.BusinessRules = cloned
	}
	if src.EnabledFlags != nil {
		if dst.EnabledFlags == nil { dst.EnabledFlags = make(map[string]bool, len(src.EnabledFlags)) }
		for k, v := range src.EnabledFlags { dst.EnabledFlags[k] = v }
	}
}

func cloneSiteRule(r *SiteCrawlerRule) *SiteCrawlerRule {
	if r == nil { return nil }
	c := *r
	if len(r.AllowedDomains) > 0 { c.AllowedDomains = cloneStringSlice(r.AllowedDomains) }
	if len(r.Selectors) > 0 { c.Selectors = cloneStringSlice(r.Selectors) }
	return &c
}

func cloneRollout(r *RolloutSpec) *RolloutSpec {
	if r == nil { return nil }
	c := *r
	if len(r.CohortDomains) > 0 { c.CohortDomains = cloneStringSlice(r.CohortDomains) }
	if len(r.CohortDomainGlobs) > 0 { c.CohortDomainGlobs = cloneStringSlice(r.CohortDomainGlobs) }
	return &c
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 { return nil }
	out := make([]string, len(in))
	copy(out, in)
	return out
}
