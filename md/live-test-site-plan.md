# Live Test Site Module Plan (Bun + React)

Status: Draft (Preparation for integration against a real running site)
Owner: Core / Testing
Purpose: Introduce a lightweight, deterministic, zero-external-dependency web application inside the monorepo that the engine + CLI integration tests can crawl. This replaces/augments current synthetic mocks with a realistic HTML/CSS/asset/SPA surface.

---
## 1. Goals
- Provide a **real HTTP origin** with realistic assets, nested routes, metadata, and failure cases.
- Deterministic content (no external API calls / time-based randomness) for stable assertions.
- Very fast startup (<200ms cold) to avoid test overhead.
- Exercise crawler behaviors: link discovery, depth, robots.txt handling, static vs dynamic content.
- Serve a mix of content types: HTML pages, markdown-like embedded sections, images, stylesheets, JSON API endpoint, and intentionally broken links.
- Allow toggling feature flags (query param or header) to test conditional rendering.

## 2. Non-Goals
- No SSR complexity beyond basic React hydration.
- No DB / stateful backend (in-memory only for simple JSON endpoint).
- No production build authenticity (dev server acceptable; we will also expose a static export mode later if needed).

---
## 3. Proposed Directory Layout
```
/apps/
  testsite/                # Bun React module root
    package.json
    bunfig.toml
    tsconfig.json
    README.md
    src/
      index.html           # Entry HTML (links to /static/* assets)
      main.tsx             # React root + router
      routes/
        index.tsx          # Home page with multi-depth internal links
        about.tsx
        blog/
          index.tsx        # Lists blog posts with relative links
          post-[id].tsx    # Dynamic route pages (pre-rendered array)
        labs/depth/depth2/depth3/leaf.tsx  # Deep chain for depth crawls
      components/
        Nav.tsx
        AssetGallery.tsx   # Emits <img>, <link>, <script>
      data/
        posts.json         # Static blog array
      api/
        server.ts          # Tiny Bun HTTP handler for /api/* JSON
    public/
      robots.txt           # Two modes (normal / disallow) toggled by env
      sitemap.xml          # Optional enhancement (Phase 2)
      static/
        styles.css
        script.js
        img/
          sample1.jpg
          sample2.png
    test-fixtures/
      snapshots/           # Golden copies of expected HTML for assertions
```

---
## 4. Module Metadata (package.json sketch)
```json
{
  "name": "@ariadne/testsite",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "bun run src/dev.ts",
    "start": "bun run src/dev.ts",
    "build": "bun run src/build.ts",  // (Phase 2: static export option)
    "lint": "eslint . --ext .ts,.tsx"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.23.0"
  },
  "devDependencies": {
    "typescript": "^5.5.0",
    "@types/react": "^18.2.0",
    "@types/react-dom": "^18.2.0",
    "eslint": "^9.0.0"
  }
}
```

---
## 5. Runtime Design
- **Port**: Default 5173 (configurable via `TESTSITE_PORT` env). Avoid collisions with existing metrics/health ports.
- **Single Bun process** launches both static file serving and React dev transform (simple custom script rather than full Vite to keep dependency graph tiny).
- Provide a lightweight router with pre-defined dynamic segments (no true server rendering required for test validity).

### API Endpoints
| Path          | Purpose                               |
|---------------|----------------------------------------|
| `/api/ping`   | Health style JSON – `{ "ok": true }` |
| `/api/posts`  | Serves `posts.json`                    |
| `/api/slow`   | Introduces 400–600ms latency window    |

### Asset Patterns
- `<img>` tags referencing `/static/img/sample1.jpg` and broken `/static/img/missing.png`.
- `<link rel="stylesheet">` to `/static/styles.css`.
- `<script src="/static/script.js" defer>` referencing a function that updates a DOM node (ensures crawler ignoring JS still sees original static content).

---
## 6. Test Integration Strategy
**New Go test helper** (`cli/internal/testsite` or `engine/internal/testutil/testsite`):
1. Detect existing running instance on configured port (reuse if present for local dev speed).
2. If absent: spawn `bun run --filter @ariadne/testsite start` (or direct `bun run src/dev.ts`) with `STDOUT` piped.
3. Wait for readiness line: `TESTSITE: listening on http://127.0.0.1:<port>` (timeout 5s).
4. Provide helper: `func WithLiveTestSite(t *testing.T, fn func(baseURL string))` that ensures cleanup (kill process) unless reuse enabled via `TESTSITE_REUSE=1`.
5. CLI integration tests override seeds: `--seeds http://127.0.0.1:<port>/` and assert crawl output includes expected discovered paths.

Assertions to add:
- Depth limiting: verify deep chain stops at configured depth.
- Broken asset: ensure error count increments but crawl continues.
- Robots gating: run once with normal `robots.txt`, once with disallow variant (env `TESTSITE_ROBOTS=deny`).
- Latency handling: `/api/slow` does not stall overall crawl longer than configured timeout.

---
## 7. Makefile / Automation Additions
Targets:
```
make testsite-dev        # Run dev server (foreground)
make testsite-check      # Lint + type check (optional Phase 2)
make integ-live          # Start site (background) + run selected integration tests
```
Implementation notes:
- Guard Bun availability: `command -v bun` else instruct install.
- For CI: add a job step installing Bun (curl installer) before integration tests.

---
## 8. Phased Delivery
| Phase | Scope | Deliverables |
|-------|-------|--------------|
| P1 | Minimal site + routes + assets + helper spawn | Directory skeleton, `dev.ts`, basic home/about/blog pages, test helper, single integration test using live site |
| P2 | Depth + latency + robots variants | Deep nested route, `/api/slow`, environment-driven `robots.txt`, assertions |
| P3 | Sitemap + structured data + static export | `sitemap.xml`, `<meta>` enrichment, build script producing `dist/` for future static mode |
| P4 | Edge cases & performance | Large page (≥50KB), many small assets, 404 coverage, concurrency stress test |

---
## 9. Risk & Mitigation
| Risk | Impact | Mitigation |
|------|--------|------------|
| Bun install flakiness in CI | Red pipeline | Cache bun install; pin version via `bunfig.toml` |
| Port conflicts | Test flakiness | Detect/choose random free port when default busy |
| Increasing test duration | Slower feedback | Keep site tiny; gate heavy scenarios behind tagged tests |
| Content drift breaking assertions | Flaky tests | Snapshot golden files; assert normalized HTML subsets |

---
## 10. Success Criteria
- Live-site integration test replaces at least one current mock-based test for discovery & asset metrics.
- Additional per-run time increase ≤ 2s locally / ≤ 5s in CI for P1.
- Deterministic (zero flakes over 20 consecutive CI runs).
- Provides at least: (a) ≥8 discoverable internal links, (b) ≥2 assets, (c) 1 broken reference, (d) 1 slow endpoint.

---
## 11. Open Questions
| Question | Resolution Path |
|----------|-----------------|
| Need JS rendering support tests? | Possibly later (headless browser integration) – out of scope P1-P2 |
| Serve gzip/brotli for compression behavior? | Only if we add response size budgeting tests |
| Multi-language pages? | Could add `/es/` subtree Phase 4 if needed |

---
## 12. Immediate Next Action (Execution Ticket P1)
1. Create `/apps/testsite` with skeleton (no build complexity).  
2. Implement minimal Bun dev script serving static + React routes (no HMR needed).  
3. Add Go helper to spawn & wait for readiness.  
4. Add new integration test leveraging helper and seeding crawler.  
5. Document usage in `md/live-test-site-plan.md` (this file) and cross-link from root README testing section.  
6. Add Makefile target `testsite-dev`.  

---
This plan is authoritative for initial introduction; refine only after P1 merged and timing impact measured.
