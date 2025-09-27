package output

import (
	"strings"
	"time"
)

// OutputProcessingPolicy defines how output should be processed
type OutputProcessingPolicy struct {
	OutputFormat         string        `json:"output_format"`         // Format for output (json, markdown, html)
	CompressionEnabled   bool          `json:"compression_enabled"`   // Whether to compress output
	CompressionLevel     int           `json:"compression_level"`     // Compression level (0-9)
	BufferingEnabled     bool          `json:"buffering_enabled"`     // Whether to buffer output
	BufferSize          int           `json:"buffer_size"`           // Size of output buffer
	FlushInterval       time.Duration `json:"flush_interval"`        // Interval for flushing buffer
	RetryEnabled        bool          `json:"retry_enabled"`         // Whether to retry failed outputs
	MaxRetries          int           `json:"max_retries"`           // Maximum retry attempts
	RetryDelay          time.Duration `json:"retry_delay"`           // Delay between retries
	TransformationRules []string      `json:"transformation_rules"`  // Rules for output transformation
	ValidationEnabled   bool          `json:"validation_enabled"`    // Whether to validate output
}

// OutputQualityPolicy defines quality requirements for output
type OutputQualityPolicy struct {
	RequireValidation     bool     `json:"require_validation"`      // Whether validation is required
	MaxOutputSize         int      `json:"max_output_size"`         // Maximum output size in bytes
	MinContentLength      int      `json:"min_content_length"`      // Minimum content length
	AllowedFormats        []string `json:"allowed_formats"`         // Allowed output formats
	RequireMetadata       bool     `json:"require_metadata"`        // Whether metadata is required
	ValidateSchema        bool     `json:"validate_schema"`         // Whether to validate against schema
	ContentIntegrityCheck bool     `json:"content_integrity_check"` // Whether to check content integrity
}

// OutputBusinessPolicy combines processing and quality policies
type OutputBusinessPolicy struct {
	ProcessingPolicy OutputProcessingPolicy `json:"processing_policy"`
	QualityPolicy    OutputQualityPolicy    `json:"quality_policy"`
}

// OutputDecision represents a decision about output processing
type OutputDecision struct {
	URL            string `json:"url"`
	ShouldProcess  bool   `json:"should_process"`
	ProcessingType string `json:"processing_type"`
	Reason         string `json:"reason"`
}

// OutputConfigurationDecision represents output configuration for a URL
type OutputConfigurationDecision struct {
	URL                 string   `json:"url"`
	OutputFormat        string   `json:"output_format"`
	CompressionEnabled  bool     `json:"compression_enabled"`
	BufferingEnabled    bool     `json:"buffering_enabled"`
	RetryEnabled        bool     `json:"retry_enabled"`
	TransformationRules []string `json:"transformation_rules"`
}

// OutputContext holds context information for output decisions
type OutputContext struct {
	URL       string                `json:"url"`
	Policy    OutputBusinessPolicy  `json:"policy"`
	CreatedAt time.Time            `json:"created_at"`
	Status    string               `json:"status"`
}

// RoutingRule defines a single routing rule
type RoutingRule struct {
	Pattern  string `json:"pattern"`   // Pattern to match (supports wildcards)
	SinkName string `json:"sink_name"` // Name of the sink to route to
}

// OutputRoutingRules defines routing rules for output
type OutputRoutingRules struct {
	DefaultSink  string        `json:"default_sink"`  // Default sink when no rules match
	Rules        []RoutingRule `json:"rules"`         // List of routing rules
	FallbackSink string        `json:"fallback_sink"` // Fallback sink on errors
}

// RoutingDestination represents a routing destination decision
type RoutingDestination struct {
	SinkName string `json:"sink_name"`
	Reason   string `json:"reason"`
}

// RoutingDecision represents a decision about output routing
type RoutingDecision struct {
	URL                   string   `json:"url"`
	ShouldUseMultipleSinks bool     `json:"should_use_multiple_sinks"`
	PrimarySink           string   `json:"primary_sink"`
	SecondarySinks        []string `json:"secondary_sinks"`
}

// OutputProcessingPolicyEvaluator evaluates output processing policies
type OutputProcessingPolicyEvaluator struct {
	// No fields needed for stateless evaluation
}

// OutputQualityPolicyEvaluator evaluates output quality policies
type OutputQualityPolicyEvaluator struct {
	// No fields needed for stateless evaluation
}

// OutputDecisionMaker makes business decisions about output processing
type OutputDecisionMaker struct {
	processingEvaluator *OutputProcessingPolicyEvaluator
	qualityEvaluator    *OutputQualityPolicyEvaluator
}

// OutputRoutingDecisionMaker makes routing decisions for output
type OutputRoutingDecisionMaker struct {
	// No fields needed for stateless routing decisions
}

// NewOutputProcessingPolicyEvaluator creates a new output processing policy evaluator
func NewOutputProcessingPolicyEvaluator() *OutputProcessingPolicyEvaluator {
	return &OutputProcessingPolicyEvaluator{}
}

// NewOutputQualityPolicyEvaluator creates a new output quality policy evaluator
func NewOutputQualityPolicyEvaluator() *OutputQualityPolicyEvaluator {
	return &OutputQualityPolicyEvaluator{}
}

// NewOutputDecisionMaker creates a new output decision maker
func NewOutputDecisionMaker() *OutputDecisionMaker {
	return &OutputDecisionMaker{
		processingEvaluator: NewOutputProcessingPolicyEvaluator(),
		qualityEvaluator:    NewOutputQualityPolicyEvaluator(),
	}
}

// NewOutputRoutingDecisionMaker creates a new output routing decision maker
func NewOutputRoutingDecisionMaker() *OutputRoutingDecisionMaker {
	return &OutputRoutingDecisionMaker{}
}

// OutputProcessingPolicyEvaluator methods

// ShouldProcessOutput determines if output should be processed based on URL and policy
func (e *OutputProcessingPolicyEvaluator) ShouldProcessOutput(url string, policy OutputProcessingPolicy) bool {
	// Default behavior is to process all output unless specifically disabled
	return true
}

// GetOutputFormat returns the appropriate output format for a URL
func (e *OutputProcessingPolicyEvaluator) GetOutputFormat(url string, policy OutputProcessingPolicy) string {
	return policy.OutputFormat
}

// ShouldEnableCompression determines if compression should be enabled
func (e *OutputProcessingPolicyEvaluator) ShouldEnableCompression(url string, policy OutputProcessingPolicy) bool {
	return policy.CompressionEnabled
}

// ShouldEnableBuffering determines if buffering should be enabled
func (e *OutputProcessingPolicyEvaluator) ShouldEnableBuffering(url string, policy OutputProcessingPolicy) bool {
	return policy.BufferingEnabled
}

// ShouldEnableRetry determines if retry should be enabled
func (e *OutputProcessingPolicyEvaluator) ShouldEnableRetry(url string, policy OutputProcessingPolicy) bool {
	return policy.RetryEnabled
}

// GetTransformationRules returns the transformation rules for output processing
func (e *OutputProcessingPolicyEvaluator) GetTransformationRules(url string, policy OutputProcessingPolicy) []string {
	return policy.TransformationRules
}

// OutputQualityPolicyEvaluator methods

// MeetsSizeRequirements checks if output size meets the requirements
func (e *OutputQualityPolicyEvaluator) MeetsSizeRequirements(size int, policy OutputQualityPolicy) bool {
	return size <= policy.MaxOutputSize
}

// MeetsContentLengthRequirements checks if content length meets the requirements
func (e *OutputQualityPolicyEvaluator) MeetsContentLengthRequirements(length int, policy OutputQualityPolicy) bool {
	return length >= policy.MinContentLength
}

// IsFormatAllowed checks if the output format is allowed
func (e *OutputQualityPolicyEvaluator) IsFormatAllowed(format string, policy OutputQualityPolicy) bool {
	for _, allowedFormat := range policy.AllowedFormats {
		if allowedFormat == format {
			return true
		}
	}
	return false
}

// MeetsMetadataRequirements checks if metadata requirements are met
func (e *OutputQualityPolicyEvaluator) MeetsMetadataRequirements(hasMetadata bool, policy OutputQualityPolicy) bool {
	if !policy.RequireMetadata {
		return true
	}
	return hasMetadata
}

// OutputDecisionMaker methods

// ShouldProcessOutput makes a decision about whether to process specific output
func (d *OutputDecisionMaker) ShouldProcessOutput(url string, policy OutputBusinessPolicy) OutputDecision {
	shouldProcess := d.processingEvaluator.ShouldProcessOutput(url, policy.ProcessingPolicy)
	
	return OutputDecision{
		URL:            url,
		ShouldProcess:  shouldProcess,
		ProcessingType: "output_processing",
		Reason:         "policy_evaluation",
	}
}

// GetOutputConfiguration returns the output configuration for a given URL and policy
func (d *OutputDecisionMaker) GetOutputConfiguration(url string, policy OutputBusinessPolicy) OutputConfigurationDecision {
	return OutputConfigurationDecision{
		URL:                 url,
		OutputFormat:        d.processingEvaluator.GetOutputFormat(url, policy.ProcessingPolicy),
		CompressionEnabled:  d.processingEvaluator.ShouldEnableCompression(url, policy.ProcessingPolicy),
		BufferingEnabled:    d.processingEvaluator.ShouldEnableBuffering(url, policy.ProcessingPolicy),
		RetryEnabled:        d.processingEvaluator.ShouldEnableRetry(url, policy.ProcessingPolicy),
		TransformationRules: d.processingEvaluator.GetTransformationRules(url, policy.ProcessingPolicy),
	}
}

// CreateOutputContext creates an output context for a URL
func (d *OutputDecisionMaker) CreateOutputContext(url string, policy OutputBusinessPolicy) OutputContext {
	return OutputContext{
		URL:       url,
		Policy:    policy,
		CreatedAt: time.Now(),
		Status:    "pending",
	}
}

// BatchShouldProcess makes output decisions for multiple URLs
func (d *OutputDecisionMaker) BatchShouldProcess(urls []string, policy OutputBusinessPolicy) []OutputDecision {
	decisions := make([]OutputDecision, len(urls))
	
	for i, url := range urls {
		decisions[i] = d.ShouldProcessOutput(url, policy)
	}
	
	return decisions
}

// OutputRoutingDecisionMaker methods

// GetSinkDestination determines the appropriate sink destination for a given input
func (r *OutputRoutingDecisionMaker) GetSinkDestination(input string, rules OutputRoutingRules) RoutingDestination {
	// Check routing rules for pattern matches
	for _, rule := range rules.Rules {
		if r.matchesPattern(input, rule.Pattern) {
			return RoutingDestination{
				SinkName: rule.SinkName,
				Reason:   "pattern_match",
			}
		}
	}
	
	// Use default sink if no rules match
	return RoutingDestination{
		SinkName: rules.DefaultSink,
		Reason:   "default_route",
	}
}

// ShouldRouteToMultipleSinks determines if output should be routed to multiple sinks
func (r *OutputRoutingDecisionMaker) ShouldRouteToMultipleSinks(url string, rules OutputRoutingRules) RoutingDecision {
	// Default behavior is single sink routing
	destination := r.GetSinkDestination(url, rules)
	
	return RoutingDecision{
		URL:                   url,
		ShouldUseMultipleSinks: false,
		PrimarySink:           destination.SinkName,
		SecondarySinks:        []string{},
	}
}

// matchesPattern checks if input matches a pattern (supports basic wildcards)
func (r *OutputRoutingDecisionMaker) matchesPattern(input, pattern string) bool {
	if pattern == "*" {
		return true
	}
	
	// Simple wildcard matching - supports *.extension patterns
	if strings.HasPrefix(pattern, "*.") {
		extension := strings.TrimPrefix(pattern, "*.")
		return strings.HasSuffix(input, "."+extension)
	}
	
	// Exact match
	return input == pattern
}