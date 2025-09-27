package pipeline

import (
	"context"
	"testing"
	"time"
)

// Ensures bounded channels + slower extraction introduce noticeable latency.
func TestPipelineBackpressure(t *testing.T){
	cfg := &PipelineConfig{DiscoveryWorkers:2,ExtractionWorkers:1,ProcessingWorkers:2,OutputWorkers:2,BufferSize:5}
	pl := NewPipeline(cfg); defer pl.Stop()
	urls := make([]string, 25)
	for i := range urls { urls[i] = "https://example.com/page" + time.Now().Add(time.Duration(i)*time.Nanosecond).Format("150405.000000") }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel()
	start := time.Now(); results := pl.ProcessURLs(ctx, urls)
	count := 0; for range results { count++ }
	elapsed := time.Since(start)
	if count != len(urls) { t.Fatalf("expected %d got %d", len(urls), count) }
	if elapsed < 100*time.Millisecond { t.Fatalf("pipeline too fast (%v) backpressure likely broken", elapsed) }
}
