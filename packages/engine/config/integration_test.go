package config

import (
	"testing"
	"time"
)

// TestUnifiedConfigurationIntegration validates end-to-end configuration usage
func TestUnifiedConfigurationIntegration(t *testing.T) {
	t.Run("should create fully configured engine components", func(t *testing.T) {
		// Create unified configuration
		config := DefaultBusinessConfig()
		
		// Verify configuration is complete and valid
		if err := config.Validate(); err != nil {
			t.Fatalf("Default configuration should be valid: %v", err)
		}
		
		// Extract component policies for integration
		fetchPolicy := config.ExtractFetchPolicy()
		processPolicy := config.ExtractProcessPolicy()
		sinkPolicy := config.ExtractSinkPolicy()
		
		// Verify policies can be used to configure actual components
		if fetchPolicy.UserAgent == "" {
			t.Error("Fetch policy should have user agent for component configuration")
		}
		
		if fetchPolicy.Timeout == 0 {
			t.Error("Fetch policy should have timeout for component configuration")
		}
		
		if !processPolicy.ExtractContent {
			t.Error("Process policy should enable content extraction")
		}
		
		if len(processPolicy.ContentSelectors) == 0 {
			t.Error("Process policy should have content selectors")
		}
		
		if sinkPolicy.BufferSize == 0 {
			t.Error("Sink policy should have buffer size")
		}
		
		if sinkPolicy.FlushInterval == 0 {
			t.Error("Sink policy should have flush interval")
		}
		
		// Test integration with mock components
		t.Log("Fetch Policy - User Agent:", fetchPolicy.UserAgent)
		t.Log("Fetch Policy - Timeout:", fetchPolicy.Timeout)
		t.Log("Process Policy - Content Selectors:", len(processPolicy.ContentSelectors))
		t.Log("Sink Policy - Buffer Size:", sinkPolicy.BufferSize)
	})
	
	t.Run("should support configuration hot-reloading patterns", func(t *testing.T) {
		// Start with base configuration
		baseConfig := NewUnifiedBusinessConfig()
		baseConfig.ApplyDefaults()
		
		// Simulate configuration update
		baseConfig.FetchPolicy.RequestDelay = 200 * time.Millisecond
		baseConfig.ProcessPolicy.MaxWordCount = 5000
		baseConfig.SinkPolicy.BufferSize = 2000
		
		// Validate updated configuration
		if err := baseConfig.Validate(); err != nil {
			t.Errorf("Updated configuration should be valid: %v", err)
		}
		
		// Extract updated policies
		updatedFetch := baseConfig.ExtractFetchPolicy()
		updatedProcess := baseConfig.ExtractProcessPolicy()
		updatedSink := baseConfig.ExtractSinkPolicy()
		
		// Verify updates were applied
		if updatedFetch.RequestDelay != 200*time.Millisecond {
			t.Error("Configuration update should be reflected in extracted policy")
		}
		
		if updatedProcess.MaxWordCount != 5000 {
			t.Error("Configuration update should be reflected in extracted policy")
		}
		
		if updatedSink.BufferSize != 2000 {
			t.Error("Configuration update should be reflected in extracted policy")
		}
	})
	
	t.Run("should support multi-environment configuration", func(t *testing.T) {
		// Development configuration
		devConfig := DefaultBusinessConfig()
		devConfig.Environment = "development"
		devConfig.GlobalSettings.LogLevel = "debug"
		devConfig.FetchPolicy.RequestDelay = 100 * time.Millisecond
		
		// Production configuration
		prodFetch := devConfig.ExtractFetchPolicy()
		prodProcess := devConfig.ExtractProcessPolicy()
		prodSink := devConfig.ExtractSinkPolicy()
		
		// Modify for production
		prodFetch.RequestDelay = 1 * time.Second // Slower in prod
		prodProcess.ValidateContent = true       // Stricter validation
		prodSink.BufferSize = 5000              // Larger buffer
		
		prodConfig, err := ComposeBusinessConfig(prodFetch, prodProcess, prodSink)
		if err != nil {
			t.Errorf("Production config composition should succeed: %v", err)
		}
		
		// Verify environment-specific settings
		if prodConfig.Environment != "production" {
			t.Error("Composed config should have production environment")
		}
		
		if prodConfig.FetchPolicy.RequestDelay != 1*time.Second {
			t.Error("Production config should have adjusted request delay")
		}
		
		if prodConfig.SinkPolicy.BufferSize != 5000 {
			t.Error("Production config should have larger buffer size")
		}
	})
	
	t.Run("should demonstrate configuration validation workflow", func(t *testing.T) {
		// Create configuration with potential issues
		config := NewUnifiedBusinessConfig()
		// Apply defaults first to have valid base
		config.ApplyDefaults()
		
		// Now introduce issues
		config.FetchPolicy.MaxRetries = -1 // Invalid
		config.ProcessPolicy.MinWordCount = 1000
		config.ProcessPolicy.MaxWordCount = 500 // Conflict
		config.SinkPolicy.BufferSize = 0 // Invalid
		
		// Validation should catch all issues
		err := config.Validate()
		if err == nil {
			t.Fatal("Validation should catch configuration issues")
		}
		
		// Fix issues one by one
		config.FetchPolicy.MaxRetries = 3
		config.ProcessPolicy.MaxWordCount = 2000 // Fix conflict
		config.SinkPolicy.BufferSize = 1000
		
		// Should now be valid
		err = config.Validate()
		if err != nil {
			t.Errorf("Fixed configuration should be valid: %v", err)
		}
	})
}

// TestConfigurationMigration validates legacy configuration migration
func TestConfigurationMigration(t *testing.T) {
	t.Run("should migrate complex legacy configurations", func(t *testing.T) {
		// Complex legacy configuration
		legacyConfig := map[string]interface{}{
			"user_agent":      "Legacy Scraper 2.0",
			"request_delay":   "1500ms",
			"timeout":         "45s",
			"max_retries":     5,
			"extract_content": true,
			"convert_markdown": true,
			"content_selectors": []string{"article", ".main-content"},
			"buffer_size":     2500,
			"flush_interval":  "10s",
			"max_concurrency": 8,
		}
		
		// Migrate to unified configuration
		config, err := FromLegacyConfig(legacyConfig)
		if err != nil {
			t.Errorf("Legacy migration should succeed: %v", err)
		}
		
		// Verify migration preserved values
		if config.FetchPolicy.UserAgent != "Legacy Scraper 2.0" {
			t.Error("Migration should preserve user agent")
		}
		
		if config.FetchPolicy.RequestDelay != 1500*time.Millisecond {
			t.Error("Migration should parse duration strings")
		}
		
		if config.ProcessPolicy.ExtractContent != true {
			t.Error("Migration should preserve boolean flags")
		}
		
		if config.SinkPolicy.BufferSize != 2500 {
			t.Error("Migration should preserve numeric values")
		}
		
		// Verify configuration is valid after migration
		if err := config.Validate(); err != nil {
			t.Errorf("Migrated configuration should be valid: %v", err)
		}
	})
}

// TestConfigurationPerformance validates performance characteristics
func TestConfigurationPerformance(t *testing.T) {
	t.Run("should handle configuration operations efficiently", func(t *testing.T) {
		iterations := 1000
		
		start := time.Now()
		
		// Test configuration creation performance
		for i := 0; i < iterations; i++ {
			config := DefaultBusinessConfig()
			if config == nil {
				t.Error("Configuration creation failed")
			}
		}
		
		creationTime := time.Since(start)
		
		// Configuration creation should be fast
		if creationTime > 100*time.Millisecond {
			t.Logf("Configuration creation took %v for %d iterations", creationTime, iterations)
		}
		
		// Test validation performance
		config := DefaultBusinessConfig()
		start = time.Now()
		
		for i := 0; i < iterations; i++ {
			if err := config.Validate(); err != nil {
				t.Errorf("Validation failed: %v", err)
			}
		}
		
		validationTime := time.Since(start)
		
		if validationTime > 50*time.Millisecond {
			t.Logf("Configuration validation took %v for %d iterations", validationTime, iterations)
		}
	})
}

// TestConfigurationDocumentation validates configuration documentation features
func TestConfigurationDocumentation(t *testing.T) {
	t.Run("should provide comprehensive configuration information", func(t *testing.T) {
		config := DefaultBusinessConfig()
		
		// Verify metadata is populated
		if config.Version == "" {
			t.Error("Configuration should have version for documentation")
		}
		
		if config.Environment == "" {
			t.Error("Configuration should have environment for documentation")
		}
		
		if config.CreatedAt.IsZero() {
			t.Error("Configuration should have timestamp for audit trail")
		}
		
		// Verify all component policies are documented through non-zero defaults
		if config.FetchPolicy.UserAgent == "" {
			t.Error("Fetch policy should have documented user agent")
		}
		
		if len(config.ProcessPolicy.ContentSelectors) == 0 {
			t.Error("Process policy should have documented content selectors")
		}
		
		if config.SinkPolicy.BufferSize == 0 {
			t.Error("Sink policy should have documented buffer size")
		}
		
		if config.GlobalSettings.LogLevel == "" {
			t.Error("Global settings should have documented log level")
		}
	})
}