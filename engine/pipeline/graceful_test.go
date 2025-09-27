package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestPipelineGracefulShutdown(t *testing.T){
	cfg := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:10}
	pl := NewPipeline(cfg)
	urls := []string{"https://example.com/a","https://example.com/b","https://example.com/c"}
	ctx, cancel := context.WithCancel(context.Background())
	results := pl.ProcessURLs(ctx, urls)
	go func(){ time.Sleep(50 * time.Millisecond); cancel() }()
	processed := 0
	for range results { processed++ }
	if processed > len(urls) { t.Fatalf("processed more than input: %d > %d", processed, len(urls)) }
	pl.Stop()
}
