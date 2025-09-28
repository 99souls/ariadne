package pipeline

import (
    "context"
    "testing"
    "time"
)

func TestPipelineDataFlow_Migrated(t *testing.T){
    config := &PipelineConfig{DiscoveryWorkers:1, ExtractionWorkers:1, ProcessingWorkers:1, OutputWorkers:1, BufferSize:10}
    p := NewPipeline(config); defer p.Stop()
    urls := []string{"https://example.com/page1","https://example.com/page2"}
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second); defer cancel()
    results := p.ProcessURLs(ctx, urls)
    count := 0
    for r := range results { if r.Stage != "output" { t.Errorf("expected output stage got %s", r.Stage) }; if r.Page == nil { t.Error("expected page data") }; count++ }
    if count != len(urls) { t.Fatalf("expected %d results got %d", len(urls), count) }
}
