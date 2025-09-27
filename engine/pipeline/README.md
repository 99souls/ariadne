Pipeline (Engine Package)
=================================

Canonical multi-stage crawling pipeline used by the Engine facade.

Current Status (Post-Migration):
* Internal legacy implementation removed.
* Tests fully reside in this package.
* Helpers like isValidURL / extractContent are simulation stubs slated for replacement by real components (processor, fetcher, parser) in later phases.

Provided Features:
* Staged concurrency (discovery → extraction → processing → output)
* Retry with exponential backoff + jitter (extraction stage)
* Optional adaptive rate limiting per domain
* Resource manager integration (in‑memory cache + spill + checkpoints + in‑flight slots)
* Metrics & snapshot primitives used by engine facade

Planned Enhancements:
1. Replace simulation helpers with real fetch / parse pipeline components.
2. Add hooks or interfaces for custom processors / output sinks.
3. Expand metrics (percentiles, stage latencies, retries counts).
4. Export structured tracing events (OpenTelemetry integration TBD).
5. Fine‑grained benchmarks (per stage) and race detection CI job.

Benchmarks
----------
Run with `go test -bench=. -benchmem ./packages/engine/pipeline` (after adding real implementations these will inform tuning).

Design Notes
------------
Channels are bounded to apply natural backpressure. Result aggregation decouples consumer pace from internal stage completion while still enforcing expected result count for early shutdown.

Usage (direct)
--------------
	cfg := &pipeline.PipelineConfig{DiscoveryWorkers:2,ExtractionWorkers:2,ProcessingWorkers:2,OutputWorkers:1,BufferSize:128}
	pl  := pipeline.NewPipeline(cfg)
	defer pl.Stop()
	results := pl.ProcessURLs(context.Background(), []string{"https://example.com"})
	for r := range results { _ = r }

Prefer constructing pipelines through the engine facade unless advanced control is required.
