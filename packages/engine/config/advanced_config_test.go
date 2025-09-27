package config

import (
	"testing"
	"time"
)

// TestGlobalSettings validates global settings functionality
func TestGlobalSettings(t *testing.T) {
	t.Run("should create default global settings", func(t *testing.T) {
		settings := DefaultGlobalSettings()
		
		if settings.MaxConcurrency != 10 {
			t.Errorf("Expected MaxConcurrency 10, got %d", settings.MaxConcurrency)
		}
		
		if settings.GlobalTimeout != 60*time.Second {
			t.Errorf("Expected GlobalTimeout 60s, got %v", settings.GlobalTimeout)
		}
		
		if settings.LogLevel != "info" {
			t.Errorf("Expected LogLevel 'info', got %s", settings.LogLevel)
		}
		
		if !settings.HealthCheckEnabled {
			t.Error("HealthCheckEnabled should be true by default")
		}
		
		if !settings.MetricsEnabled {
			t.Error("MetricsEnabled should be true by default")
		}
		
		if !settings.EnableTLS {
			t.Error("EnableTLS should be true by default")
		}
		
		if settings.AllowInsecure {
			t.Error("AllowInsecure should be false by default")
		}
	})
}

// TestConfigurationSerialization validates configuration metadata
func TestConfigurationSerialization(t *testing.T) {
	t.Run("should maintain configuration metadata", func(t *testing.T) {
		config := DefaultBusinessConfig()
		
		if config.Version == "" {
			t.Error("Configuration should have version information")
		}
		
		if config.Environment == "" {
			t.Error("Configuration should have environment information")
		}
		
		if config.CreatedAt.IsZero() {
			t.Error("Configuration should have creation timestamp")
		}
	})
	
	t.Run("should differentiate environment for composed configs", func(t *testing.T) {
		fetchPolicy := DefaultBusinessConfig().ExtractFetchPolicy()
		processPolicy := DefaultBusinessConfig().ExtractProcessPolicy()
		sinkPolicy := DefaultBusinessConfig().ExtractSinkPolicy()
		
		config, err := ComposeBusinessConfig(fetchPolicy, processPolicy, sinkPolicy)
		if err != nil {
			t.Errorf("Failed to compose config: %v", err)
		}
		
		if config.Environment != "production" {
			t.Errorf("Composed config should have production environment, got %s", config.Environment)
		}
	})
}

// TestAdvancedValidation validates complex validation scenarios
func TestAdvancedValidation(t *testing.T) {
	t.Run("should validate log levels", func(t *testing.T) {
		config := DefaultBusinessConfig()
		
		// Valid log levels
		validLevels := []string{"debug", "info", "warn", "error", "fatal", "DEBUG", "INFO"}
		for _, level := range validLevels {
			config.GlobalSettings.LogLevel = level
			if err := config.Validate(); err != nil {
				t.Errorf("Log level %s should be valid: %v", level, err)
			}
		}
		
		// Invalid log level
		config.GlobalSettings.LogLevel = "invalid"
		if err := config.Validate(); err == nil {
			t.Error("Invalid log level should be rejected")
		}
	})
	
	t.Run("should validate complex policy interactions", func(t *testing.T) {
		config := DefaultBusinessConfig()
		
		// Test fetch policy interactions
		config.FetchPolicy.MaxRetries = 5
		config.FetchPolicy.Timeout = 10 * time.Second
		if err := config.Validate(); err != nil {
			t.Errorf("Valid fetch policy should pass: %v", err)
		}
		
		// Test process policy content selector validation
		config.ProcessPolicy.ContentSelectors = []string{} // Empty is allowed
		if err := config.Validate(); err != nil {
			t.Errorf("Empty content selectors should be allowed: %v", err)
		}
	})
	
	t.Run("should handle nil policy components gracefully", func(t *testing.T) {
		config := NewUnifiedBusinessConfig()
		config.FetchPolicy = nil
		
		err := config.Validate()
		if err == nil {
			t.Error("Nil fetch policy should be rejected")
		}
		
		config = NewUnifiedBusinessConfig()
		config.ProcessPolicy = nil
		
		err = config.Validate()
		if err == nil {
			t.Error("Nil process policy should be rejected")
		}
		
		config = NewUnifiedBusinessConfig()
		config.SinkPolicy = nil
		
		err = config.Validate()
		if err == nil {
			t.Error("Nil sink policy should be rejected")
		}
		
		config = NewUnifiedBusinessConfig()
		config.GlobalSettings = nil
		
		err = config.Validate()
		if err == nil {
			t.Error("Nil global settings should be rejected")
		}
	})
}

// TestConfigurationBoundaries validates boundary conditions
func TestConfigurationBoundaries(t *testing.T) {
	t.Run("should handle zero values appropriately", func(t *testing.T) {
		config := NewUnifiedBusinessConfig()
		
		// Zero request delay should be valid (no delay)
		config.FetchPolicy.RequestDelay = 0
		config.ApplyDefaults()
		if config.FetchPolicy.RequestDelay == 0 {
			t.Error("ApplyDefaults should set non-zero request delay")
		}
		
		// Zero buffer size should be replaced with default
		config.SinkPolicy.BufferSize = 0
		config.ApplySinkDefaults()
		if config.SinkPolicy.BufferSize == 0 {
			t.Error("ApplySinkDefaults should set non-zero buffer size")
		}
	})
	
	t.Run("should handle maximum values", func(t *testing.T) {
		config := DefaultBusinessConfig()
		
		// Very large values should be accepted
		config.ProcessPolicy.MaxWordCount = 1000000
		config.SinkPolicy.BufferSize = 100000
		config.GlobalSettings.MaxConcurrency = 1000
		
		err := config.Validate()
		if err != nil {
			t.Errorf("Large valid values should be accepted: %v", err)
		}
	})
}

// TestDefaultPreservation validates that defaults don't override existing values
func TestDefaultPreservation(t *testing.T) {
	t.Run("should preserve custom values when applying defaults", func(t *testing.T) {
		config := NewUnifiedBusinessConfig()
		
		// Set custom values
		customUserAgent := "Custom Test Agent"
		customBufferSize := 2000
		customLogLevel := "debug"
		
		config.FetchPolicy.UserAgent = customUserAgent
		config.SinkPolicy.BufferSize = customBufferSize
		config.GlobalSettings.LogLevel = customLogLevel
		
		// Apply defaults
		config.ApplyDefaults()
		
		// Verify custom values are preserved
		if config.FetchPolicy.UserAgent != customUserAgent {
			t.Errorf("Custom user agent should be preserved: expected %s, got %s", 
				customUserAgent, config.FetchPolicy.UserAgent)
		}
		
		if config.SinkPolicy.BufferSize != customBufferSize {
			t.Errorf("Custom buffer size should be preserved: expected %d, got %d",
				customBufferSize, config.SinkPolicy.BufferSize)
		}
		
		if config.GlobalSettings.LogLevel != customLogLevel {
			t.Errorf("Custom log level should be preserved: expected %s, got %s",
				customLogLevel, config.GlobalSettings.LogLevel)
		}
	})
}

// TestPolicyExtraction validates policy extraction methods
func TestPolicyExtraction(t *testing.T) {
	t.Run("should extract policies safely", func(t *testing.T) {
		config := DefaultBusinessConfig()
		
		// Modify original config
		config.FetchPolicy.UserAgent = "Original Agent"
		
		// Extract policy
		extracted := config.ExtractFetchPolicy()
		
		// Modify extracted policy
		extracted.UserAgent = "Modified Agent"
		
		// Original should be unchanged
		if config.FetchPolicy.UserAgent != "Original Agent" {
			t.Error("Original config should not be affected by extracted policy modification")
		}
	})
	
	t.Run("should handle nil config extraction gracefully", func(t *testing.T) {
		var config *UnifiedBusinessConfig
		
		fetchPolicy := config.ExtractFetchPolicy()
		if fetchPolicy.UserAgent != "" {
			t.Error("Extracted policy from nil config should be empty")
		}
		
		processPolicy := config.ExtractProcessPolicy()
		if processPolicy.ExtractContent {
			t.Error("Extracted policy from nil config should have default false values")
		}
		
		sinkPolicy := config.ExtractSinkPolicy()
		if sinkPolicy.BufferSize != 0 {
			t.Error("Extracted policy from nil config should have zero values")
		}
	})
}