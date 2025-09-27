package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestPipelineMetricsSnapshot(t *testing.T){
	cfg := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:8}
	pl := NewPipeline(cfg); defer pl.Stop()
	urls := []string{"https://example.com/m1","https://example.com/m2"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel()
	for range pl.ProcessURLs(ctx, urls) {}
	m := pl.Metrics()
	if m.TotalProcessed < len(urls) { t.Fatalf("expected >=%d processed got %d", len(urls), m.TotalProcessed) }
	if m.Duration <= 0 { t.Fatalf("expected duration > 0") }
	for _, stage := range []string{"discovery","extraction","processing","output"} { if m.StageMetrics[stage].Processed == 0 { t.Fatalf("stage %s zero processed", stage) } }
}
