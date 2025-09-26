package pipeline

import "testing"

// Migrated from internal/pipeline/components_test.go
func TestPipelineComponents(t *testing.T) {
	// URL validation
	config := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:2}
	pl := NewPipeline(config); defer pl.Stop()
	if !pl.isValidURL("https://example.com/test") { t.Error("valid URL should pass") }
	if pl.isValidURL("invalid-url") { t.Error("invalid URL should fail") }
	if pl.isValidURL("") { t.Error("empty URL should fail") }
}
