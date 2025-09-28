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

### Expanded Coverage (Wiki / Obsidian Quartz Parity)

Augment baseline goals so the site exercises a broad spectrum of elements commonly found in modern personal knowledge bases / docs sites:

- Rich typography set: semantic headings (h1–h6), paragraphs, blockquotes, horizontal rules.
- Code fences with language classes (```tsx, ```bash) + inline code.
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
        tags.json          # Tag metadata (Phase 2)
        search-index.json  # Lightweight search index (Phase 3)
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
          large-placeholder.jpg  # >150KB for stress / streaming test (Phase 4)
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

Assertions to add:

- Depth limiting: verify deep chain stops at configured depth.
- Broken asset: ensure error count increments but crawl continues.
- Robots gating: run once with normal `robots.txt`, once with disallow variant (env `TESTSITE_ROBOTS=deny`).
- Latency handling: `/api/slow` does not stall overall crawl longer than configured timeout.
- Admonitions rendered with proper role/aria labels.
- Code fences captured (ensure language attribute preserved) and not double-transformed.
- Footnotes produce backlinks (#fnref markers) and anchor traversal remains inside site.
- Tag pages enumerated and backlink lists generated (ensures multi-directional link graph coverage).
- Dark mode attribute does not produce duplicate “logical” pages (URL normalization test).
- Large asset fetch (Phase 4) does not block smaller fetch queue (throughput test).

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

| Phase | Scope                                         | Deliverables                                                                                                    |
| ----- | --------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| P1    | Minimal site + routes + basic styles + helper | Skeleton, Tailwind config, shadcn base (Button/Card), home/about/blog pages, helper & first integration test     |
| P2    | Depth + robots + latency + content richness   | Deep chain, `/api/slow`, robots modes, admonitions, code fences, footnotes, tags, dark mode, snapshot tooling    |
| P3    | Structured data & metadata & export           | `sitemap.xml`, search index JSON, meta/OG tags, build step (static export), compiled CSS stability               |
| P4    | Edge cases & performance                      | Large asset (>150KB), 404, concurrency stress, diagram/math placeholders, backlink graph validation              |
| P5    | Quality & Accessibility                       | a11y lint, color contrast check, ARIA roles on components, snapshot diff harness hardening                      |

---

## 9. Risk & Mitigation

| Risk                              | Impact          | Mitigation                                               |
| --------------------------------- | --------------- | -------------------------------------------------------- |
| Bun install flakiness in CI       | Red pipeline    | Cache bun install; pin version via `bunfig.toml`         |
| Port conflicts                    | Test flakiness  | Detect/choose random free port when default busy         |
| Increasing test duration          | Slower feedback | Keep site tiny; gate heavy scenarios behind tagged tests |
| Content drift breaking assertions | Flaky tests     | Snapshot golden files; assert normalized HTML subsets    |
| Tailwind stylesheet churn         | Spurious snapshot diffs | Pin Tailwind version + export compiled CSS hash, diff check |
| Dark mode state leakage           | Duplicate content set | Normalize theme attribute; treat as single logical page |
| Large image fetch slowdown        | Throughput regression | Parallel fetch cap + integration perf assertion |
| Increased bundle size             | Slower startup    | Keep dependency set tiny; avoid full shadcn tree imports |

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

## 12. Immediate Next Action (Execution Ticket P1)

1. Create `/apps/testsite` with skeleton (no build complexity).
2. Implement minimal Bun dev script serving static + React routes (no HMR needed) + Tailwind JIT (development only).
3. Add Go helper to spawn & wait for readiness.
4. Add new integration test leveraging helper and seeding crawler (assert link count, asset count, presence of broken image reference).
5. Document usage in `md/live-test-site-plan.md` (this file) and cross-link from root README testing section.
6. Add Makefile target `testsite-dev`.
7. Add basic shadcn components (Button, Card) and apply to home page.
8. Add Tailwind config with minimal safelist & deterministic class ordering (e.g. Preflight + chosen components only).

---

This plan is authoritative for initial introduction; refine only after P1 merged and timing impact measured.
