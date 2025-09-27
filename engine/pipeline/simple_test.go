package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestSimplePipeline(t *testing.T){
	config := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:2}
	pl := NewPipeline(config); defer pl.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel()
	urls := []string{"https://example.com/test"}
	results := pl.ProcessURLs(ctx, urls)
	count := 0
	for range results { count++ }
	if count != 1 { t.Fatalf("expected 1 result got %d", count) }
}
