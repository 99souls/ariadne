# AssetPolicy Configuration Reference

Status: Draft (Iteration 8)

```go
type AssetPolicy struct {
    Enabled        bool          // Master switch
    MaxBytes       int64         // Per-page cumulative fetch cap (0 = unlimited)
    MaxPerPage     int           // Maximum assets selected per page (0 = unlimited)
    InlineMaxBytes int64         // Heuristic inline threshold (future: actual size probe)
    Optimize       bool          // Apply lightweight optimization passes
    RewritePrefix  string        // Output path prefix (must start with /)
    AllowTypes     []string      // Whitelist of types (empty = all not blocked)
    BlockTypes     []string      // Explicit blocklist
    MaxConcurrent  int           // Worker pool size (0 => auto capped)
}
```

## Supported Types (Current)
`img`, `script`, `stylesheet`, `media`, `doc`

Additional discovery recognizes preload hints and srcset/source variants, but types are normalized to the above (e.g. preload as=image -> `img`).

## Behavior Notes
- `Optimize` currently performs whitespace collapse for css/js and tags image assets with a placeholder optimization marker.
- `InlineMaxBytes` uses a heuristic (filename hints) until actual size probing is implemented; inline mode still fetches bytes.
- `MaxConcurrent` is bounded internally (hard cap 8) to prevent oversubscription.

## Validation Rules
- If `Enabled` and `RewritePrefix` does not start with `/`, configuration validation fails.

## Metrics Snapshot Access
```go
snap := engine.AssetMetricsSnapshot{
    Discovered: ..., Selected: ..., Downloaded: ..., Failed: ..., BytesIn: ..., BytesOut: ...,
}
```
Use after processing hooks to observe per-run aggregate counts.

## Events
Inspect with `eng.AssetEvents()`; ring buffer (size 1024) returns recent events.

---
End of Config Reference.
