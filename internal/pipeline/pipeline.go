package pipeline

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/99souls/ariadne/engine/ratelimit"
	"ariadne/internal/resources"

	"github.com/99souls/ariadne/engine/models"
)

// PipelineConfig defines configuration for multi-stage pipeline
// PipelineConfig defines configuration for multi-stage pipeline
type PipelineConfig struct {
	DiscoveryWorkers  int `yaml:"discovery_workers" json:"discovery_workers"`   // URL discovery workers
	ExtractionWorkers int `yaml:"extraction_workers" json:"extraction_workers"` // Content extraction workers
	ProcessingWorkers int `yaml:"processing_workers" json:"processing_workers"` // Content processing workers
	OutputWorkers     int `yaml:"output_workers" json:"output_workers"`         // Output generation workers
	BufferSize        int `yaml:"buffer_size" json:"buffer_size"`               // Channel buffer size

	RateLimiter      ratelimit.RateLimiter `yaml:"-" json:"-"`
	RetryBaseDelay   time.Duration         `yaml:"retry_base_delay" json:"retry_base_delay"`
	RetryMaxDelay    time.Duration         `yaml:"retry_max_delay" json:"retry_max_delay"`
	RetryMaxAttempts int                   `yaml:"retry_max_attempts" json:"retry_max_attempts"`
	ResourceManager  *resources.Manager    `yaml:"-" json:"-"`
}

type extractionTask struct {
	url     string
	attempt int
}

// StageStatus represents the status of a pipeline stage
// StageStatus represents the status of a pipeline stage
type StageStatus struct {
	Name    string `json:"name"`
	Workers int    `json:"workers"`
	Active  bool   `json:"active"`
	Queue   int    `json:"queue"` // Items in queue for this stage
}

// StageMetrics represents metrics for a pipeline stage
// StageMetrics represents metrics for a pipeline stage
type StageMetrics struct {
	Processed int           `json:"processed"`
	Failed    int           `json:"failed"`
	AvgTime   time.Duration `json:"avg_time"`
}

// PipelineMetrics represents overall pipeline metrics
// PipelineMetrics represents overall pipeline metrics
type PipelineMetrics struct {
	TotalProcessed int                     `json:"total_processed"`
	TotalFailed    int                     `json:"total_failed"`
	StartTime      time.Time               `json:"start_time"`
	Duration       time.Duration           `json:"duration"`
	StageMetrics   map[string]StageMetrics `json:"stage_metrics"`
}

// Pipeline represents the multi-stage processing pipeline
// Pipeline represents the multi-stage processing pipeline
type Pipeline struct {
	config *PipelineConfig

	// Pipeline stages channels
	urlQueue        chan string              // URLs to discover
	extractionQueue chan extractionTask      // URLs ready for extraction
	processingQueue chan *models.Page        // Pages ready for processing
	outputQueue     chan *models.CrawlResult // Results ready for output
	resultsInternal chan *models.CrawlResult // Internal aggregation channel
	results         chan *models.CrawlResult // Final results exposed to callers

	// Stage control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	mutex   sync.RWMutex
	metrics *PipelineMetrics

	// Stage workers tracking
	stageStatus      map[string]*StageStatus
	closeResultsOnce sync.Once
	expectedResults  int64
	resultCount      int64

	discoveryWG  sync.WaitGroup
	extractionWG sync.WaitGroup
	processingWG sync.WaitGroup
	outputWG     sync.WaitGroup

	retryWG sync.WaitGroup

	limiter         ratelimit.RateLimiter
	resourceManager *resources.Manager

	randMu sync.Mutex
	rand   *rand.Rand
}

// NewPipeline creates a new multi-stage pipeline.
//
// DEPRECATION NOTICE (P6 roadmap): External callers should migrate to the engine
// facade (`packages/engine`). Direct construction will become internal in a future
// phase once the facade and CLI migration stabilize. Tests within this package
// continue to rely on NewPipeline; production entrypoints should not.
// NewPipeline creates a new multi-stage pipeline.
//
// DEPRECATION NOTICE: external callers should migrate to the engine facade.
func NewPipeline(config *PipelineConfig) *Pipeline {
	ctx, cancel := context.WithCancel(context.Background())

	if config.RetryBaseDelay <= 0 {
		config.RetryBaseDelay = 200 * time.Millisecond
	}
	if config.RetryMaxDelay <= 0 {
		config.RetryMaxDelay = 5 * time.Second
	}
	if config.RetryMaxAttempts <= 0 {
		config.RetryMaxAttempts = 3
	}

	randSource := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(randSource)

	pipeline := &Pipeline{
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		urlQueue:        make(chan string, config.BufferSize),
		extractionQueue: make(chan extractionTask, config.BufferSize),
		processingQueue: make(chan *models.Page, config.BufferSize),
		outputQueue:     make(chan *models.CrawlResult, config.BufferSize),
		resultsInternal: make(chan *models.CrawlResult, config.BufferSize),
		results:         make(chan *models.CrawlResult, config.BufferSize),
		metrics: &PipelineMetrics{
			StartTime:    time.Now(),
			StageMetrics: make(map[string]StageMetrics),
		},
		stageStatus:     make(map[string]*StageStatus),
		limiter:         config.RateLimiter,
		resourceManager: config.ResourceManager,
		rand:            randGen,
	}

	// Initialize stage status
	pipeline.initStageStatus()

	// Start pipeline stages
	pipeline.startStages()

	// Start result aggregation and completion monitoring
	pipeline.startResultAggregator()

	return pipeline
}

// Config returns the pipeline configuration
// Config returns the pipeline configuration
func (p *Pipeline) Config() *PipelineConfig {
	return p.config
}

// StageStatus returns the status of a specific stage
// StageStatus returns the status of a specific stage
func (p *Pipeline) StageStatus(stageName string) *StageStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if status, exists := p.stageStatus[stageName]; exists {
		return status
	}

	return &StageStatus{Name: stageName, Active: false}
}

// ProcessURLs processes a list of URLs through the complete pipeline
// ProcessURLs processes a list of URLs through the complete pipeline
func (p *Pipeline) ProcessURLs(ctx context.Context, urls []string) <-chan *models.CrawlResult {
	atomic.StoreInt64(&p.expectedResults, int64(len(urls)))
	atomic.StoreInt64(&p.resultCount, 0)

	// Create a context that respects both pipeline and caller context
	processCtx, processCancel := context.WithCancel(ctx)

	go func() {
		defer processCancel()
		defer close(p.urlQueue)

		// Feed URLs into the pipeline
		for _, url := range urls {
			select {
			case p.urlQueue <- url:
			case <-processCtx.Done():
				return
			case <-p.ctx.Done():
				return
			}
		}
	}()

	// Return results channel
	return p.results
}

// Metrics returns current pipeline metrics
// Metrics returns current pipeline metrics
func (p *Pipeline) Metrics() *PipelineMetrics {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Create a copy with current duration
	metrics := *p.metrics
	metrics.Duration = time.Since(metrics.StartTime)

	return &metrics
}

// Stop gracefully stops the pipeline
// Stop gracefully stops the pipeline
func (p *Pipeline) Stop() {
	p.cancel()
	p.retryWG.Wait()
	p.wg.Wait() // Wait for all workers (including aggregator) to finish

	// Mark stages inactive
	p.mutex.Lock()
	for _, st := range p.stageStatus {
		st.Active = false
	}
	p.mutex.Unlock()
	p.closeResults()

	if closable, ok := p.limiter.(interface{ Close() error }); ok {
		_ = closable.Close()
	}
}

// (Remaining implementation already present below in original file sections.)

// startStages starts all pipeline stage workers
func (p *Pipeline) startStages() {
	// Start discovery workers
	p.discoveryWG.Add(p.config.DiscoveryWorkers)
	for i := 0; i < p.config.DiscoveryWorkers; i++ {
		p.wg.Add(1)
		go p.discoveryWorker()
	}
	go func() {
		p.discoveryWG.Wait()
		<-p.ctx.Done()
		p.retryWG.Wait()
		close(p.extractionQueue)
	}()

	// Start extraction workers
	p.extractionWG.Add(p.config.ExtractionWorkers)
	for i := 0; i < p.config.ExtractionWorkers; i++ {
		p.wg.Add(1)
		go p.extractionWorker()
	}
	go func() {
		p.extractionWG.Wait()
		close(p.processingQueue)
	}()

	// Start processing workers
	p.processingWG.Add(p.config.ProcessingWorkers)
	for i := 0; i < p.config.ProcessingWorkers; i++ {
		p.wg.Add(1)
		go p.processingWorker()
	}
	go func() {
		p.processingWG.Wait()
		close(p.outputQueue)
	}()

	// Start output workers
	p.outputWG.Add(p.config.OutputWorkers)
	for i := 0; i < p.config.OutputWorkers; i++ {
		p.wg.Add(1)
		go p.outputWorker()
	}
	go func() {
		p.outputWG.Wait()
		close(p.resultsInternal)
	}()
}

func (p *Pipeline) startResultAggregator() {
	p.wg.Add(1)
	go p.monitorResults()
}

func (p *Pipeline) monitorResults() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			p.drainResultsInternal()
			p.closeResults()
			return
		case result, ok := <-p.resultsInternal:
			if !ok {
				p.closeResults()
				return
			}

			if !p.forwardResult(result) {
				p.drainResultsInternal()
				p.closeResults()
				return
			}

			newCount := atomic.AddInt64(&p.resultCount, 1)
			expected := atomic.LoadInt64(&p.expectedResults)

			if expected > 0 && newCount >= expected {
				p.cancel()
				p.drainResultsInternal()
				p.closeResults()
				return
			}
		}
	}
}

func (p *Pipeline) forwardResult(result *models.CrawlResult) bool {
	select {
	case <-p.ctx.Done():
		return false
	case p.results <- result:
		return true
	}
}

func (p *Pipeline) closeResults() {
	p.closeResultsOnce.Do(func() {
		close(p.results)
	})
}

func (p *Pipeline) drainResultsInternal() {
	for {
		select {
		case _, ok := <-p.resultsInternal:
			if !ok {
				return
			}
			continue
		default:
			return
		}
	}
}

func (p *Pipeline) deliverResult(result *models.CrawlResult) bool {
	select {
	case <-p.ctx.Done():
		return false
	case p.resultsInternal <- result:
		if p.resourceManager != nil && result != nil {
			checkpointURL := result.URL
			if checkpointURL == "" && result.Page != nil && result.Page.URL != nil {
				checkpointURL = result.Page.URL.String()
			}
			if checkpointURL != "" {
				p.resourceManager.Checkpoint(checkpointURL)
			}
		}
		return true
	}
}

func (p *Pipeline) forwardToProcessing(page *models.Page, fromCache bool) bool {
	if page == nil {
		return false
	}

	select {
	case p.processingQueue <- page:
		if fromCache {
			p.updateStageMetrics("cache", true)
		} else {
			p.updateStageMetrics("extraction", true)
		}
		return true
	case <-p.ctx.Done():
		return false
	}
}

func (p *Pipeline) enqueueExtraction(url string, attempt int) bool {
	task := extractionTask{url: url, attempt: attempt}
	var sent bool
	defer func() {
		if r := recover(); r != nil {
			sent = false
		}
	}()

	select {
	case <-p.ctx.Done():
		return false
	case p.extractionQueue <- task:
		sent = true
	}

	return sent
}

func (p *Pipeline) scheduleRetry(url string, attempt int, delay time.Duration) {
	if p.config.RetryMaxAttempts > 0 && attempt >= p.config.RetryMaxAttempts {
		return
	}

	if err := p.ctx.Err(); err != nil {
		return
	}

	p.retryWG.Add(1)
	go func() {
		defer p.retryWG.Done()

		if delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop()

			select {
			case <-p.ctx.Done():
				return
			case <-timer.C:
			}
		} else {
			select {
			case <-p.ctx.Done():
				return
			default:
			}
		}

		if err := p.ctx.Err(); err != nil {
			return
		}

		p.enqueueExtraction(url, attempt)
	}()
}

func (p *Pipeline) shouldRetry(task extractionTask) bool {
	if p.config.RetryMaxAttempts <= 0 {
		return false
	}
	return task.attempt+1 < p.config.RetryMaxAttempts
}

func (p *Pipeline) backoffDelay(attempt int) time.Duration {
	base := p.config.RetryBaseDelay
	max := p.config.RetryMaxDelay
	if base <= 0 {
		base = 200 * time.Millisecond
	}
	if max <= 0 {
		max = 5 * time.Second
	}

	delay := base * time.Duration(1<<(attempt-1))
	if delay > max {
		delay = max
	}

	jitter := p.randomizedDelay(delay)
	if jitter <= 0 {
		return delay
	}
	return jitter
}

func (p *Pipeline) randomizedDelay(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	p.randMu.Lock()
	defer p.randMu.Unlock()
	return time.Duration(p.rand.Float64() * float64(max))
}

func (p *Pipeline) acquirePermit(task extractionTask, domain string) (ratelimit.Permit, error) {
	if p.limiter == nil || domain == "" {
		return nil, nil
	}

	permit, err := p.limiter.Acquire(p.ctx, domain)
	if err != nil {
		return nil, err
	}

	return permit, nil
}

func extractDomain(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}

// discoveryWorker processes URL discovery stage
func (p *Pipeline) discoveryWorker() {
	defer p.wg.Done()
	defer p.discoveryWG.Done()

	for {
		select {
		case url, ok := <-p.urlQueue:
			if !ok {
				return
			}

			// Simulate URL validation and processing
			if p.isValidURL(url) {
				if p.enqueueExtraction(url, 0) {
					p.updateStageMetrics("discovery", true)
				} else {
					return
				}
			} else {
				p.updateStageMetrics("discovery", false)
				p.sendErrorResult(url, "discovery", "invalid URL", false)
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// extractionWorker processes content extraction stage
func (p *Pipeline) extractionWorker() {
	defer p.wg.Done()
	defer p.extractionWG.Done()

	for {
		select {
		case task, ok := <-p.extractionQueue:
			if !ok {
				return
			}

			domain := extractDomain(task.url)
			manager := p.resourceManager
			if manager != nil {
				cachedPage, hit, err := manager.GetPage(task.url)
				if err != nil {
					p.updateStageMetrics("extraction", false)
					p.sendErrorResult(task.url, "extraction", fmt.Sprintf("cache lookup failed: %v", err), false)
					continue
				}
				if hit && cachedPage != nil {
					if !p.forwardToProcessing(cachedPage, true) {
						return
					}
					continue
				}
			}

			permit, err := p.acquirePermit(task, domain)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				p.updateStageMetrics("extraction", false)
				if errors.Is(err, ratelimit.ErrCircuitOpen) && p.shouldRetry(task) {
					delay := p.backoffDelay(task.attempt + 1)
					p.scheduleRetry(task.url, task.attempt+1, delay)
					continue
				}
				p.sendErrorResult(task.url, "extraction", err.Error(), false)
				continue
			}

			var slotAcquired bool
			if manager != nil {
				if acquireErr := manager.Acquire(p.ctx); acquireErr != nil {
					if permit != nil {
						permit.Release()
					}
					if errors.Is(acquireErr, context.Canceled) {
						return
					}
					p.updateStageMetrics("extraction", false)
					p.sendErrorResult(task.url, "extraction", acquireErr.Error(), false)
					continue
				}
				slotAcquired = true
			}

			start := time.Now()
			page := p.extractContent(task.url)
			latency := time.Since(start)

			if permit != nil {
				permit.Release()
			}

			releaseSlot := func() {
				if slotAcquired && manager != nil {
					manager.Release()
					slotAcquired = false
				}
			}

			if page != nil {
				if manager != nil {
					if err := manager.StorePage(task.url, page); err != nil {
						releaseSlot()
						p.updateStageMetrics("extraction", false)
						p.sendErrorResult(task.url, "extraction", fmt.Sprintf("cache store failed: %v", err), false)
						continue
					}
				}
				releaseSlot()

				if p.limiter != nil && domain != "" {
					p.limiter.Feedback(domain, ratelimit.Feedback{StatusCode: 200, Latency: latency})
				}
				if !p.forwardToProcessing(page, false) {
					return
				}
			} else {
				releaseSlot()
				if p.limiter != nil && domain != "" {
					p.limiter.Feedback(domain, ratelimit.Feedback{StatusCode: 503, Latency: latency, Err: errors.New("extraction failed")})
				}
				p.updateStageMetrics("extraction", false)
				if p.shouldRetry(task) {
					delay := p.backoffDelay(task.attempt + 1)
					p.scheduleRetry(task.url, task.attempt+1, delay)
					continue
				}
				p.sendErrorResult(task.url, "extraction", fmt.Sprintf("failed after %d attempts", task.attempt+1), false)
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// processingWorker processes content processing stage
func (p *Pipeline) processingWorker() {
	defer p.wg.Done()
	defer p.processingWG.Done()

	for {
		select {
		case page, ok := <-p.processingQueue:
			if !ok {
				return
			}

			// Simulate content processing (would call actual processor)
			result := p.processContent(page)

			select {
			case p.outputQueue <- result:
				p.updateStageMetrics("processing", result.Success)
			case <-p.ctx.Done():
				return
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// outputWorker processes output generation stage
func (p *Pipeline) outputWorker() {
	defer p.wg.Done()
	defer p.outputWG.Done()

	for {
		select {
		case result, ok := <-p.outputQueue:
			if !ok {
				return
			}

			// Mark as output stage and send final result
			result.Stage = "output"
			if !p.deliverResult(result) {
				return
			}
			p.updateStageMetrics("output", result.Success)

		case <-p.ctx.Done():
			return
		}
	}
}

// Helper methods for simulation (to be replaced with real implementations)

func (p *Pipeline) isValidURL(url string) bool {
	// Simulate URL validation
	return url != "" && url != "invalid-url"
}

func (p *Pipeline) extractContent(rawURL string) *models.Page {
	// Simulate content extraction behavior with controllable scenarios
	if strings.Contains(rawURL, "fail-extraction") {
		time.Sleep(5 * time.Millisecond)
		return nil
	}

	if strings.Contains(rawURL, "slow") {
		time.Sleep(50 * time.Millisecond)
	} else {
		time.Sleep(10 * time.Millisecond)
	}

	page := &models.Page{
		Title:   "Test Page",
		Content: "<h1>Test Content</h1>",
	}

	if parsed, err := url.Parse(rawURL); err == nil {
		page.URL = parsed
	}
	page.CrawledAt = time.Now()
	return page
}

func (p *Pipeline) processContent(page *models.Page) *models.CrawlResult {
	// Simulate content processing
	time.Sleep(5 * time.Millisecond)
	if page != nil {
		page.ProcessedAt = time.Now()
	}

	resultURL := ""
	if page != nil && page.URL != nil {
		resultURL = page.URL.String()
	}
	return &models.CrawlResult{
		URL:     resultURL,
		Page:    page,
		Success: true,
		Stage:   "processing",
	}
}

func (p *Pipeline) sendErrorResult(url, stage, message string, retry bool) {
	result := &models.CrawlResult{
		URL:     url,
		Error:   models.NewCrawlError(url, stage, errors.New(message)),
		Success: false,
		Stage:   stage,
		Retry:   retry,
	}

	p.deliverResult(result)
}

func (p *Pipeline) updateStageMetrics(stage string, success bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	metrics := p.metrics.StageMetrics[stage]

	if success {
		metrics.Processed++
		if stage != "cache" {
			p.metrics.TotalProcessed++
		}
	} else {
		metrics.Failed++
		if stage != "cache" {
			p.metrics.TotalFailed++
		}
	}

	p.metrics.StageMetrics[stage] = metrics
}

// initStageStatus initializes the stage status metadata for inspection via StageStatus()
func (p *Pipeline) initStageStatus() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.stageStatus["discovery"] = &StageStatus{Name: "discovery", Workers: p.config.DiscoveryWorkers, Active: p.config.DiscoveryWorkers > 0}
	p.stageStatus["extraction"] = &StageStatus{Name: "extraction", Workers: p.config.ExtractionWorkers, Active: p.config.ExtractionWorkers > 0}
	p.stageStatus["processing"] = &StageStatus{Name: "processing", Workers: p.config.ProcessingWorkers, Active: p.config.ProcessingWorkers > 0}
	p.stageStatus["output"] = &StageStatus{Name: "output", Workers: p.config.OutputWorkers, Active: p.config.OutputWorkers > 0}
}
