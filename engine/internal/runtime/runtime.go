package runtime

// Internalized runtime configuration & A/B testing framework.
// All types previously exported under config/runtime.go have been relocated here.
// No public re-export is provided for Wave 4; if external need emerges we will
// re-introduce a minimal stable facade.

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

	"github.com/99souls/ariadne/engine/internal/business/policies"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type RuntimeBusinessConfig struct {
	Version          string
	UpdatedAt        time.Time
	BusinessPolicies *policies.BusinessPolicies
	HotReloadEnabled bool
	ConfigSource     string
	Checksum         string
}

type RuntimeConfigManager struct {
	configPath    string
	currentConfig *RuntimeBusinessConfig
	mutex         sync.RWMutex
	validators    []ConfigValidator
}

type ConfigValidator interface {
	Validate(config *RuntimeBusinessConfig) error
}

type HotReloadSystem struct {
	configPath string
	watcher    *fsnotify.Watcher
	isWatching bool
	mutex      sync.Mutex
}

type ConfigChange struct {
	*RuntimeBusinessConfig
	ChangeType       string
	ChangedAt        time.Time
	PreviousChecksum string
}

type ConfigVersionManager struct {
	versionsDir string
	mutex       sync.RWMutex
}

type ConfigVersion struct {
	Version           string
	Config            *RuntimeBusinessConfig
	SavedAt           time.Time
	ChangeDescription string
	PreviousVersion   string
}

type ABTestingFramework struct {
	testsDir string
	mutex    sync.RWMutex
}

type ABTest struct {
	ID               string
	Name             string
	ControlConfig    *RuntimeBusinessConfig
	ExperimentConfig *RuntimeBusinessConfig
	TrafficSplit     float64
	CreatedAt        time.Time
	Status           string
}

type ABTestResult struct {
	TestID                  string
	VariantResults          map[string]*VariantResult
	StatisticalSignificance bool
	Recommendation          string
	AnalyzedAt              time.Time
}

type VariantResult struct {
	VariantName         string
	SampleSize          int
	SuccessRate         float64
	AverageResponseTime float64
	ErrorRate           float64
}

type TestResultRecord struct {
	TestID       string
	UserID       string
	Variant      string
	Success      bool
	ResponseTime float64
	RecordedAt   time.Time
}

type IntegratedRuntimeSystem struct {
	configManager  *RuntimeConfigManager
	hotReloader    *HotReloadSystem
	versionManager *ConfigVersionManager
	mutex          sync.RWMutex
}

func NewRuntimeConfigManager(configPath string) (*RuntimeConfigManager, error) {
	manager := &RuntimeConfigManager{configPath: configPath, currentConfig: &RuntimeBusinessConfig{}, validators: make([]ConfigValidator, 0)}
	manager.AddValidator(&defaultConfigValidator{})
	return manager, nil
}

func (rcm *RuntimeConfigManager) AddValidator(validator ConfigValidator) {
	rcm.mutex.Lock()
	defer rcm.mutex.Unlock()
	rcm.validators = append(rcm.validators, validator)
}

func (rcm *RuntimeConfigManager) LoadConfiguration() error {
	rcm.mutex.Lock()
	defer rcm.mutex.Unlock()
	if _, err := os.Stat(rcm.configPath); os.IsNotExist(err) {
		rcm.currentConfig = &RuntimeBusinessConfig{UpdatedAt: time.Now(), BusinessPolicies: &policies.BusinessPolicies{}}
		return nil
	}
	data, err := os.ReadFile(rcm.configPath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	var cfg RuntimeBusinessConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	rcm.currentConfig = &cfg
	return nil
}

func (rcm *RuntimeConfigManager) UpdateConfiguration(cfg *RuntimeBusinessConfig) error {
	rcm.mutex.Lock()
	defer rcm.mutex.Unlock()
	if err := rcm.validateConfiguration(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	cfg.UpdatedAt = time.Now()
	cfg.Checksum = rcm.calculateChecksum(cfg)
	rcm.currentConfig = cfg
	return rcm.saveConfigurationToFile(cfg)
}

func (rcm *RuntimeConfigManager) GetCurrentConfig() *RuntimeBusinessConfig {
	rcm.mutex.RLock()
	defer rcm.mutex.RUnlock()
	cpy := *rcm.currentConfig
	return &cpy
}
func (rcm *RuntimeConfigManager) ValidateConfiguration(cfg *RuntimeBusinessConfig) error {
	rcm.mutex.RLock()
	defer rcm.mutex.RUnlock()
	return rcm.validateConfiguration(cfg)
}
func (rcm *RuntimeConfigManager) validateConfiguration(cfg *RuntimeBusinessConfig) error {
	for _, v := range rcm.validators {
		if err := v.Validate(cfg); err != nil {
			return err
		}
	}
	return nil
}

func (rcm *RuntimeConfigManager) saveConfigurationToFile(cfg *RuntimeBusinessConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(rcm.configPath), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return os.WriteFile(rcm.configPath, data, 0644)
}

func (rcm *RuntimeConfigManager) calculateChecksum(cfg *RuntimeBusinessConfig) string {
	cpy := *cfg
	cpy.Checksum = ""
	data, _ := json.Marshal(cpy)
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func NewHotReloadSystem(configPath string) (*HotReloadSystem, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create file watcher: %w", err)
	}
	return &HotReloadSystem{configPath: configPath, watcher: watcher}, nil
}

func (hrs *HotReloadSystem) WatchConfigChanges(ctx context.Context) (<-chan *ConfigChange, <-chan error) {
	changes := make(chan *ConfigChange, 10)
	errs := make(chan error, 10)
	hrs.mutex.Lock()
	if hrs.isWatching {
		hrs.mutex.Unlock()
		close(changes)
		close(errs)
		return changes, errs
	}
	configDir := filepath.Dir(hrs.configPath)
	if err := hrs.watcher.Add(configDir); err != nil {
		hrs.mutex.Unlock()
		errs <- fmt.Errorf("watch dir %s: %w", configDir, err)
		close(changes)
		close(errs)
		return changes, errs
	}
	hrs.isWatching = true
	hrs.mutex.Unlock()

	go func() {
		defer close(changes)
		defer close(errs)
		var last *RuntimeBusinessConfig
		for {
			select {
			case e, ok := <-hrs.watcher.Events:
				if !ok {
					return
				}
				if e.Name != hrs.configPath {
					continue
				}
				if e.Op&fsnotify.Write == fsnotify.Write {
					nc, err := hrs.loadConfigFromFile()
					if err != nil {
						errs <- err
						continue
					}
					if hrs.DetectChanges(last, nc) {
						ch := &ConfigChange{RuntimeBusinessConfig: nc, ChangeType: "file_modified", ChangedAt: time.Now()}
						if last != nil {
							ch.PreviousChecksum = last.Checksum
						}
						changes <- ch
						last = nc
					}
				}
			case err, ok := <-hrs.watcher.Errors:
				if !ok {
					return
				}
				errs <- err
			case <-ctx.Done():
				return
			}
		}
	}()
	return changes, errs
}

func (hrs *HotReloadSystem) StopWatching() error {
	hrs.mutex.Lock()
	defer hrs.mutex.Unlock()
	if hrs.isWatching {
		hrs.isWatching = false
		return hrs.watcher.Close()
	}
	return nil
}
func (hrs *HotReloadSystem) DetectChanges(oldC, newC *RuntimeBusinessConfig) bool {
	if oldC == nil && newC == nil {
		return false
	}
	if oldC == nil || newC == nil {
		return true
	}
	if oldC.Checksum != "" && newC.Checksum != "" {
		return oldC.Checksum != newC.Checksum
	}
	od, _ := json.Marshal(oldC)
	nd, _ := json.Marshal(newC)
	return string(od) != string(nd)
}
func (hrs *HotReloadSystem) loadConfigFromFile() (*RuntimeBusinessConfig, error) {
	if _, err := os.Stat(hrs.configPath); os.IsNotExist(err) {
		return &RuntimeBusinessConfig{}, nil
	}
	data, err := os.ReadFile(hrs.configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	var cfg RuntimeBusinessConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	return &cfg, nil
}

func NewConfigVersionManager(dir string) (*ConfigVersionManager, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create versions dir: %w", err)
	}
	return &ConfigVersionManager{versionsDir: dir}, nil
}
func (cvm *ConfigVersionManager) SaveVersion(cfg *RuntimeBusinessConfig, changeDescription string, args ...interface{}) error {
	cvm.mutex.Lock()
	defer cvm.mutex.Unlock()
	desc := fmt.Sprintf(changeDescription, args...)
	v := &ConfigVersion{Version: cfg.Version, Config: cfg, SavedAt: time.Now(), ChangeDescription: desc}
	vf := filepath.Join(cvm.versionsDir, fmt.Sprintf("%s.json", cfg.Version))
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal version: %w", err)
	}
	return os.WriteFile(vf, data, 0644)
}
func (cvm *ConfigVersionManager) GetVersionHistory() ([]*ConfigVersion, error) {
	cvm.mutex.RLock()
	defer cvm.mutex.RUnlock()
	files, err := os.ReadDir(cvm.versionsDir)
	if err != nil {
		return nil, fmt.Errorf("read versions dir: %w", err)
	}
	var versions []*ConfigVersion
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		vf := filepath.Join(cvm.versionsDir, f.Name())
		data, err := os.ReadFile(vf)
		if err != nil {
			continue
		}
		var v ConfigVersion
		if err := json.Unmarshal(data, &v); err != nil {
			continue
		}
		versions = append(versions, &v)
	}
	return versions, nil
}
func (cvm *ConfigVersionManager) RollbackToVersion(v string) (*RuntimeBusinessConfig, error) {
	cvm.mutex.RLock()
	defer cvm.mutex.RUnlock()
	vf := filepath.Join(cvm.versionsDir, fmt.Sprintf("%s.json", v))
	if _, err := os.Stat(vf); os.IsNotExist(err) {
		return nil, fmt.Errorf("version not found: %s", v)
	}
	data, err := os.ReadFile(vf)
	if err != nil {
		return nil, fmt.Errorf("read version file: %w", err)
	}
	var ver ConfigVersion
	if err := json.Unmarshal(data, &ver); err != nil {
		return nil, fmt.Errorf("parse version file: %w", err)
	}
	return ver.Config, nil
}

func NewABTestingFramework(dir string) (*ABTestingFramework, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create tests dir: %w", err)
	}
	return &ABTestingFramework{testsDir: dir}, nil
}
func (abt *ABTestingFramework) CreateABTest(name string, control, experiment *RuntimeBusinessConfig, split float64) (string, error) {
	abt.mutex.Lock()
	defer abt.mutex.Unlock()
	id := fmt.Sprintf("test_%s_%d", name, time.Now().Unix())
	test := &ABTest{ID: id, Name: name, ControlConfig: control, ExperimentConfig: experiment, TrafficSplit: split, CreatedAt: time.Now(), Status: "active"}
	tf := filepath.Join(abt.testsDir, fmt.Sprintf("%s.json", id))
	data, err := json.MarshalIndent(test, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal test: %w", err)
	}
	if err := os.WriteFile(tf, data, 0644); err != nil {
		return "", fmt.Errorf("save test: %w", err)
	}
	return id, nil
}
func (abt *ABTestingFramework) GetConfigForUser(userID, testID string) *RuntimeBusinessConfig {
	abt.mutex.RLock()
	defer abt.mutex.RUnlock()
	test := abt.loadTest(testID)
	if test == nil {
		return nil
	}
	h := fnv.New32a()
	h.Write([]byte(userID + testID))
	uh := float64(h.Sum32()) / float64(1<<32)
	if uh < test.TrafficSplit {
		return test.ExperimentConfig
	}
	return test.ControlConfig
}
func (abt *ABTestingFramework) RecordTestResult(testID, userID, variant string, success bool, responseTime float64) error {
	abt.mutex.Lock()
	defer abt.mutex.Unlock()
	rec := &TestResultRecord{TestID: testID, UserID: userID, Variant: variant, Success: success, ResponseTime: responseTime, RecordedAt: time.Now()}
	rf := filepath.Join(abt.testsDir, fmt.Sprintf("%s_results.json", testID))
	var all []*TestResultRecord
	if data, err := os.ReadFile(rf); err == nil {
		_ = json.Unmarshal(data, &all)
	}
	all = append(all, rec)
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}
	return os.WriteFile(rf, data, 0644)
}
func (abt *ABTestingFramework) AnalyzeTestResults(testID string) (*ABTestResult, error) {
	abt.mutex.RLock()
	defer abt.mutex.RUnlock()
	rf := filepath.Join(abt.testsDir, fmt.Sprintf("%s_results.json", testID))
	data, err := os.ReadFile(rf)
	if err != nil {
		return nil, fmt.Errorf("read results: %w", err)
	}
	var recs []*TestResultRecord
	if err := json.Unmarshal(data, &recs); err != nil {
		return nil, fmt.Errorf("parse results: %w", err)
	}
	vs := map[string]*variantStats{}
	for _, r := range recs {
		if _, ok := vs[r.Variant]; !ok {
			vs[r.Variant] = &variantStats{}
		}
		st := vs[r.Variant]
		st.sampleSize++
		st.totalResponseTime += r.ResponseTime
		if r.Success {
			st.successCount++
		}
	}
	vr := map[string]*VariantResult{}
	for variant, st := range vs {
		successRate := float64(st.successCount) / float64(st.sampleSize)
		vr[variant] = &VariantResult{VariantName: variant, SampleSize: st.sampleSize, SuccessRate: successRate, AverageResponseTime: st.totalResponseTime / float64(st.sampleSize), ErrorRate: 1 - successRate}
	}
	return &ABTestResult{TestID: testID, VariantResults: vr, StatisticalSignificance: len(recs) >= 100, Recommendation: abt.generateRecommendation(vr), AnalyzedAt: time.Now()}, nil
}
func (abt *ABTestingFramework) loadTest(testID string) *ABTest {
	tf := filepath.Join(abt.testsDir, fmt.Sprintf("%s.json", testID))
	data, err := os.ReadFile(tf)
	if err != nil {
		return nil
	}
	var t ABTest
	if err := json.Unmarshal(data, &t); err != nil {
		return nil
	}
	return &t
}
func (abt *ABTestingFramework) generateRecommendation(results map[string]*VariantResult) string {
	if len(results) < 2 {
		return "Insufficient data for recommendation"
	}
	var best string
	var bestRate float64
	for v, r := range results {
		if r.SuccessRate > bestRate {
			bestRate = r.SuccessRate
			best = v
		}
	}
	return fmt.Sprintf("Recommend using variant '%s' with %.1f%% success rate", best, bestRate*100)
}

type variantStats struct {
	sampleSize        int
	successCount      int
	totalResponseTime float64
}

func NewIntegratedRuntimeSystem(cm *RuntimeConfigManager, hr *HotReloadSystem, vm *ConfigVersionManager) (*IntegratedRuntimeSystem, error) {
	return &IntegratedRuntimeSystem{configManager: cm, hotReloader: hr, versionManager: vm}, nil
}
func (irs *IntegratedRuntimeSystem) DeployConfiguration(cfg *RuntimeBusinessConfig, changeDescription string) error {
	irs.mutex.Lock()
	defer irs.mutex.Unlock()
	if irs.versionManager != nil {
		if err := irs.versionManager.SaveVersion(cfg, "%s", changeDescription); err != nil {
			return fmt.Errorf("save version: %w", err)
		}
	}
	if err := irs.configManager.UpdateConfiguration(cfg); err != nil {
		return fmt.Errorf("update configuration: %w", err)
	}
	return nil
}
func (irs *IntegratedRuntimeSystem) GetCurrentConfiguration() *RuntimeBusinessConfig {
	irs.mutex.RLock()
	defer irs.mutex.RUnlock()
	return irs.configManager.GetCurrentConfig()
}
func (irs *IntegratedRuntimeSystem) RollbackToVersion(v string) error {
	irs.mutex.Lock()
	defer irs.mutex.Unlock()
	if irs.versionManager == nil {
		return fmt.Errorf("version manager not available")
	}
	target, err := irs.versionManager.RollbackToVersion(v)
	if err != nil {
		return fmt.Errorf("get target version: %w", err)
	}
	if err := irs.configManager.UpdateConfiguration(target); err != nil {
		return fmt.Errorf("apply rollback: %w", err)
	}
	return nil
}

type defaultConfigValidator struct{}

func (dcv *defaultConfigValidator) Validate(cfg *RuntimeBusinessConfig) error {
	if cfg.BusinessPolicies != nil && cfg.BusinessPolicies.GlobalPolicy != nil {
		gp := cfg.BusinessPolicies.GlobalPolicy
		if gp.MaxConcurrency < 0 {
			return fmt.Errorf("invalid global policy: concurrency must be non-negative")
		}
		if gp.Timeout < 0 {
			return fmt.Errorf("invalid global policy: timeout must be non-negative")
		}
	}
	return nil
}
