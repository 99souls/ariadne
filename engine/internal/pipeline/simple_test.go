package pipeline

import (
    "context"
    "testing"
    "time"
)

func TestSimplePipeline_Migrated(t *testing.T){
    config := &PipelineConfig{DiscoveryWorkers:1, ExtractionWorkers:1, ProcessingWorkers:1, OutputWorkers:1, BufferSize:2}
    p := NewPipeline(config); defer p.Stop()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel()
    results := p.ProcessURLs(ctx, []string{"https://example.com/test"})
    count := 0
    for range results { count++ }
    if count != 1 { t.Fatalf("expected 1 result got %d", count) }
}
