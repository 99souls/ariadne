Pipeline (Engine Package)

This is the canonical pipeline implementation after migration off internal/pipeline.

Migration Status:

- Engine facade now depends on packages/engine/pipeline.
- Internal pipeline package is still present ONLY because its tests remain there.

Planned Next Steps:

1. Port tests from internal/pipeline/_\_test.go into packages/engine/pipeline (adjust imports to eng_ packages).
2. Remove internal/pipeline entirely once test parity is confirmed.
3. Tighten visibility (unexport helpers that should not be public on facade) and expose only necessary API via engine.
4. Add benchmarks & doc examples here.

After step (2), external code should construct pipelines exclusively via the engine facade (preferred) or directly via engpipeline.NewPipeline if low-level control is required.
