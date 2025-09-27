package pipeline

import (
	"context"
	"testing"
)

// Basic benchmark for end-to-end processing (synthetic helpers).
func BenchmarkPipelineThroughput(b *testing.B){
	cfg := &PipelineConfig{DiscoveryWorkers:2,ExtractionWorkers:2,ProcessingWorkers:2,OutputWorkers:1,BufferSize:256}
	for i := 0; i < b.N; i++ {
		pl := NewPipeline(cfg)
		urls := []string{"https://example.com/a","https://example.com/b","https://example.com/c"}
		ctx := context.Background()
		for range pl.ProcessURLs(ctx, urls) {}
		pl.Stop()
	}
}
