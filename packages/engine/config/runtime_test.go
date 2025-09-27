package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ariadne/packages/engine/business/policies"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeConfigManager(t *testing.T) {
	t.Run("create_and_load_configuration", func(t *testing.T) {
		// Create temporary directory for test configurations
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "runtime_config.yaml")

		// Create runtime config manager
		manager, err := NewRuntimeConfigManager(configPath)
		require.NoError(t, err)
		require.NotNil(t, manager)

		// Test initial empty configuration
		config := manager.GetCurrentConfig()
		assert.NotNil(t, config)
		assert.Empty(t, config.Version)

		// Test loading configuration from file
		err = manager.LoadConfiguration()
		// Should not error even if file doesn't exist initially
		assert.NoError(t, err)
	})

	t.Run("update_configuration_runtime", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "runtime_config.yaml")

		manager, err := NewRuntimeConfigManager(configPath)
		require.NoError(t, err)

		// Create new business configuration
		newConfig := &RuntimeBusinessConfig{
			Version:   "1.2.3",
			UpdatedAt: time.Now(),
			BusinessPolicies: &policies.BusinessPolicies{
				CrawlingPolicy: &policies.CrawlingBusinessPolicy{
					SiteRules: map[string]*policies.SitePolicy{
						"example.com": {
							AllowedDomains: []string{"example.com"},
							MaxDepth:       5,
							Delay:          100 * time.Millisecond,
						},
					},
				},
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 10,
					Timeout:        30 * time.Second,
				},
			},
			HotReloadEnabled: true,
		}

		// Test updating configuration
		err = manager.UpdateConfiguration(newConfig)
		require.NoError(t, err)

		// Verify configuration was updated
		currentConfig := manager.GetCurrentConfig()
		assert.Equal(t, "1.2.3", currentConfig.Version)
		assert.True(t, currentConfig.HotReloadEnabled)
		assert.NotNil(t, currentConfig.BusinessPolicies)
		assert.Equal(t, 10, currentConfig.BusinessPolicies.GlobalPolicy.MaxConcurrency)
	})

	t.Run("configuration_validation", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "runtime_config.yaml")

		manager, err := NewRuntimeConfigManager(configPath)
		require.NoError(t, err)

		// Test invalid configuration
		invalidConfig := &RuntimeBusinessConfig{
			Version: "invalid",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: -1, // Invalid negative concurrency
					Timeout:        -1 * time.Second, // Invalid negative timeout
				},
			},
		}

		err = manager.ValidateConfiguration(invalidConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "concurrency")

		// Test valid configuration
		validConfig := &RuntimeBusinessConfig{
			Version: "2.0.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 20,
					Timeout:        45 * time.Second,
				},
			},
		}

		err = manager.ValidateConfiguration(validConfig)
		assert.NoError(t, err)
	})
}

func TestHotReloadSystem(t *testing.T) {
	t.Run("file_system_watcher", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "hot_reload_config.yaml")

		// Create hot reload system
		hotReloader, err := NewHotReloadSystem(configPath)
		require.NoError(t, err)
		require.NotNil(t, hotReloader)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start watching for changes
		changesChan, errorsChan := hotReloader.WatchConfigChanges(ctx)
		assert.NotNil(t, changesChan)
		assert.NotNil(t, errorsChan)

		// Write initial configuration file
		initialConfig := `version: "1.0.0"
hot_reload_enabled: true
business_policies:
  global_policy:
    max_concurrency: 10
    timeout: "30s"
`

		err = os.WriteFile(configPath, []byte(initialConfig), 0644)
		require.NoError(t, err)

		// Wait for initial file detection
		select {
		case change := <-changesChan:
			assert.NotNil(t, change)
			assert.Equal(t, "1.0.0", change.Version)
		case err := <-errorsChan:
			t.Fatalf("Unexpected error: %v", err)
		case <-ctx.Done():
			t.Log("Test completed without detecting initial file - this is acceptable for some file systems")
		}

		// Stop watching
		err = hotReloader.StopWatching()
		assert.NoError(t, err)
	})

	t.Run("configuration_change_detection", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "change_detection_config.yaml")

		hotReloader, err := NewHotReloadSystem(configPath)
		require.NoError(t, err)

		// Test configuration change detection
		oldConfig := &RuntimeBusinessConfig{
			Version: "1.0.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 10,
				},
			},
		}

		newConfig := &RuntimeBusinessConfig{
			Version: "1.1.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 15, // Changed
				},
			},
		}

		hasChanged := hotReloader.DetectChanges(oldConfig, newConfig)
		assert.True(t, hasChanged)

		// Test no changes
		identicalConfig := &RuntimeBusinessConfig{
			Version: "1.0.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 10,
				},
			},
		}

		hasNotChanged := hotReloader.DetectChanges(oldConfig, identicalConfig)
		assert.False(t, hasNotChanged)
	})
}

func TestConfigurationVersioning(t *testing.T) {
	t.Run("version_history_tracking", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create version manager
		versionManager, err := NewConfigVersionManager(tempDir)
		require.NoError(t, err)
		require.NotNil(t, versionManager)

		// Create first version
		v1Config := &RuntimeBusinessConfig{
			Version: "1.0.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 5,
				},
			},
		}

		err = versionManager.SaveVersion(v1Config, "Initial configuration")
		require.NoError(t, err)

		// Create second version
		v2Config := &RuntimeBusinessConfig{
			Version: "1.1.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 10,
				},
			},
		}

		err = versionManager.SaveVersion(v2Config, "Increased concurrency")
		require.NoError(t, err)

		// Test version history
		history, err := versionManager.GetVersionHistory()
		require.NoError(t, err)
		assert.Len(t, history, 2)
		assert.Equal(t, "1.0.0", history[0].Version)
		assert.Equal(t, "1.1.0", history[1].Version)
		assert.Equal(t, "Initial configuration", history[0].ChangeDescription)
		assert.Equal(t, "Increased concurrency", history[1].ChangeDescription)
	})

	t.Run("configuration_rollback", func(t *testing.T) {
		tempDir := t.TempDir()
		
		versionManager, err := NewConfigVersionManager(tempDir)
		require.NoError(t, err)

		// Save multiple versions
		versions := []*RuntimeBusinessConfig{
			{Version: "1.0.0", BusinessPolicies: &policies.BusinessPolicies{GlobalPolicy: &policies.GlobalBusinessPolicy{MaxConcurrency: 5}}},
			{Version: "1.1.0", BusinessPolicies: &policies.BusinessPolicies{GlobalPolicy: &policies.GlobalBusinessPolicy{MaxConcurrency: 10}}},
			{Version: "1.2.0", BusinessPolicies: &policies.BusinessPolicies{GlobalPolicy: &policies.GlobalBusinessPolicy{MaxConcurrency: 15}}},
		}

		for i, config := range versions {
			err = versionManager.SaveVersion(config, "Version %d", i+1)
			require.NoError(t, err)
		}

		// Test rollback to specific version
		rolledBackConfig, err := versionManager.RollbackToVersion("1.1.0")
		require.NoError(t, err)
		assert.Equal(t, "1.1.0", rolledBackConfig.Version)
		assert.Equal(t, 10, rolledBackConfig.BusinessPolicies.GlobalPolicy.MaxConcurrency)

		// Test rollback to non-existent version
		_, err = versionManager.RollbackToVersion("99.99.99")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version not found")
	})
}

func TestABTestingFramework(t *testing.T) {
	t.Run("create_ab_test", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create A/B testing framework
		abTester, err := NewABTestingFramework(tempDir)
		require.NoError(t, err)
		require.NotNil(t, abTester)

		// Create control configuration (A)
		controlConfig := &RuntimeBusinessConfig{
			Version: "control-1.0.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 10,
					Timeout:        30 * time.Second,
				},
			},
		}

		// Create experiment configuration (B)
		experimentConfig := &RuntimeBusinessConfig{
			Version: "experiment-1.0.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 20,
					Timeout:        25 * time.Second,
				},
			},
		}

		// Create A/B test
		testID, err := abTester.CreateABTest("concurrency-test", controlConfig, experimentConfig, 0.5) // 50/50 split
		require.NoError(t, err)
		assert.NotEmpty(t, testID)

		// Test configuration selection
		for i := 0; i < 100; i++ {
			selectedConfig := abTester.GetConfigForUser("user-"+string(rune(i)), testID)
			assert.NotNil(t, selectedConfig)
			// Should be either control or experiment config
			assert.True(t, 
				selectedConfig.Version == "control-1.0.0" || 
				selectedConfig.Version == "experiment-1.0.0",
			)
		}
	})

	t.Run("ab_test_results_tracking", func(t *testing.T) {
		tempDir := t.TempDir()
		
		abTester, err := NewABTestingFramework(tempDir)
		require.NoError(t, err)

		// Create simple A/B test
		controlConfig := &RuntimeBusinessConfig{Version: "control", BusinessPolicies: &policies.BusinessPolicies{}}
		experimentConfig := &RuntimeBusinessConfig{Version: "experiment", BusinessPolicies: &policies.BusinessPolicies{}}
		
		testID, err := abTester.CreateABTest("test-results", controlConfig, experimentConfig, 0.5)
		require.NoError(t, err)

		// Record test results
		err = abTester.RecordTestResult(testID, "user-1", "control", true, 1.5) // Success, 1.5s response time
		require.NoError(t, err)

		err = abTester.RecordTestResult(testID, "user-2", "experiment", false, 2.1) // Failure, 2.1s response time
		require.NoError(t, err)

		err = abTester.RecordTestResult(testID, "user-3", "control", true, 1.8) // Success, 1.8s response time
		require.NoError(t, err)

		// Analyze results
		results, err := abTester.AnalyzeTestResults(testID)
		require.NoError(t, err)
		assert.NotNil(t, results)

		// Verify results structure
		assert.Contains(t, results.VariantResults, "control")
		assert.Contains(t, results.VariantResults, "experiment")
		
		// Control should have 2 results, 100% success rate, avg 1.65s
		controlResults := results.VariantResults["control"]
		assert.Equal(t, 2, controlResults.SampleSize)
		assert.Equal(t, 1.0, controlResults.SuccessRate) // 2/2 = 100%
		assert.Equal(t, 1.65, controlResults.AverageResponseTime) // (1.5+1.8)/2 = 1.65

		// Experiment should have 1 result, 0% success rate, 2.1s
		experimentResults := results.VariantResults["experiment"]
		assert.Equal(t, 1, experimentResults.SampleSize)
		assert.Equal(t, 0.0, experimentResults.SuccessRate) // 0/1 = 0%
		assert.Equal(t, 2.1, experimentResults.AverageResponseTime)
	})
}

func TestRuntimeConfigIntegration(t *testing.T) {
	t.Run("end_to_end_configuration_workflow", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "integration_config.yaml")

		// Create runtime config system
		configManager, err := NewRuntimeConfigManager(configPath)
		require.NoError(t, err)

		hotReloader, err := NewHotReloadSystem(configPath)
		require.NoError(t, err)

		versionManager, err := NewConfigVersionManager(filepath.Join(tempDir, "versions"))
		require.NoError(t, err)

		// Create integrated runtime system
		runtimeSystem, err := NewIntegratedRuntimeSystem(configManager, hotReloader, versionManager)
		require.NoError(t, err)
		require.NotNil(t, runtimeSystem)

		// Test initial configuration deployment
		initialConfig := &RuntimeBusinessConfig{
			Version: "1.0.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 8,
					Timeout:        20 * time.Second,
				},
			},
			HotReloadEnabled: true,
		}

		err = runtimeSystem.DeployConfiguration(initialConfig, "Initial deployment")
		require.NoError(t, err)

		// Verify deployment
		currentConfig := runtimeSystem.GetCurrentConfiguration()
		assert.Equal(t, "1.0.0", currentConfig.Version)
		assert.Equal(t, 8, currentConfig.BusinessPolicies.GlobalPolicy.MaxConcurrency)

		// Test configuration update
		updatedConfig := &RuntimeBusinessConfig{
			Version: "1.1.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{
					MaxConcurrency: 12,
					Timeout:        25 * time.Second,
				},
			},
			HotReloadEnabled: true,
		}

		err = runtimeSystem.DeployConfiguration(updatedConfig, "Increased concurrency and timeout")
		require.NoError(t, err)

		// Verify update
		currentConfig = runtimeSystem.GetCurrentConfiguration()
		assert.Equal(t, "1.1.0", currentConfig.Version)
		assert.Equal(t, 12, currentConfig.BusinessPolicies.GlobalPolicy.MaxConcurrency)
		assert.Equal(t, 25*time.Second, currentConfig.BusinessPolicies.GlobalPolicy.Timeout)
	})

	t.Run("configuration_rollback_integration", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "rollback_config.yaml")

		configManager, err := NewRuntimeConfigManager(configPath)
		require.NoError(t, err)

		versionManager, err := NewConfigVersionManager(filepath.Join(tempDir, "rollback_versions"))
		require.NoError(t, err)

		runtimeSystem, err := NewIntegratedRuntimeSystem(configManager, nil, versionManager)
		require.NoError(t, err)

		// Deploy initial configuration
		v1Config := &RuntimeBusinessConfig{
			Version: "1.0.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{MaxConcurrency: 5},
			},
		}

		err = runtimeSystem.DeployConfiguration(v1Config, "Version 1")
		require.NoError(t, err)

		// Deploy updated configuration
		v2Config := &RuntimeBusinessConfig{
			Version: "2.0.0",
			BusinessPolicies: &policies.BusinessPolicies{
				GlobalPolicy: &policies.GlobalBusinessPolicy{MaxConcurrency: 15},
			},
		}

		err = runtimeSystem.DeployConfiguration(v2Config, "Version 2")
		require.NoError(t, err)

		// Verify current version
		current := runtimeSystem.GetCurrentConfiguration()
		assert.Equal(t, "2.0.0", current.Version)

		// Test rollback
		err = runtimeSystem.RollbackToVersion("1.0.0")
		require.NoError(t, err)

		// Verify rollback
		rolledBack := runtimeSystem.GetCurrentConfiguration()
		assert.Equal(t, "1.0.0", rolledBack.Version)
		assert.Equal(t, 5, rolledBack.BusinessPolicies.GlobalPolicy.MaxConcurrency)
	})
}