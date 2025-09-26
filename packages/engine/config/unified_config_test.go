package config

import (
	"testing"
	"time"

	"ariadne/packages/engine/crawler"
	"ariadne/packages/engine/output"
	"ariadne/packages/engine/processor"
)

// TestUnifiedBusinessConfig validates unified configuration design
func TestUnifiedBusinessConfig(t *testing.T) {
	t.Run("should provide unified business configuration", func(t *testing.T) {
		config := NewUnifiedBusinessConfig()

		// Unified configuration should exist
		if config == nil {
			t.Fatal("NewUnifiedBusinessConfig should return a valid configuration")
		}

		// Should contain all component policies
		if config.FetchPolicy == nil {
			t.Error("UnifiedBusinessConfig should contain FetchPolicy")
		}

		if config.ProcessPolicy == nil {
			t.Error("UnifiedBusinessConfig should contain ProcessPolicy")
		}

		if config.SinkPolicy == nil {
			t.Error("UnifiedBusinessConfig should contain SinkPolicy")
		}
	})

	t.Run("should provide sensible defaults", func(t *testing.T) {
		config := DefaultBusinessConfig()

		// Verify fetch policy defaults
		if config.FetchPolicy.UserAgent == "" {
			t.Error("Default fetch policy should have UserAgent")
		}

		if config.FetchPolicy.RequestDelay == 0 {
			t.Error("Default fetch policy should have RequestDelay")
		}

		if config.FetchPolicy.Timeout == 0 {
			t.Error("Default fetch policy should have Timeout")
		}

		// Verify process policy defaults
		if !config.ProcessPolicy.ExtractContent {
			t.Error("Default process policy should extract content")
		}

		if len(config.ProcessPolicy.ContentSelectors) == 0 {
			t.Error("Default process policy should have content selectors")
		}

		// Verify sink policy defaults
		if config.SinkPolicy.BufferSize == 0 {
			t.Error("Default sink policy should have buffer size")
		}

		if config.SinkPolicy.FlushInterval == 0 {
			t.Error("Default sink policy should have flush interval")
		}
	})
}

// TestConfigurationValidation validates configuration validation system
func TestConfigurationValidation(t *testing.T) {
	t.Run("should validate complete configuration", func(t *testing.T) {
		config := DefaultBusinessConfig()

		err := config.Validate()
		if err != nil {
			t.Errorf("Default configuration should be valid: %v", err)
		}
	})

	t.Run("should detect invalid fetch configuration", func(t *testing.T) {
		config := DefaultBusinessConfig()
		config.FetchPolicy.Timeout = -1 * time.Second // Invalid timeout

		err := config.Validate()
		if err == nil {
			t.Error("Should detect invalid timeout in fetch policy")
		}
	})

	t.Run("should detect invalid process configuration", func(t *testing.T) {
		config := DefaultBusinessConfig()
		config.ProcessPolicy.MaxWordCount = -1 // Invalid word count

		err := config.Validate()
		if err == nil {
			t.Error("Should detect invalid word count in process policy")
		}
	})

	t.Run("should detect invalid sink configuration", func(t *testing.T) {
		config := DefaultBusinessConfig()
		config.SinkPolicy.BufferSize = 0 // Invalid buffer size

		err := config.Validate()
		if err == nil {
			t.Error("Should detect invalid buffer size in sink policy")
		}
	})
}

// TestConfigurationComposition validates configuration composition
func TestConfigurationComposition(t *testing.T) {
	t.Run("should compose individual policies", func(t *testing.T) {
		fetchPolicy := crawler.FetchPolicy{
			UserAgent:    "Test Agent",
			RequestDelay: 100 * time.Millisecond,
			Timeout:      5 * time.Second,
		}

		processPolicy := processor.ProcessPolicy{
			ExtractContent:    true,
			ContentSelectors:  []string{"article", "main"},
			ConvertToMarkdown: true,
		}

		sinkPolicy := output.SinkPolicy{
			BufferSize:    500,
			FlushInterval: 2 * time.Second,
		}

		config, err := ComposeBusinessConfig(fetchPolicy, processPolicy, sinkPolicy)
		if err != nil {
			t.Errorf("Should compose valid policies: %v", err)
		}

		if config.FetchPolicy.UserAgent != "Test Agent" {
			t.Error("Composed config should preserve fetch policy")
		}

		if !config.ProcessPolicy.ExtractContent {
			t.Error("Composed config should preserve process policy")
		}

		if config.SinkPolicy.BufferSize != 500 {
			t.Error("Composed config should preserve sink policy")
		}
	})

	t.Run("should reject invalid policy composition", func(t *testing.T) {
		fetchPolicy := crawler.FetchPolicy{
			Timeout: -1 * time.Second, // Invalid
		}

		processPolicy := processor.ProcessPolicy{
			ExtractContent: true,
		}

		sinkPolicy := output.SinkPolicy{
			BufferSize: 100,
		}

		_, err := ComposeBusinessConfig(fetchPolicy, processPolicy, sinkPolicy)
		if err == nil {
			t.Error("Should reject invalid policy composition")
		}
	})
}

// TestConfigurationCompatibility validates backward compatibility
func TestConfigurationCompatibility(t *testing.T) {
	t.Run("should convert from unified config to component policies", func(t *testing.T) {
		unified := DefaultBusinessConfig()

		// Extract individual policies
		fetchPolicy := unified.ExtractFetchPolicy()
		processPolicy := unified.ExtractProcessPolicy()
		sinkPolicy := unified.ExtractSinkPolicy()

		// Verify extraction preserved values
		if fetchPolicy.UserAgent != unified.FetchPolicy.UserAgent {
			t.Error("Fetch policy extraction should preserve values")
		}

		if processPolicy.ExtractContent != unified.ProcessPolicy.ExtractContent {
			t.Error("Process policy extraction should preserve values")
		}

		if sinkPolicy.BufferSize != unified.SinkPolicy.BufferSize {
			t.Error("Sink policy extraction should preserve values")
		}
	})

	t.Run("should create unified config from legacy config", func(t *testing.T) {
		// Mock legacy configuration structure
		legacyConfig := map[string]interface{}{
			"user_agent":      "Legacy Agent",
			"request_delay":   "200ms",
			"extract_content": true,
			"buffer_size":     1000,
		}

		unified, err := FromLegacyConfig(legacyConfig)
		if err != nil {
			t.Errorf("Should convert from legacy config: %v", err)
		}

		if unified.FetchPolicy.UserAgent != "Legacy Agent" {
			t.Error("Legacy conversion should preserve user agent")
		}

		if unified.ProcessPolicy.ExtractContent != true {
			t.Error("Legacy conversion should preserve extract content flag")
		}

		if unified.SinkPolicy.BufferSize != 1000 {
			t.Error("Legacy conversion should preserve buffer size")
		}
	})
}

// TestConfigurationEdgeCases validates edge case handling
func TestConfigurationEdgeCases(t *testing.T) {
	t.Run("should handle nil policies gracefully", func(t *testing.T) {
		var config *UnifiedBusinessConfig

		err := config.Validate()
		if err == nil {
			t.Error("Should handle nil config validation gracefully")
		}
	})

	t.Run("should handle empty string values", func(t *testing.T) {
		config := DefaultBusinessConfig()
		config.FetchPolicy.UserAgent = ""

		err := config.Validate()
		if err == nil {
			t.Error("Should reject empty user agent")
		}
	})

	t.Run("should handle zero duration values", func(t *testing.T) {
		config := DefaultBusinessConfig()
		config.FetchPolicy.RequestDelay = 0

		// Zero delay should be valid (no delay)
		err := config.Validate()
		if err != nil {
			t.Errorf("Zero request delay should be valid: %v", err)
		}
	})

	t.Run("should handle negative numeric values", func(t *testing.T) {
		config := DefaultBusinessConfig()
		config.ProcessPolicy.MinWordCount = -1

		err := config.Validate()
		if err == nil {
			t.Error("Should reject negative word count")
		}
	})

	t.Run("should handle conflicting policies", func(t *testing.T) {
		config := DefaultBusinessConfig()
		config.ProcessPolicy.MinWordCount = 1000
		config.ProcessPolicy.MaxWordCount = 500 // Max < Min

		err := config.Validate()
		if err == nil {
			t.Error("Should detect conflicting word count limits")
		}
	})
}

// TestConfigurationDefaults validates default value system
func TestConfigurationDefaults(t *testing.T) {
	t.Run("should apply component defaults", func(t *testing.T) {
		config := NewUnifiedBusinessConfig()

		// Apply defaults
		config.ApplyDefaults()

		// Verify all defaults are applied
		if config.FetchPolicy.UserAgent == "" {
			t.Error("ApplyDefaults should set fetch policy defaults")
		}

		if config.ProcessPolicy.TimeoutDuration == 0 {
			t.Error("ApplyDefaults should set process policy defaults")
		}

		if config.SinkPolicy.MaxRetries == 0 {
			t.Error("ApplyDefaults should set sink policy defaults")
		}
	})

	t.Run("should preserve existing values when applying defaults", func(t *testing.T) {
		config := NewUnifiedBusinessConfig()
		config.FetchPolicy.UserAgent = "Custom Agent"

		config.ApplyDefaults()

		// Custom value should be preserved
		if config.FetchPolicy.UserAgent != "Custom Agent" {
			t.Error("ApplyDefaults should preserve existing values")
		}
	})

	t.Run("should apply selective defaults", func(t *testing.T) {
		config := NewUnifiedBusinessConfig()

		// Apply only fetch defaults
		config.ApplyFetchDefaults()

		if config.FetchPolicy.UserAgent == "" {
			t.Error("ApplyFetchDefaults should set fetch defaults")
		}

		// Other policies should remain uninitialized
		if config.ProcessPolicy.TimeoutDuration != 0 {
			t.Error("ApplyFetchDefaults should not affect process policy")
		}
	})
}
