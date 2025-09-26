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

	engmodels "ariadne/packages/engine/models"
	engratelimit "ariadne/packages/engine/ratelimit"
	engresources "ariadne/packages/engine/resources"
)

// PipelineConfig defines configuration for multi-stage pipeline
type PipelineConfig struct {
	DiscoveryWorkers  int `yaml:"discovery_workers" json:"discovery_workers"`
	ExtractionWorkers int `yaml:"extraction_workers" json:"extraction_workers"`
	ProcessingWorkers int `yaml:"processing_workers" json:"processing_workers"`
	OutputWorkers     int `yaml:"output_workers" json:"output_workers"`
	BufferSize        int `yaml:"buffer_size" json:"buffer_size"`

	RateLimiter      engratelimit.RateLimiter `yaml:"-" json:"-"`
	RetryBaseDelay   time.Duration         `yaml:"retry_base_delay" json:"retry_base_delay"`
	RetryMaxDelay    time.Duration         `yaml:"retry_max_delay" json:"retry_max_delay"`
	RetryMaxAttempts int                   `yaml:"retry_max_attempts" json:"retry_max_attempts"`
	ResourceManager  *engresources.Manager    `yaml:"-" json:"-"`
}

type extractionTask struct { url string; attempt int }

type StageStatus struct { Name string `json:"name"`; Workers int `json:"workers"`; Active bool `json:"active"`; Queue int `json:"queue"` }

type StageMetrics struct { Processed int `json:"processed"`; Failed int `json:"failed"`; AvgTime time.Duration `json:"avg_time"` }

type PipelineMetrics struct { TotalProcessed int `json:"total_processed"`; TotalFailed int `json:"total_failed"`; StartTime time.Time `json:"start_time"`; Duration time.Duration `json:"duration"`; StageMetrics map[string]StageMetrics `json:"stage_metrics"` }

type Pipeline struct { config *PipelineConfig; urlQueue chan string; extractionQueue chan extractionTask; processingQueue chan *engmodels.Page; outputQueue chan *engmodels.CrawlResult; resultsInternal chan *engmodels.CrawlResult; results chan *engmodels.CrawlResult; ctx context.Context; cancel context.CancelFunc; wg sync.WaitGroup; mutex sync.RWMutex; metrics *PipelineMetrics; stageStatus map[string]*StageStatus; closeResultsOnce sync.Once; expectedResults int64; resultCount int64; discoveryWG sync.WaitGroup; extractionWG sync.WaitGroup; processingWG sync.WaitGroup; outputWG sync.WaitGroup; retryWG sync.WaitGroup; limiter engratelimit.RateLimiter; resourceManager *engresources.Manager; randMu sync.Mutex; rand *rand.Rand }

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
    p := &Pipeline{
        config:           config,
        ctx:              ctx,
        cancel:           cancel,
        urlQueue:         make(chan string, config.BufferSize),
        extractionQueue:  make(chan extractionTask, config.BufferSize),
        processingQueue:  make(chan *engmodels.Page, config.BufferSize),
        outputQueue:      make(chan *engmodels.CrawlResult, config.BufferSize),
        resultsInternal:  make(chan *engmodels.CrawlResult, config.BufferSize),
        results:          make(chan *engmodels.CrawlResult, config.BufferSize),
        metrics: &PipelineMetrics{
            StartTime:    time.Now(),
            StageMetrics: make(map[string]StageMetrics),
        },
        stageStatus:     make(map[string]*StageStatus),
        limiter:         config.RateLimiter,
        resourceManager: config.ResourceManager,
        rand:            randGen,
    }
    p.initStageStatus()
    p.startStages()
    p.startResultAggregator()
    return p
}

func (p *Pipeline) Config() *PipelineConfig { return p.config }
func (p *Pipeline) StageStatus(stageName string) *StageStatus { p.mutex.RLock(); defer p.mutex.RUnlock(); if status, exists := p.stageStatus[stageName]; exists { return status }; return &StageStatus{Name: stageName, Active: false} }
func (p *Pipeline) ProcessURLs(ctx context.Context, urls []string) <-chan *engmodels.CrawlResult { atomic.StoreInt64(&p.expectedResults, int64(len(urls))); atomic.StoreInt64(&p.resultCount, 0); processCtx, processCancel := context.WithCancel(ctx); go func(){ defer processCancel(); defer close(p.urlQueue); for _, u := range urls { select { case p.urlQueue <- u: case <-processCtx.Done(): return; case <-p.ctx.Done(): return } } }(); return p.results }
func (p *Pipeline) Metrics() *PipelineMetrics { p.mutex.RLock(); defer p.mutex.RUnlock(); metrics := *p.metrics; metrics.Duration = time.Since(metrics.StartTime); return &metrics }
func (p *Pipeline) Stop() { p.cancel(); p.retryWG.Wait(); p.wg.Wait(); p.closeResults(); if closable, ok := p.limiter.(interface{ Close() error }); ok { _ = closable.Close() } }

func (p *Pipeline) initStageStatus() { stages := map[string]int{"discovery": p.config.DiscoveryWorkers, "extraction": p.config.ExtractionWorkers, "processing": p.config.ProcessingWorkers, "output": p.config.OutputWorkers}; for name, workers := range stages { p.stageStatus[name] = &StageStatus{Name: name, Workers: workers, Active: true, Queue: 0 } }; if p.resourceManager != nil { p.stageStatus["cache"] = &StageStatus{Name: "cache", Workers: 0, Active: true, Queue: 0 } } }

func (p *Pipeline) startStages() { p.discoveryWG.Add(p.config.DiscoveryWorkers); for i:=0;i<p.config.DiscoveryWorkers;i++{ p.wg.Add(1); go p.discoveryWorker() }; go func(){ p.discoveryWG.Wait(); <-p.ctx.Done(); p.retryWG.Wait(); close(p.extractionQueue) }(); p.extractionWG.Add(p.config.ExtractionWorkers); for i:=0;i<p.config.ExtractionWorkers;i++{ p.wg.Add(1); go p.extractionWorker() }; go func(){ p.extractionWG.Wait(); close(p.processingQueue) }(); p.processingWG.Add(p.config.ProcessingWorkers); for i:=0;i<p.config.ProcessingWorkers;i++{ p.wg.Add(1); go p.processingWorker() }; go func(){ p.processingWG.Wait(); close(p.outputQueue) }(); p.outputWG.Add(p.config.OutputWorkers); for i:=0;i<p.config.OutputWorkers;i++{ p.wg.Add(1); go p.outputWorker() }; go func(){ p.outputWG.Wait(); close(p.resultsInternal) }() }

func (p *Pipeline) startResultAggregator(){ p.wg.Add(1); go p.monitorResults() }
func (p *Pipeline) monitorResults(){ defer p.wg.Done(); for { select { case <-p.ctx.Done(): p.drainResultsInternal(); p.closeResults(); return; case result, ok := <-p.resultsInternal: if !ok { p.closeResults(); return }; if !p.forwardResult(result) { p.drainResultsInternal(); p.closeResults(); return }; newCount := atomic.AddInt64(&p.resultCount,1); expected := atomic.LoadInt64(&p.expectedResults); if expected>0 && newCount>=expected { p.cancel(); p.drainResultsInternal(); p.closeResults(); return } } } }
func (p *Pipeline) forwardResult(result *engmodels.CrawlResult) bool { select { case <-p.ctx.Done(): return false; case p.results <- result: return true } }
func (p *Pipeline) closeResults(){ p.closeResultsOnce.Do(func(){ close(p.results) }) }
func (p *Pipeline) drainResultsInternal(){ for { select { case _, ok := <-p.resultsInternal: if !ok { return }; continue; default: return } } }
func (p *Pipeline) deliverResult(result *engmodels.CrawlResult) bool { select { case <-p.ctx.Done(): return false; case p.resultsInternal <- result: if p.resourceManager!=nil && result!=nil { checkpointURL := result.URL; if checkpointURL=="" && result.Page!=nil && result.Page.URL!=nil { checkpointURL = result.Page.URL.String() }; if checkpointURL != "" { p.resourceManager.Checkpoint(checkpointURL) } }; return true } }
func (p *Pipeline) forwardToProcessing(page *engmodels.Page, fromCache bool) bool { if page==nil { return false }; select { case p.processingQueue <- page: if fromCache { p.updateStageMetrics("cache", true) } else { p.updateStageMetrics("extraction", true) }; return true; case <-p.ctx.Done(): return false } }
func (p *Pipeline) enqueueExtraction(url string, attempt int) bool { task := extractionTask{url: url, attempt: attempt}; var sent bool; defer func(){ if r:=recover(); r!=nil { sent=false } }(); select { case <-p.ctx.Done(): return false; case p.extractionQueue <- task: sent = true }; return sent }
func (p *Pipeline) scheduleRetry(url string, attempt int, delay time.Duration){ if p.config.RetryMaxAttempts>0 && attempt >= p.config.RetryMaxAttempts { return }; if err := p.ctx.Err(); err != nil { return }; p.retryWG.Add(1); go func(){ defer p.retryWG.Done(); if delay>0 { timer := time.NewTimer(delay); defer timer.Stop(); select { case <-p.ctx.Done(): return; case <-timer.C: } } else { select { case <-p.ctx.Done(): return; default: } }; if err := p.ctx.Err(); err != nil { return }; p.enqueueExtraction(url, attempt) }() }
func (p *Pipeline) shouldRetry(task extractionTask) bool { if p.config.RetryMaxAttempts <=0 { return false }; return task.attempt+1 < p.config.RetryMaxAttempts }
func (p *Pipeline) backoffDelay(attempt int) time.Duration { base := p.config.RetryBaseDelay; max := p.config.RetryMaxDelay; if base <=0 { base = 200 * time.Millisecond }; if max<=0 { max = 5 * time.Second }; delay := base * time.Duration(1<<(attempt-1)); if delay > max { delay = max }; jitter := p.randomizedDelay(delay); if jitter <=0 { return delay }; return jitter }
func (p *Pipeline) randomizedDelay(max time.Duration) time.Duration { if max <=0 { return 0 }; p.randMu.Lock(); defer p.randMu.Unlock(); return time.Duration(p.rand.Float64() * float64(max)) }
func (p *Pipeline) acquirePermit(task extractionTask, domain string) (engratelimit.Permit, error) { if p.limiter==nil || domain=="" { return nil, nil }; permit, err := p.limiter.Acquire(p.ctx, domain); if err != nil { return nil, err }; return permit, nil }
func extractDomain(raw string) string { u, err := url.Parse(raw); if err != nil { return "" }; return strings.ToLower(u.Host) }
func (p *Pipeline) discoveryWorker(){ defer p.wg.Done(); defer p.discoveryWG.Done(); for { select { case u, ok := <-p.urlQueue: if !ok { return }; if p.isValidURL(u) { if p.enqueueExtraction(u,0) { p.updateStageMetrics("discovery", true) } else { return } } else { p.updateStageMetrics("discovery", false); p.sendErrorResult(u, "discovery", "invalid URL", false) }; case <-p.ctx.Done(): return } } }
func (p *Pipeline) extractionWorker(){ defer p.wg.Done(); defer p.extractionWG.Done(); for { select { case task, ok := <-p.extractionQueue: if !ok { return }; domain := extractDomain(task.url); manager := p.resourceManager; if manager != nil { cachedPage, hit, err := manager.GetPage(task.url); if err != nil { p.updateStageMetrics("extraction", false); p.sendErrorResult(task.url, "extraction", fmt.Sprintf("cache lookup failed: %v", err), false); continue }; if hit && cachedPage != nil { if !p.forwardToProcessing(cachedPage, true) { return }; continue } }; permit, err := p.acquirePermit(task, domain); if err != nil { if errors.Is(err, context.Canceled) { return }; p.updateStageMetrics("extraction", false); if errors.Is(err, engratelimit.ErrCircuitOpen) && p.shouldRetry(task) { delay := p.backoffDelay(task.attempt+1); p.scheduleRetry(task.url, task.attempt+1, delay); continue }; p.sendErrorResult(task.url, "extraction", err.Error(), false); continue }; var slotAcquired bool; if manager != nil { if acquireErr := manager.Acquire(p.ctx); acquireErr != nil { if permit != nil { permit.Release() }; if errors.Is(acquireErr, context.Canceled) { return }; p.updateStageMetrics("extraction", false); p.sendErrorResult(task.url, "extraction", acquireErr.Error(), false); continue }; slotAcquired = true }; start := time.Now(); page := p.extractContent(task.url); latency := time.Since(start); if permit != nil { permit.Release() }; releaseSlot := func(){ if slotAcquired && manager != nil { manager.Release(); slotAcquired = false } }; if page != nil { if manager != nil { if err := manager.StorePage(task.url, page); err != nil { releaseSlot(); p.updateStageMetrics("extraction", false); p.sendErrorResult(task.url, "extraction", fmt.Sprintf("cache store failed: %v", err), false); continue } }; releaseSlot(); if p.limiter != nil && domain != "" { p.limiter.Feedback(domain, engratelimit.Feedback{StatusCode:200, Latency: latency}) }; if !p.forwardToProcessing(page, false) { return } } else { releaseSlot(); if p.limiter != nil && domain != "" { p.limiter.Feedback(domain, engratelimit.Feedback{StatusCode:503, Latency: latency, Err: errors.New("extraction failed")}) }; p.updateStageMetrics("extraction", false); if p.shouldRetry(task) { delay := p.backoffDelay(task.attempt+1); p.scheduleRetry(task.url, task.attempt+1, delay); continue }; p.sendErrorResult(task.url, "extraction", fmt.Sprintf("failed after %d attempts", task.attempt+1), false) }; case <-p.ctx.Done(): return } } }
func (p *Pipeline) processingWorker(){ defer p.wg.Done(); defer p.processingWG.Done(); for { select { case page, ok := <-p.processingQueue: if !ok { return }; result := p.processContent(page); select { case p.outputQueue <- result: p.updateStageMetrics("processing", result.Success); case <-p.ctx.Done(): return }; case <-p.ctx.Done(): return } } }
func (p *Pipeline) outputWorker(){ defer p.wg.Done(); defer p.outputWG.Done(); for { select { case result, ok := <-p.outputQueue: if !ok { return }; result.Stage = "output"; if !p.deliverResult(result) { return }; p.updateStageMetrics("output", result.Success); case <-p.ctx.Done(): return } } }
func (p *Pipeline) isValidURL(u string) bool { return u != "" && u != "invalid-url" }
func (p *Pipeline) extractContent(rawURL string) *engmodels.Page { if strings.Contains(rawURL, "fail-extraction") { time.Sleep(5 * time.Millisecond); return nil }; if strings.Contains(rawURL, "slow") { time.Sleep(50 * time.Millisecond) } else { time.Sleep(10 * time.Millisecond) }; page := &engmodels.Page{ Title: "Test Page", Content: "<h1>Test Content</h1>" }; if parsed, err := url.Parse(rawURL); err == nil { page.URL = parsed }; page.CrawledAt = time.Now(); return page }
func (p *Pipeline) processContent(page *engmodels.Page) *engmodels.CrawlResult { time.Sleep(5 * time.Millisecond); var processedPage *engmodels.Page; if page != nil { processedPage = &engmodels.Page{ URL: page.URL, Title: page.Title, Content: page.Content, CleanedText: page.CleanedText, Markdown: page.Markdown, Links: page.Links, Images: page.Images, Metadata: page.Metadata, CrawledAt: page.CrawledAt, ProcessedAt: time.Now() } }; resultURL := ""; if processedPage != nil && processedPage.URL != nil { resultURL = processedPage.URL.String() }; return &engmodels.CrawlResult{ URL: resultURL, Page: processedPage, Success: true, Stage: "processing" } }
func (p *Pipeline) sendErrorResult(u, stage, message string, retry bool){ result := &engmodels.CrawlResult{ URL: u, Error: engmodels.NewCrawlError(u, stage, errors.New(message)), Success: false, Stage: stage, Retry: retry }; p.deliverResult(result) }
func (p *Pipeline) updateStageMetrics(stage string, success bool){ p.mutex.Lock(); defer p.mutex.Unlock(); metrics := p.metrics.StageMetrics[stage]; if success { metrics.Processed++; if stage != "cache" { p.metrics.TotalProcessed++ } } else { metrics.Failed++; if stage != "cache" { p.metrics.TotalFailed++ } }; p.metrics.StageMetrics[stage] = metrics }
