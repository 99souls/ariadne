package pipeline

import (
    "context"
    "testing"
)

func TestPipelineResultCounting_Migrated(t *testing.T){
    config := &PipelineConfig{DiscoveryWorkers:1, ExtractionWorkers:1, ProcessingWorkers:1, OutputWorkers:1, BufferSize:2}
    p := NewPipeline(config); defer p.Stop()
    urls := []string{"https://example.com/test"}
    ctx := context.Background()
    results := p.ProcessURLs(ctx, urls)
    count := 0
    for range results { count++ }
    if count != 1 { t.Fatalf("expected 1 result got %d", count) }
}
