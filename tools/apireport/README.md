# apireport Tool

Generates `API_REPORT.md` enumerating exported symbols in curated engine packages.

Usage:

```
go run ./tools/apireport -out API_REPORT.md
```

Included packages are defined in `main.go` via `includePkgs` slice.
