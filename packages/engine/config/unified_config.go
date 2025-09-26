package config

import (
	"fmt"
	"strings"
	"time"

	"ariadne/packages/engine/crawler"
	"ariadne/packages/engine/output"
	"ariadne/packages/engine/processor"
)

// UnifiedBusinessConfig provides a unified configuration for all engine components
type UnifiedBusinessConfig struct {
	// Component policies
	FetchPolicy   *crawler.FetchPolicy
	ProcessPolicy *processor.ProcessPolicy
	SinkPolicy    *output.SinkPolicy

	// Global settings
	GlobalSettings *GlobalSettings

	// Metadata
	Version     string
	Environment string
	CreatedAt   time.Time
}

// GlobalSettings contains cross-cutting configuration
type GlobalSettings struct {
	// Performance settings
	MaxConcurrency     int
	GlobalTimeout      time.Duration
	HealthCheckEnabled bool

	// Monitoring settings
	MetricsEnabled bool
	LogLevel       string
	TraceEnabled   bool

	// Security settings
	EnableTLS     bool
	AllowInsecure bool
	TrustedCerts  []string
}

// NewUnifiedBusinessConfig creates a new unified configuration with empty policies
func NewUnifiedBusinessConfig() *UnifiedBusinessConfig {
	return &UnifiedBusinessConfig{
		FetchPolicy:    &crawler.FetchPolicy{},
		ProcessPolicy:  &processor.ProcessPolicy{},
		SinkPolicy:     &output.SinkPolicy{},
		GlobalSettings: &GlobalSettings{},
		Version:        "1.0.0",
		Environment:    "development",
		CreatedAt:      time.Now(),
	}
}

// DefaultBusinessConfig creates a unified configuration with sensible defaults
func DefaultBusinessConfig() *UnifiedBusinessConfig {
	config := NewUnifiedBusinessConfig()
	config.ApplyDefaults()
	return config
}

// ComposeBusinessConfig creates a unified configuration from individual policies
func ComposeBusinessConfig(fetchPolicy crawler.FetchPolicy, processPolicy processor.ProcessPolicy, sinkPolicy output.SinkPolicy) (*UnifiedBusinessConfig, error) {
	config := &UnifiedBusinessConfig{
		FetchPolicy:    &fetchPolicy,
		ProcessPolicy:  &processPolicy,
		SinkPolicy:     &sinkPolicy,
		GlobalSettings: DefaultGlobalSettings(),
		Version:        "1.0.0",
		Environment:    "production", // Composed configs are typically for production
		CreatedAt:      time.Now(),
	}

	// Validate the composed configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy composition: %w", err)
	}

	return config, nil
}

// FromLegacyConfig creates a unified configuration from legacy configuration map
func FromLegacyConfig(legacyConfig map[string]interface{}) (*UnifiedBusinessConfig, error) {
	config := NewUnifiedBusinessConfig()

	// Convert fetch policy settings
	if userAgent, ok := legacyConfig["user_agent"].(string); ok {
		config.FetchPolicy.UserAgent = userAgent
	}

	if delayStr, ok := legacyConfig["request_delay"].(string); ok {
		if delay, err := time.ParseDuration(delayStr); err == nil {
			config.FetchPolicy.RequestDelay = delay
		}
	}

	// Convert process policy settings
	if extractContent, ok := legacyConfig["extract_content"].(bool); ok {
		config.ProcessPolicy.ExtractContent = extractContent
	}

	// Convert sink policy settings
	if bufferSize, ok := legacyConfig["buffer_size"].(int); ok {
		config.SinkPolicy.BufferSize = bufferSize
	}

	// Apply defaults for missing values
	config.ApplyDefaults()

	return config, nil
}

// Validate performs comprehensive validation of the unified configuration
func (c *UnifiedBusinessConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("unified configuration cannot be nil")
	}

	// Validate fetch policy
	if err := c.validateFetchPolicy(); err != nil {
		return fmt.Errorf("fetch policy validation failed: %w", err)
	}

	// Validate process policy
	if err := c.validateProcessPolicy(); err != nil {
		return fmt.Errorf("process policy validation failed: %w", err)
	}

	// Validate sink policy
	if err := c.validateSinkPolicy(); err != nil {
		return fmt.Errorf("sink policy validation failed: %w", err)
	}

	// Validate global settings
	if err := c.validateGlobalSettings(); err != nil {
		return fmt.Errorf("global settings validation failed: %w", err)
	}

	return nil
}

// validateFetchPolicy validates the fetch policy
func (c *UnifiedBusinessConfig) validateFetchPolicy() error {
	if c.FetchPolicy == nil {
		return fmt.Errorf("fetch policy cannot be nil")
	}

	// Validate user agent
	if strings.TrimSpace(c.FetchPolicy.UserAgent) == "" {
		return fmt.Errorf("user agent cannot be empty")
	}

	// Validate timeout
	if c.FetchPolicy.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative: %v", c.FetchPolicy.Timeout)
	}

	// Validate retry count
	if c.FetchPolicy.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative: %d", c.FetchPolicy.MaxRetries)
	}

	return nil
}

// validateProcessPolicy validates the process policy
func (c *UnifiedBusinessConfig) validateProcessPolicy() error {
	if c.ProcessPolicy == nil {
		return fmt.Errorf("process policy cannot be nil")
	}

	// Validate word count limits
	if c.ProcessPolicy.MinWordCount < 0 {
		return fmt.Errorf("min word count cannot be negative: %d", c.ProcessPolicy.MinWordCount)
	}

	if c.ProcessPolicy.MaxWordCount < 0 {
		return fmt.Errorf("max word count cannot be negative: %d", c.ProcessPolicy.MaxWordCount)
	}

	// Validate conflicting word counts
	if c.ProcessPolicy.MaxWordCount > 0 && c.ProcessPolicy.MinWordCount > c.ProcessPolicy.MaxWordCount {
		return fmt.Errorf("min word count (%d) cannot exceed max word count (%d)",
			c.ProcessPolicy.MinWordCount, c.ProcessPolicy.MaxWordCount)
	}

	// Validate timeout
	if c.ProcessPolicy.TimeoutDuration < 0 {
		return fmt.Errorf("timeout duration cannot be negative: %v", c.ProcessPolicy.TimeoutDuration)
	}

	return nil
}

// validateSinkPolicy validates the sink policy
func (c *UnifiedBusinessConfig) validateSinkPolicy() error {
	if c.SinkPolicy == nil {
		return fmt.Errorf("sink policy cannot be nil")
	}

	// Validate buffer size
	if c.SinkPolicy.BufferSize <= 0 {
		return fmt.Errorf("buffer size must be positive: %d", c.SinkPolicy.BufferSize)
	}

	// Validate retry settings
	if c.SinkPolicy.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative: %d", c.SinkPolicy.MaxRetries)
	}

	if c.SinkPolicy.RetryDelay < 0 {
		return fmt.Errorf("retry delay cannot be negative: %v", c.SinkPolicy.RetryDelay)
	}

	// Validate flush interval
	if c.SinkPolicy.FlushInterval < 0 {
		return fmt.Errorf("flush interval cannot be negative: %v", c.SinkPolicy.FlushInterval)
	}

	return nil
}

// validateGlobalSettings validates global settings
func (c *UnifiedBusinessConfig) validateGlobalSettings() error {
	if c.GlobalSettings == nil {
		return fmt.Errorf("global settings cannot be nil")
	}

	// Validate concurrency
	if c.GlobalSettings.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be positive: %d", c.GlobalSettings.MaxConcurrency)
	}

	// Validate timeout
	if c.GlobalSettings.GlobalTimeout < 0 {
		return fmt.Errorf("global timeout cannot be negative: %v", c.GlobalSettings.GlobalTimeout)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[strings.ToLower(c.GlobalSettings.LogLevel)] {
		return fmt.Errorf("invalid log level: %s", c.GlobalSettings.LogLevel)
	}

	return nil
}

// ApplyDefaults applies default values to all components
func (c *UnifiedBusinessConfig) ApplyDefaults() {
	if c == nil {
		return
	}

	c.ApplyFetchDefaults()
	c.ApplyProcessDefaults()
	c.ApplySinkDefaults()
	c.ApplyGlobalDefaults()
}

// ApplyFetchDefaults applies fetch policy defaults
func (c *UnifiedBusinessConfig) ApplyFetchDefaults() {
	if c == nil || c.FetchPolicy == nil {
		return
	}

	if c.FetchPolicy.UserAgent == "" {
		c.FetchPolicy.UserAgent = "Ariadne/1.0 (Engine Configuration)"
	}

	if c.FetchPolicy.RequestDelay == 0 {
		c.FetchPolicy.RequestDelay = 500 * time.Millisecond
	}

	if c.FetchPolicy.Timeout == 0 {
		c.FetchPolicy.Timeout = 30 * time.Second
	}

	if c.FetchPolicy.MaxRetries == 0 {
		c.FetchPolicy.MaxRetries = 3
	}

	if !c.FetchPolicy.RespectRobots {
		c.FetchPolicy.RespectRobots = true
	}

	if !c.FetchPolicy.FollowRedirects {
		c.FetchPolicy.FollowRedirects = true
	}

	if c.FetchPolicy.MaxDepth == 0 {
		c.FetchPolicy.MaxDepth = 10
	}
}

// ApplyProcessDefaults applies process policy defaults
func (c *UnifiedBusinessConfig) ApplyProcessDefaults() {
	if c == nil || c.ProcessPolicy == nil {
		return
	}

	if !c.ProcessPolicy.ExtractContent {
		c.ProcessPolicy.ExtractContent = true
	}

	if len(c.ProcessPolicy.ContentSelectors) == 0 {
		c.ProcessPolicy.ContentSelectors = []string{
			"article", "main", ".content", "#content", ".post-content",
		}
	}

	if len(c.ProcessPolicy.RemoveSelectors) == 0 {
		c.ProcessPolicy.RemoveSelectors = []string{
			"script", "style", "nav", "footer", ".ads", ".advertisement",
		}
	}

	if !c.ProcessPolicy.ConvertToMarkdown {
		c.ProcessPolicy.ConvertToMarkdown = true
	}

	if !c.ProcessPolicy.ExtractMetadata {
		c.ProcessPolicy.ExtractMetadata = true
	}

	if !c.ProcessPolicy.ValidateContent {
		c.ProcessPolicy.ValidateContent = true
	}

	if c.ProcessPolicy.MinWordCount == 0 {
		c.ProcessPolicy.MinWordCount = 10
	}

	if c.ProcessPolicy.MaxWordCount == 0 {
		c.ProcessPolicy.MaxWordCount = 50000
	}

	if c.ProcessPolicy.TimeoutDuration == 0 {
		c.ProcessPolicy.TimeoutDuration = 10 * time.Second
	}

	if c.ProcessPolicy.MaxRetries == 0 {
		c.ProcessPolicy.MaxRetries = 2
	}
}

// ApplySinkDefaults applies sink policy defaults
func (c *UnifiedBusinessConfig) ApplySinkDefaults() {
	if c == nil || c.SinkPolicy == nil {
		return
	}

	if c.SinkPolicy.MaxRetries == 0 {
		c.SinkPolicy.MaxRetries = 3
	}

	if c.SinkPolicy.RetryDelay == 0 {
		c.SinkPolicy.RetryDelay = 100 * time.Millisecond
	}

	if c.SinkPolicy.BufferSize == 0 {
		c.SinkPolicy.BufferSize = 1000
	}

	if c.SinkPolicy.FlushInterval == 0 {
		c.SinkPolicy.FlushInterval = 5 * time.Second
	}

	if c.SinkPolicy.MaxConcurrency == 0 {
		c.SinkPolicy.MaxConcurrency = 4
	}

	if c.SinkPolicy.TimeoutDuration == 0 {
		c.SinkPolicy.TimeoutDuration = 30 * time.Second
	}
}

// ApplyGlobalDefaults applies global settings defaults
func (c *UnifiedBusinessConfig) ApplyGlobalDefaults() {
	if c == nil || c.GlobalSettings == nil {
		return
	}

	if c.GlobalSettings.MaxConcurrency == 0 {
		c.GlobalSettings.MaxConcurrency = 10
	}

	if c.GlobalSettings.GlobalTimeout == 0 {
		c.GlobalSettings.GlobalTimeout = 60 * time.Second
	}

	if c.GlobalSettings.LogLevel == "" {
		c.GlobalSettings.LogLevel = "info"
	}

	if !c.GlobalSettings.HealthCheckEnabled {
		c.GlobalSettings.HealthCheckEnabled = true
	}

	if !c.GlobalSettings.MetricsEnabled {
		c.GlobalSettings.MetricsEnabled = true
	}
}

// ExtractFetchPolicy returns a copy of the fetch policy
func (c *UnifiedBusinessConfig) ExtractFetchPolicy() crawler.FetchPolicy {
	if c == nil || c.FetchPolicy == nil {
		return crawler.FetchPolicy{}
	}
	return *c.FetchPolicy
}

// ExtractProcessPolicy returns a copy of the process policy
func (c *UnifiedBusinessConfig) ExtractProcessPolicy() processor.ProcessPolicy {
	if c == nil || c.ProcessPolicy == nil {
		return processor.ProcessPolicy{}
	}
	return *c.ProcessPolicy
}

// ExtractSinkPolicy returns a copy of the sink policy
func (c *UnifiedBusinessConfig) ExtractSinkPolicy() output.SinkPolicy {
	if c == nil || c.SinkPolicy == nil {
		return output.SinkPolicy{}
	}
	return *c.SinkPolicy
}

// DefaultGlobalSettings returns sensible global settings defaults
func DefaultGlobalSettings() *GlobalSettings {
	return &GlobalSettings{
		MaxConcurrency:     10,
		GlobalTimeout:      60 * time.Second,
		HealthCheckEnabled: true,
		MetricsEnabled:     true,
		LogLevel:           "info",
		TraceEnabled:       false,
		EnableTLS:          true,
		AllowInsecure:      false,
		TrustedCerts:       []string{},
	}
}
