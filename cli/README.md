# Ariadne CLI (Phase 5F Skeleton)

This module provides the evolving command-line interface for the Ariadne engine.

Status: Early scaffold (hard cut extraction). Flags expanding in Wave 4 (metrics, health, basic config file).

Basic example:

```
go run ./cli/cmd/ariadne -seeds https://example.com -snapshot-interval 5s
```

Metrics & health (experimental):

```
go run ./cli/cmd/ariadne -seeds https://example.com \
	-enable-metrics -metrics :9090 -metrics-backend prom -health :9091
```

Minimal JSON config (temporary format until layered YAML arrives):

config.json

```json
{
  "discovery_workers": 2,
  "extraction_workers": 4,
  "processing_workers": 2,
  "output_workers": 1
}
```

Run with config overlay:

```
go run ./cli/cmd/ariadne -config config.json -seeds https://example.com
```

Available flags (current Wave 4):

| Flag               | Purpose                                           |
| ------------------ | ------------------------------------------------- | ---- | ---------------------------------------------------- |
| -seeds             | Comma separated seed URLs                         |
| -seed-file         | File with one seed per line (comments with #)     |
| -resume            | Resume mode (skip seeds found in checkpoint)      |
| -checkpoint        | Checkpoint log path (default checkpoint.log)      |
| -snapshot-interval | Progress snapshot cadence (0 disables)            |
| -metrics           | Metrics listen address (requires -enable-metrics) |
| -enable-metrics    | Enable metrics provider selection                 |
| -metrics-backend   | prom                                              | otel | noop (effective only if -enable-metrics is supplied) |
| -health            | Health endpoint listen address                    |
| -config            | Minimal JSON config overlay (temporary)           |
| -version           | Print version / build info                        |

Metrics adapter notes:

- When `-enable-metrics -metrics :PORT` are provided and backend is `prom` the Prometheus registry is exposed directly.
- For backends that do not provide an HTTP handler the CLI serves a small placeholder metric so `/metrics` never 404s.
- Health endpoint responds with JSON: `{"status":"healthy|degraded|unhealthy|unknown","probes":[...],"generated":RFC3339,"ttl":seconds}`.

Planned additions (Phase 6):

- YAML config layering & env overrides
- Structured output mode selection (JSONL / NDJSON / human)
- Richer snapshot formatting / progress bars
- Pluggable output sinks (files, directory tree) beyond stdout

Development:

- Module replacement in go.work points to local `../engine` during development.
- Run via: `go run ./cli/cmd/ariadne -seeds https://example.com`.
