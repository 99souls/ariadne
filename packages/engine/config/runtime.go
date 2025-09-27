package config

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ariadne/packages/engine/business/policies"

	"gopkg.in/yaml.v3"
	"github.com/fsnotify/fsnotify"
)

// RuntimeBusinessConfig represents a complete runtime configuration
type RuntimeBusinessConfig struct {
	Version          string                     `yaml:"version" json:"version"`
	UpdatedAt        time.Time                  `yaml:"updated_at" json:"updated_at"`
	BusinessPolicies *policies.BusinessPolicies `yaml:"business_policies" json:"business_policies"`
	HotReloadEnabled bool                       `yaml:"hot_reload_enabled" json:"hot_reload_enabled"`
	ConfigSource     string                     `yaml:"config_source,omitempty" json:"config_source,omitempty"`
	Checksum         string                     `yaml:"checksum,omitempty" json:"checksum,omitempty"`
}

// RuntimeConfigManager manages runtime configuration updates
type RuntimeConfigManager struct {
	configPath    string
	currentConfig *RuntimeBusinessConfig
	mutex         sync.RWMutex
	validators    []ConfigValidator
}

// ConfigValidator validates configuration before applying updates
type ConfigValidator interface {
	Validate(config *RuntimeBusinessConfig) error
}

// HotReloadSystem manages file system watching and configuration hot-reloading
type HotReloadSystem struct {
	configPath string
	watcher    *fsnotify.Watcher
	isWatching bool
	mutex      sync.Mutex
}

// ConfigChange represents a detected configuration change
type ConfigChange struct {
	*RuntimeBusinessConfig
	ChangeType  string    `json:"change_type"`
	ChangedAt   time.Time `json:"changed_at"`
	PreviousChecksum string `json:"previous_checksum"`
}

// ConfigVersionManager manages configuration version history and rollbacks
type ConfigVersionManager struct {
	versionsDir string
	mutex       sync.RWMutex
}

// ConfigVersion represents a stored configuration version
type ConfigVersion struct {
	Version           string                     `json:"version"`
	Config            *RuntimeBusinessConfig     `json:"config"`
	SavedAt           time.Time                  `json:"saved_at"`
	ChangeDescription string                     `json:"change_description"`
	PreviousVersion   string                     `json:"previous_version,omitempty"`
}

// ABTestingFramework manages A/B testing for configuration changes
type ABTestingFramework struct {
	testsDir string
	mutex    sync.RWMutex
}

// ABTest represents an A/B test configuration
type ABTest struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	ControlConfig    *RuntimeBusinessConfig     `json:"control_config"`
	ExperimentConfig *RuntimeBusinessConfig     `json:"experiment_config"`
	TrafficSplit     float64                    `json:"traffic_split"`
	CreatedAt        time.Time                  `json:"created_at"`
	Status           string                     `json:"status"`
}

// ABTestResult represents results from an A/B test
type ABTestResult struct {
	TestID         string                        `json:"test_id"`
	VariantResults map[string]*VariantResult     `json:"variant_results"`
	StatisticalSignificance bool               `json:"statistical_significance"`
	Recommendation string                       `json:"recommendation"`
	AnalyzedAt     time.Time                    `json:"analyzed_at"`
}

// VariantResult represents results for a specific variant
type VariantResult struct {
	VariantName         string  `json:"variant_name"`
	SampleSize          int     `json:"sample_size"`
	SuccessRate         float64 `json:"success_rate"`
	AverageResponseTime float64 `json:"average_response_time"`
	ErrorRate           float64 `json:"error_rate"`
}

// TestResultRecord represents a single test result record
type TestResultRecord struct {
	TestID       string    `json:"test_id"`
	UserID       string    `json:"user_id"`
	Variant      string    `json:"variant"`
	Success      bool      `json:"success"`
	ResponseTime float64   `json:"response_time"`
	RecordedAt   time.Time `json:"recorded_at"`
}

// IntegratedRuntimeSystem combines all runtime configuration management components
type IntegratedRuntimeSystem struct {
	configManager  *RuntimeConfigManager
	hotReloader    *HotReloadSystem
	versionManager *ConfigVersionManager
	abTester       *ABTestingFramework
	mutex          sync.RWMutex
}

// NewRuntimeConfigManager creates a new runtime configuration manager
func NewRuntimeConfigManager(configPath string) (*RuntimeConfigManager, error) {
	manager := &RuntimeConfigManager{
		configPath:    configPath,
		currentConfig: &RuntimeBusinessConfig{},
		validators:    make([]ConfigValidator, 0),
	}

	// Add default validator
	manager.AddValidator(&defaultConfigValidator{})

	return manager, nil
}

// AddValidator adds a configuration validator
func (rcm *RuntimeConfigManager) AddValidator(validator ConfigValidator) {
	rcm.mutex.Lock()
	defer rcm.mutex.Unlock()
	rcm.validators = append(rcm.validators, validator)
}

// LoadConfiguration loads configuration from file
func (rcm *RuntimeConfigManager) LoadConfiguration() error {
	rcm.mutex.Lock()
	defer rcm.mutex.Unlock()

	// If file doesn't exist, use empty config
	if _, err := os.Stat(rcm.configPath); os.IsNotExist(err) {
		rcm.currentConfig = &RuntimeBusinessConfig{
			UpdatedAt:        time.Now(),
			BusinessPolicies: &policies.BusinessPolicies{},
		}
		return nil
	}

	data, err := os.ReadFile(rcm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config RuntimeBusinessConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	rcm.currentConfig = &config
	return nil
}

// UpdateConfiguration updates the current configuration
func (rcm *RuntimeConfigManager) UpdateConfiguration(config *RuntimeBusinessConfig) error {
	rcm.mutex.Lock()
	defer rcm.mutex.Unlock()

	// Validate configuration
	if err := rcm.validateConfiguration(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Update timestamp and checksum
	config.UpdatedAt = time.Now()
	config.Checksum = rcm.calculateChecksum(config)

	// Update current configuration
	rcm.currentConfig = config

	// Save to file
	return rcm.saveConfigurationToFile(config)
}

// GetCurrentConfig returns the current configuration (read-only copy)
func (rcm *RuntimeConfigManager) GetCurrentConfig() *RuntimeBusinessConfig {
	rcm.mutex.RLock()
	defer rcm.mutex.RUnlock()

	// Return a copy to prevent external modifications
	configCopy := *rcm.currentConfig
	return &configCopy
}

// ValidateConfiguration validates a configuration without applying it
func (rcm *RuntimeConfigManager) ValidateConfiguration(config *RuntimeBusinessConfig) error {
	rcm.mutex.RLock()
	defer rcm.mutex.RUnlock()
	return rcm.validateConfiguration(config)
}

func (rcm *RuntimeConfigManager) validateConfiguration(config *RuntimeBusinessConfig) error {
	for _, validator := range rcm.validators {
		if err := validator.Validate(config); err != nil {
			return err
		}
	}
	return nil
}

func (rcm *RuntimeConfigManager) saveConfigurationToFile(config *RuntimeBusinessConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(rcm.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return os.WriteFile(rcm.configPath, data, 0644)
}

func (rcm *RuntimeConfigManager) calculateChecksum(config *RuntimeBusinessConfig) string {
	// Create a copy without checksum for calculation
	configForHash := *config
	configForHash.Checksum = ""
	
	data, _ := json.Marshal(configForHash)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// NewHotReloadSystem creates a new hot reload system
func NewHotReloadSystem(configPath string) (*HotReloadSystem, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &HotReloadSystem{
		configPath: configPath,
		watcher:    watcher,
		isWatching: false,
	}, nil
}

// WatchConfigChanges starts watching for configuration file changes
func (hrs *HotReloadSystem) WatchConfigChanges(ctx context.Context) (<-chan *ConfigChange, <-chan error) {
	changesChan := make(chan *ConfigChange, 10)
	errorsChan := make(chan error, 10)

	hrs.mutex.Lock()
	if hrs.isWatching {
		hrs.mutex.Unlock()
		close(changesChan)
		close(errorsChan)
		return changesChan, errorsChan
	}

	// Add the directory to watcher (watching directory is more reliable than watching file)
	configDir := filepath.Dir(hrs.configPath)
	if err := hrs.watcher.Add(configDir); err != nil {
		hrs.mutex.Unlock()
		errorsChan <- fmt.Errorf("failed to watch directory %s: %w", configDir, err)
		close(changesChan)
		close(errorsChan)
		return changesChan, errorsChan
	}

	hrs.isWatching = true
	hrs.mutex.Unlock()

	go func() {
		defer close(changesChan)
		defer close(errorsChan)

		var lastConfig *RuntimeBusinessConfig

		for {
			select {
			case event, ok := <-hrs.watcher.Events:
				if !ok {
					return
				}

				// Only process events for our specific config file
				if event.Name != hrs.configPath {
					continue
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					// Read the updated configuration
					newConfig, err := hrs.loadConfigFromFile()
					if err != nil {
						errorsChan <- err
						continue
					}

					// Check if configuration actually changed
					if hrs.DetectChanges(lastConfig, newConfig) {
						change := &ConfigChange{
							RuntimeBusinessConfig: newConfig,
							ChangeType:            "file_modified",
							ChangedAt:             time.Now(),
						}

						if lastConfig != nil {
							change.PreviousChecksum = lastConfig.Checksum
						}

						changesChan <- change
						lastConfig = newConfig
					}
				}

			case err, ok := <-hrs.watcher.Errors:
				if !ok {
					return
				}
				errorsChan <- err

			case <-ctx.Done():
				return
			}
		}
	}()

	return changesChan, errorsChan
}

// StopWatching stops the file system watcher
func (hrs *HotReloadSystem) StopWatching() error {
	hrs.mutex.Lock()
	defer hrs.mutex.Unlock()

	if hrs.isWatching {
		hrs.isWatching = false
		return hrs.watcher.Close()
	}

	return nil
}

// DetectChanges compares two configurations and returns true if they differ
func (hrs *HotReloadSystem) DetectChanges(oldConfig, newConfig *RuntimeBusinessConfig) bool {
	if oldConfig == nil && newConfig == nil {
		return false
	}

	if oldConfig == nil || newConfig == nil {
		return true
	}

	// Compare checksums if available
	if oldConfig.Checksum != "" && newConfig.Checksum != "" {
		return oldConfig.Checksum != newConfig.Checksum
	}

	// Compare JSON representations
	oldData, _ := json.Marshal(oldConfig)
	newData, _ := json.Marshal(newConfig)
	
	return string(oldData) != string(newData)
}

func (hrs *HotReloadSystem) loadConfigFromFile() (*RuntimeBusinessConfig, error) {
	if _, err := os.Stat(hrs.configPath); os.IsNotExist(err) {
		return &RuntimeBusinessConfig{}, nil
	}

	data, err := os.ReadFile(hrs.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config RuntimeBusinessConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// NewConfigVersionManager creates a new configuration version manager
func NewConfigVersionManager(versionsDir string) (*ConfigVersionManager, error) {
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create versions directory: %w", err)
	}

	return &ConfigVersionManager{
		versionsDir: versionsDir,
	}, nil
}

// SaveVersion saves a configuration version with description
func (cvm *ConfigVersionManager) SaveVersion(config *RuntimeBusinessConfig, changeDescription string, args ...interface{}) error {
	cvm.mutex.Lock()
	defer cvm.mutex.Unlock()

	formattedDescription := fmt.Sprintf(changeDescription, args...)

	version := &ConfigVersion{
		Version:           config.Version,
		Config:            config,
		SavedAt:           time.Now(),
		ChangeDescription: formattedDescription,
	}

	versionFile := filepath.Join(cvm.versionsDir, fmt.Sprintf("%s.json", config.Version))
	
	data, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	return os.WriteFile(versionFile, data, 0644)
}

// GetVersionHistory returns the version history
func (cvm *ConfigVersionManager) GetVersionHistory() ([]*ConfigVersion, error) {
	cvm.mutex.RLock()
	defer cvm.mutex.RUnlock()

	files, err := os.ReadDir(cvm.versionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read versions directory: %w", err)
	}

	var versions []*ConfigVersion

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			versionFile := filepath.Join(cvm.versionsDir, file.Name())
			
			data, err := os.ReadFile(versionFile)
			if err != nil {
				continue // Skip problematic files
			}

			var version ConfigVersion
			if err := json.Unmarshal(data, &version); err != nil {
				continue // Skip invalid files
			}

			versions = append(versions, &version)
		}
	}

	return versions, nil
}

// RollbackToVersion rolls back to a specific version
func (cvm *ConfigVersionManager) RollbackToVersion(targetVersion string) (*RuntimeBusinessConfig, error) {
	cvm.mutex.RLock()
	defer cvm.mutex.RUnlock()

	versionFile := filepath.Join(cvm.versionsDir, fmt.Sprintf("%s.json", targetVersion))

	if _, err := os.Stat(versionFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("version not found: %s", targetVersion)
	}

	data, err := os.ReadFile(versionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read version file: %w", err)
	}

	var version ConfigVersion
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, fmt.Errorf("failed to parse version file: %w", err)
	}

	return version.Config, nil
}

// NewABTestingFramework creates a new A/B testing framework
func NewABTestingFramework(testsDir string) (*ABTestingFramework, error) {
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tests directory: %w", err)
	}

	return &ABTestingFramework{
		testsDir: testsDir,
	}, nil
}

// CreateABTest creates a new A/B test
func (abt *ABTestingFramework) CreateABTest(name string, controlConfig, experimentConfig *RuntimeBusinessConfig, trafficSplit float64) (string, error) {
	abt.mutex.Lock()
	defer abt.mutex.Unlock()

	testID := fmt.Sprintf("test_%s_%d", name, time.Now().Unix())

	test := &ABTest{
		ID:               testID,
		Name:             name,
		ControlConfig:    controlConfig,
		ExperimentConfig: experimentConfig,
		TrafficSplit:     trafficSplit,
		CreatedAt:        time.Now(),
		Status:           "active",
	}

	testFile := filepath.Join(abt.testsDir, fmt.Sprintf("%s.json", testID))

	data, err := json.MarshalIndent(test, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal test: %w", err)
	}

	err = os.WriteFile(testFile, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save test: %w", err)
	}

	return testID, nil
}

// GetConfigForUser returns the appropriate configuration for a user based on A/B test
func (abt *ABTestingFramework) GetConfigForUser(userID, testID string) *RuntimeBusinessConfig {
	abt.mutex.RLock()
	defer abt.mutex.RUnlock()

	test := abt.loadTest(testID)
	if test == nil {
		return nil
	}

	// Use consistent hash-based assignment
	hash := fnv.New32a()
	hash.Write([]byte(userID + testID))
	userHash := float64(hash.Sum32()) / float64(1<<32)

	if userHash < test.TrafficSplit {
		return test.ExperimentConfig
	}

	return test.ControlConfig
}

// RecordTestResult records a result from an A/B test
func (abt *ABTestingFramework) RecordTestResult(testID, userID, variant string, success bool, responseTime float64) error {
	abt.mutex.Lock()
	defer abt.mutex.Unlock()

	result := &TestResultRecord{
		TestID:       testID,
		UserID:       userID,
		Variant:      variant,
		Success:      success,
		ResponseTime: responseTime,
		RecordedAt:   time.Now(),
	}

	resultsFile := filepath.Join(abt.testsDir, fmt.Sprintf("%s_results.json", testID))

	var results []*TestResultRecord

	// Load existing results
	if data, err := os.ReadFile(resultsFile); err == nil {
		json.Unmarshal(data, &results)
	}

	// Append new result
	results = append(results, result)

	// Save updated results
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	return os.WriteFile(resultsFile, data, 0644)
}

// AnalyzeTestResults analyzes A/B test results
func (abt *ABTestingFramework) AnalyzeTestResults(testID string) (*ABTestResult, error) {
	abt.mutex.RLock()
	defer abt.mutex.RUnlock()

	resultsFile := filepath.Join(abt.testsDir, fmt.Sprintf("%s_results.json", testID))

	data, err := os.ReadFile(resultsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read results file: %w", err)
	}

	var results []*TestResultRecord
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}

	// Group results by variant
	variantStatsMap := make(map[string]*variantStats)
	
	for _, result := range results {
		if _, exists := variantStatsMap[result.Variant]; !exists {
			variantStatsMap[result.Variant] = &variantStats{}
		}

		stats := variantStatsMap[result.Variant]
		stats.sampleSize++
		stats.totalResponseTime += result.ResponseTime

		if result.Success {
			stats.successCount++
		}
	}

	// Calculate final metrics
	variantResults := make(map[string]*VariantResult)
	for variant, stats := range variantStatsMap {
		successRate := float64(stats.successCount) / float64(stats.sampleSize)
		avgResponseTime := stats.totalResponseTime / float64(stats.sampleSize)
		errorRate := 1.0 - successRate

		variantResults[variant] = &VariantResult{
			VariantName:         variant,
			SampleSize:          stats.sampleSize,
			SuccessRate:         successRate,
			AverageResponseTime: avgResponseTime,
			ErrorRate:           errorRate,
		}
	}

	return &ABTestResult{
		TestID:                  testID,
		VariantResults:          variantResults,
		StatisticalSignificance: len(results) >= 100, // Simple threshold
		Recommendation:          abt.generateRecommendation(variantResults),
		AnalyzedAt:             time.Now(),
	}, nil
}

func (abt *ABTestingFramework) loadTest(testID string) *ABTest {
	testFile := filepath.Join(abt.testsDir, fmt.Sprintf("%s.json", testID))

	data, err := os.ReadFile(testFile)
	if err != nil {
		return nil
	}

	var test ABTest
	if err := json.Unmarshal(data, &test); err != nil {
		return nil
	}

	return &test
}

func (abt *ABTestingFramework) generateRecommendation(results map[string]*VariantResult) string {
	if len(results) < 2 {
		return "Insufficient data for recommendation"
	}

	var bestVariant string
	var bestSuccessRate float64

	for variant, result := range results {
		if result.SuccessRate > bestSuccessRate {
			bestSuccessRate = result.SuccessRate
			bestVariant = variant
		}
	}

	return fmt.Sprintf("Recommend using variant '%s' with %.1f%% success rate", bestVariant, bestSuccessRate*100)
}

type variantStats struct {
	sampleSize        int
	successCount      int
	totalResponseTime float64
}

// NewIntegratedRuntimeSystem creates a new integrated runtime system
func NewIntegratedRuntimeSystem(configManager *RuntimeConfigManager, hotReloader *HotReloadSystem, versionManager *ConfigVersionManager) (*IntegratedRuntimeSystem, error) {
	return &IntegratedRuntimeSystem{
		configManager:  configManager,
		hotReloader:    hotReloader,
		versionManager: versionManager,
	}, nil
}

// DeployConfiguration deploys a new configuration with versioning
func (irs *IntegratedRuntimeSystem) DeployConfiguration(config *RuntimeBusinessConfig, changeDescription string) error {
	irs.mutex.Lock()
	defer irs.mutex.Unlock()

	// Save version if version manager available
	if irs.versionManager != nil {
		if err := irs.versionManager.SaveVersion(config, "%s", changeDescription); err != nil {
			return fmt.Errorf("failed to save version: %w", err)
		}
	}

	// Update configuration
	if err := irs.configManager.UpdateConfiguration(config); err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	return nil
}

// GetCurrentConfiguration returns the current configuration
func (irs *IntegratedRuntimeSystem) GetCurrentConfiguration() *RuntimeBusinessConfig {
	irs.mutex.RLock()
	defer irs.mutex.RUnlock()
	
	return irs.configManager.GetCurrentConfig()
}

// RollbackToVersion rolls back to a specific configuration version
func (irs *IntegratedRuntimeSystem) RollbackToVersion(version string) error {
	irs.mutex.Lock()
	defer irs.mutex.Unlock()

	if irs.versionManager == nil {
		return fmt.Errorf("version manager not available")
	}

	// Get the target version configuration
	targetConfig, err := irs.versionManager.RollbackToVersion(version)
	if err != nil {
		return fmt.Errorf("failed to get target version: %w", err)
	}

	// Deploy the rollback configuration
	if err := irs.configManager.UpdateConfiguration(targetConfig); err != nil {
		return fmt.Errorf("failed to apply rollback configuration: %w", err)
	}

	return nil
}

// defaultConfigValidator provides basic configuration validation
type defaultConfigValidator struct{}

func (dcv *defaultConfigValidator) Validate(config *RuntimeBusinessConfig) error {
	if config.BusinessPolicies != nil && config.BusinessPolicies.GlobalPolicy != nil {
		globalPolicy := config.BusinessPolicies.GlobalPolicy

		if globalPolicy.MaxConcurrency < 0 {
			return fmt.Errorf("global policy max concurrency cannot be negative")
		}

		if globalPolicy.Timeout < 0 {
			return fmt.Errorf("global policy timeout cannot be negative")
		}
	}

	return nil
}