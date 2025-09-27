# Ariadne CLI (Phase 5F Skeleton)

This module provides the evolving command-line interface for the Ariadne engine.

Status: Early scaffold (hard cut extraction). Flags are intentionally minimal.

Current command:

```
ariadne -seeds https://example.com,https://example.org -snapshot-interval 5s
```

Planned additions (Phase 6):
- Metrics / health endpoint flags
- Config file layering (YAML + overrides)
- Structured output modes
- Resume / snapshot UX polish

Development:
- Module replacement in go.work points to local `../engine` during development.
- Run via: `go run ./cli/cmd/ariadne -seeds https://example.com`.
