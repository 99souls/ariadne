# Live Test Site Module Plan (Bun + React)

Status: **IN PROGRESS** (Robots, Snapshot diff, Depth limiting, Broken asset, Slow endpoint tests added)
Owner: Core / Testing
Purpose: Introduce a lightweight, deterministic, zero-external-dependency web application inside the monorepo that the engine + CLI integration tests can crawl. This replaces/augments current synthetic mocks with a realistic HTML/CSS/asset/SPA surface.

## ✅ COMPLETED WORK (Phase 1)

**Implementation Date: 2025-09-29**

### Core Infrastructure ✅

- **Package Structure**: `tools/test-site` with proper `@ariadne/testsite` package name
- **Bun + React Setup**: Complete with TypeScript, Tailwind CSS, shadcn/ui components
- **Dev Server**: Custom Bun server with TypeScript transpilation via Bun.build()
- **Module Resolution**: Fixed import paths from alias (@/) to relative paths, added file extensions
- **Static Assets**: CSS, JavaScript, SVG images, intentionally missing assets for 404 testing

### Routes Implemented ✅

- `/` - HomePage with comprehensive wiki-style content
- `/about` - AboutPage with feature cards, metadata, architecture diagrams
- `/blog` - Blog index with post listings
- `/blog/post-{id}` - Individual blog posts
- `/docs/getting-started` - Documentation page
- `/docs/deep/n1/n2/n3/leaf` - Deep nested route for depth testing (legacy path)
- `/labs/depth/depth2/depth3/leaf` - New deep nested route used by depth limiting integration test ✅
- `/tags` - Tag index page

### API Endpoints ✅

- `/api/ping` - Health check endpoint
- `/api/posts` - Static JSON post data
- `/api/slow` - Latency injection endpoint (400-600ms delay)

### Content Features ✅

- **Rich Typography**: Semantic headings (h1-h6), paragraphs, blockquotes, horizontal rules
- **Code Examples**: TypeScript and bash code fences with syntax highlighting
- **Admonitions**: Warning/note callouts using shadcn Alert components
- **Tables**: Feature comparison and metadata tables
- **Mathematical Notation**: Inline and block math placeholders
- **Diagrams**: ASCII art architecture diagrams in `<pre data-type="diagram">`
- **Metadata**: Page frontmatter simulation with author, date, tags
- **Assets**: Local images, broken image references, CSS/JS loading
- **Responsive Design**: Mobile-friendly layout with Tailwind utilities

### Technical Implementation ✅

- **TypeScript Transpilation**: Bun.build() with ESM format, browser target
- **Static File Serving**: Proper Content-Type headers for all asset types
- **Robots.txt**: Dynamic robots.txt based on TESTSITE_ROBOTS env variable
- **Sitemap.xml**: Complete sitemap for all discoverable routes
- **Port Configuration**: Configurable via TESTSITE_PORT (default 5173)
- **Startup Banner**: `TESTSITE: listening on...` for test harness readiness detection
- **Error Handling**: Proper 404s for missing assets, JSON error responses

### Quality Assurance ✅

- **Import Resolution**: All TypeScript imports working correctly in browser
- **Content Rendering**: React components mounting and rendering properly
- **Asset Loading**: CSS styles applied, images loading, broken links generating expected 404s
- **API Functionality**: All endpoints returning correct JSON responses
- **Performance**: Fast startup (<200ms on modern hardware)
- **Deterministic Content**: No timestamp/random content, stable for testing

---

## 1. Goals

- Provide a **real HTTP origin** with realistic assets, nested routes, metadata, and failure cases.
- Deterministic content (no external API calls / time-based randomness) for stable assertions.
- Very fast startup (<200ms cold) to avoid test overhead.
- Exercise crawler behaviors: link discovery, depth, robots.txt handling, static vs dynamic content.
- Serve a mix of content types: HTML pages, markdown-like embedded sections, images, stylesheets, JSON API endpoint, and intentionally broken links.
- Allow toggling feature flags (query param or header) to test conditional rendering.

### Expanded Coverage (Wiki / Obsidian Quartz Parity)

Augment baseline goals so the site exercises a broad spectrum of elements commonly found in modern personal knowledge bases / docs sites:

- Rich typography set: semantic headings (h1–h6), paragraphs, blockquotes, horizontal rules.
- Code fences with language classes (`tsx, `bash) + inline code.
- Admonitions / callouts (e.g. NOTE, WARNING, TIP) styled via shadcn components.
- Tables (alignment variations) + definition lists.
- Footnotes / reference style links + internal anchor (#section) navigation.
- Tag pages (e.g. /tags/{tag}) + backlinks section on each note.
- Frontmatter simulation (YAML block at top) rendered as a metadata table.
- Mermaid / diagram placeholder blocks (static <pre data-type="diagram">) for parser future-proofing.
- Math (inline $x$ and block $$E=mc^2$$) placeholder nodes for later rendering path.
- Images (local, broken, SVG) plus responsive <picture> example.
- Internal search index JSON (later phase) to test ignoring non-page API endpoints.
- Dark mode toggle + persisted theme attribute (`data-theme` on <html>) to test attribute-based variant scanning.

## 2. Non-Goals

- No SSR complexity beyond basic React hydration.
- No DB / stateful backend (in-memory only for simple JSON endpoint).
- No production build authenticity (dev server acceptable; we will also expose a static export mode later if needed).

---

## 3. ✅ IMPLEMENTED Directory Layout

```
tools/test-site/           # ✅ Bun React module root (IMPLEMENTED)
  package.json             # ✅ @ariadne/testsite package
  bunfig.toml              # ✅ Bun configuration
  tsconfig.json            # ✅ TypeScript config
  README.md                # ✅ Documentation
  src/
    index.html             # ✅ Entry HTML template
    main.tsx               # ✅ React root + React Router setup
    App.tsx                # ✅ Main app component with routing
    dev.ts                 # ✅ Custom Bun dev server with TS transpilation
    routes/                # ✅ Route components
      HomePage.tsx         # ✅ Home page with comprehensive content
      AboutPage.tsx        # ✅ About page with feature cards
      BlogIndex.tsx        # ✅ Blog listing page
      BlogPost.tsx         # ✅ Individual blog post component
      DocsGettingStarted.tsx # ✅ Documentation page
      DeepLeafPage.tsx     # ✅ Deep nested route (depth testing)
      TagsIndex.tsx        # ✅ Tags listing page
    components/            # ✅ Shared UI components
      ui/                  # ✅ shadcn/ui components (Card, Button, Alert)
        card.tsx           # ✅ Card component
        button.tsx         # ✅ Button component
        alert.tsx          # ✅ Alert/admonition component
      Navigation.tsx       # ✅ Site navigation component
  public/                  # ✅ Static assets
    robots.txt             # ✅ Dynamic robots (allow/disallow via env)
    sitemap.xml            # ✅ Complete sitemap
    static/                # ✅ Asset directory
      styles.css           # ✅ Additional CSS
      script.js            # ✅ Client-side JavaScript
      img/                 # ✅ Images directory
        sample1.svg        # ✅ Working image asset
        sample2.svg        # ✅ Working image asset
        missing.png        # ⚠️  Intentionally missing (404 test)
        deep-missing.png   # ⚠️  Intentionally missing (404 test)
  styles/                  # ✅ Tailwind CSS configuration
    globals.css            # ✅ Global styles with Tailwind imports
```

**Status Legend:**

- ✅ Fully implemented and working
- ⚠️ Intentionally missing/broken for testing purposes
- 📋 Planned for future phases

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
    "build": "bun run src/build.ts", // (Phase 2: static export option)
    "lint": "eslint . --ext .ts,.tsx"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.23.0",
    "clsx": "^2.1.0"
  },
  "devDependencies": {
    "typescript": "^5.5.0",
    "@types/react": "^18.2.0",
    "@types/react-dom": "^18.2.0",
    "eslint": "^9.0.0",
    "tailwindcss": "^3.4.0",
    "postcss": "^8.4.0",
    "autoprefixer": "^10.4.0"
  }
}
```

---

## 5. Runtime Design

- **Port**: Default 5173 (configurable via `TESTSITE_PORT` env). Avoid collisions with existing metrics/health ports.
- **Single Bun process** launches both static file serving and React dev transform (simple custom script rather than full Vite to keep dependency graph tiny).
- Provide a lightweight router with pre-defined dynamic segments (no true server rendering required for test validity).

### API Endpoints

| Path         | Purpose                              |
| ------------ | ------------------------------------ |
| `/api/ping`  | Health style JSON – `{ "ok": true }` |
| `/api/posts` | Serves `posts.json`                  |
| `/api/slow`  | Introduces 400–600ms latency window  |

### Asset Patterns

- `<img>` tags referencing `/static/img/sample1.jpg` and broken `/static/img/missing.png`.
- `<link rel="stylesheet">` to `/static/styles.css`.
- `<script src="/static/script.js" defer>` referencing a function that updates a DOM node (ensures crawler ignoring JS still sees original static content).
- Tailwind + shadcn component styles (Button, Card, Alert, Tabs) to ensure style link + class scanning behaves and to stress test HTML/class attribute parsing.
- Dark mode variant (`class="dark"` on `<html>`) via query param `?theme=dark` for snapshot diff normalization.

---

## 6. Test Integration Strategy

**New Go test helper** (`cli/internal/testsite` or `engine/internal/testutil/testsite`):

1. Detect existing running instance on configured port (reuse if present for local dev speed).
2. If absent: spawn `bun run --filter @ariadne/testsite start` (or direct `bun run src/dev.ts`) with `STDOUT` piped.
3. Wait for readiness line: `TESTSITE: listening on http://127.0.0.1:<port>` (timeout 5s).
4. Provide helper: `func WithLiveTestSite(t *testing.T, fn func(baseURL string))` that ensures cleanup (kill process) unless reuse enabled via `TESTSITE_REUSE=1`.
5. CLI integration tests override seeds: `--seeds http://127.0.0.1:<port>/` and assert crawl output includes expected discovered paths.

Assertions implemented:

- Depth limiting: MaxDepth=4 excludes depth-5 leaf ✅ (`TestLiveSiteDepthLimit`)
- Robots gating allow/deny ✅ (`TestLiveSiteDiscovery`, `TestLiveSiteRobotsDeny`)
- Broken asset surfaced as asset result with >=400 status ✅ (`TestLiveSiteBrokenAsset`)
- Slow endpoint latency non-blocking ✅ (`TestLiveSiteSlowEndpoint`)
- Snapshot diff enforcement ✅ (`TestGenerateSnapshots` with drift check)

Remaining assertions to add:

- Admonitions rendered with proper role/aria labels.
- Code fences captured (language class preserved) and not double-transformed.
- Footnotes produce backlinks (#fnref markers) and anchor traversal constrained to site.
- Tag pages enumerated + backlink lists generated.
- Dark mode attribute normalization (no duplicate logical pages).
- Large asset fetch (Phase 4) throughput / non-blocking test.

---

## 7. Makefile / Automation Additions

Targets:

```
make testsite-dev        # Run dev server (foreground)
make testsite-check      # Lint + type check (optional Phase 2)
make integ-live          # Start site (background) + run selected integration tests
make testsite-snapshots  # (Phase 2+) Regenerate normalized HTML golden snapshots
```

Implementation notes:

- Guard Bun availability: `command -v bun` else instruct install.
- For CI: add a job step installing Bun (curl installer) before integration tests.
- Tailwind build: generate a minimal static CSS (Phase 1: dev JIT acceptable; Phase 2+: compile deterministic stylesheet for snapshot stability, store hash in test fixtures to detect accidental style drift).
- Snapshot normalization script: strip dynamic attributes (data-reactroot, hashed class names if any), unify whitespace, remove `nonce`/`data-theme` variance if needed.

---

## 8. Phased Delivery

| Phase | Scope                                         | Deliverables                                                                                                  |
| ----- | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| P1    | Minimal site + routes + basic styles + helper | Skeleton, Tailwind config, shadcn base (Button/Card), home/about/blog pages, helper & first integration test  |
| P2    | Depth + robots + latency + content richness   | Deep chain, `/api/slow`, robots modes, admonitions, code fences, footnotes, tags, dark mode, snapshot tooling |
| P3    | Structured data & metadata & export           | `sitemap.xml`, search index JSON, meta/OG tags, build step (static export), compiled CSS stability            |
| P4    | Edge cases & performance                      | Large asset (>150KB), 404, concurrency stress, diagram/math placeholders, backlink graph validation           |
| P5    | Quality & Accessibility                       | a11y lint, color contrast check, ARIA roles on components, snapshot diff harness hardening                    |

---

## 9. Risk & Mitigation

| Risk                              | Impact                  | Mitigation                                                  |
| --------------------------------- | ----------------------- | ----------------------------------------------------------- |
| Bun install flakiness in CI       | Red pipeline            | Cache bun install; pin version via `bunfig.toml`            |
| Port conflicts                    | Test flakiness          | Detect/choose random free port when default busy            |
| Increasing test duration          | Slower feedback         | Keep site tiny; gate heavy scenarios behind tagged tests    |
| Content drift breaking assertions | Flaky tests             | Snapshot golden files; assert normalized HTML subsets       |
| Tailwind stylesheet churn         | Spurious snapshot diffs | Pin Tailwind version + export compiled CSS hash, diff check |
| Dark mode state leakage           | Duplicate content set   | Normalize theme attribute; treat as single logical page     |
| Large image fetch slowdown        | Throughput regression   | Parallel fetch cap + integration perf assertion             |
| Increased bundle size             | Slower startup          | Keep dependency set tiny; avoid full shadcn tree imports    |

---

## 10. Success Criteria

- Live-site integration test replaces at least one current mock-based test for discovery & asset metrics.
- Additional per-run time increase ≤ 2s locally / ≤ 5s in CI for P1.
- Deterministic (zero flakes over 20 consecutive CI runs).
- Provides at least: (a) ≥8 discoverable internal links, (b) ≥2 assets, (c) 1 broken reference, (d) 1 slow endpoint.
- P2 adds: ≥3 admonitions, ≥2 code fences (different languages), ≥1 table, ≥1 footnote, ≥1 tag index, dark mode toggle.
- P3 adds: sitemap coverage rate == 100% of HTML pages & search index parity check.
- P5 adds: a11y basic checks pass (headings order, alt text coverage ≥95%).

---

## 11. Open Questions

| Question                                    | Resolution Path                                                    |
| ------------------------------------------- | ------------------------------------------------------------------ |
| Need JS rendering support tests?            | Possibly later (headless browser integration) – out of scope P1-P2 |
| Serve gzip/brotli for compression behavior? | Only if we add response size budgeting tests                       |
| Multi-language pages?                       | Could add `/es/` subtree Phase 4 if needed                         |

---

## 📋 PENDING WORK (Phase 2 – Updated)

### Go Test Harness Integration (Completed)

- [x] **Helper Function** implemented (`WithLiveTestSite`)
- [x] **Process Management** with reuse via TESTSITE_REUSE=1
- [x] **Port Handling** ephemeral selection
- [x] **Readiness Detection** via banner + health check
- [x] **Graceful Cleanup** (skip on reuse)
- [ ] **CI Reuse Validation** job (pending)

### Integration Test Implementation (Updated)

- [x] **Replace Mock Tests**: Discovery test in place
- [x] **Core Discovery Assertion** (multi-page set)
- [x] **Asset Counting / Broken Image Tracking** test (basic broken asset assertion)
- [x] **Depth Limiting** test
- [x] **Robots Allow/Deny** tests
- [x] **Slow Endpoint Non-Blocking** test
- [x] **Golden Snapshot Generation** (non-enforcing)
- [x] **Snapshot Diff Enforcement** (with UPDATE_SNAPSHOTS)

### Enhanced Content Features

- [ ] **Dark Mode**: Toggle via query param `?theme=dark`, test attribute-based scanning
- [ ] **Search Index**: `/api/search.json` endpoint for testing API endpoint ignoring
- [ ] **Enhanced Metadata**: More comprehensive OpenGraph tags, structured data
- [ ] **Performance Assets**: Large image (>150KB) for streaming/memory tests

### Makefile & Docs Integration

- [x] **Targets**: `testsite-dev`, `integ-live`, `testsite-check`, `testsite-snapshots`
- [ ] **CI Integration**: Bun install + caching + reuse step
- [ ] **Documentation**: Root README “Live Test Site Usage” section
- [ ] **Snapshot Update Workflow Docs**

### Additional Near-Term Tasks

- [x] Robots enforcement logic in crawler (respect override flag)
- [ ] Flake detector script (loop integration test N times)
- [ ] Depth/latency metrics instrumentation (optional lightweight timers)

## 12. ✅ COMPLETED PHASE 1 WORK

**All Phase 1 objectives achieved (2025-09-29):**

1. ✅ **Created** `tools/test-site` with complete package structure
2. ✅ **Implemented** Bun dev server with TypeScript transpilation via Bun.build()
3. ✅ **Added** comprehensive React routes with wiki-style content
4. ✅ **Integrated** shadcn/ui components (Button, Card, Alert) throughout site
5. ✅ **Configured** Tailwind CSS with proper component styling
6. ✅ **Established** API endpoints (/api/ping, /api/posts, /api/slow)
7. ✅ **Created** realistic content with rich typography, code fences, admonitions, tables
8. ✅ **Added** deterministic robots.txt and sitemap.xml generation

### Next Immediate Actions (Phase 2 Start)

**Priority 1**: Go test harness implementation
**Priority 2**: Integration test migration  
**Priority 3**: Makefile targets and CI integration
**Priority 4**: Enhanced content features for comprehensive crawler testing

---

This plan reflects **Phase 1 completion** as of 2025-09-29. Phase 2 work should begin with Go test harness implementation to enable integration testing.
