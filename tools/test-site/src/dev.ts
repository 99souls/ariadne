import { serve, file } from "bun";
import { resolve, join } from "path";

const port = parseInt(process.env.TESTSITE_PORT || "5173");
const robotsMode = process.env.TESTSITE_ROBOTS || "allow";

// Static data for API endpoints
const postsData = [
  {
    id: "getting-started",
    title: "Getting Started with Ariadne",
    excerpt:
      "Learn how to set up and use Ariadne for your documentation needs.",
    date: "2024-01-15",
    tags: ["setup", "documentation"],
  },
  {
    id: "advanced-features",
    title: "Advanced Features",
    excerpt: "Explore the advanced capabilities of Ariadne.",
    date: "2024-01-20",
    tags: ["advanced", "features"],
  },
];

// Per-process instance identity so Go test harness can verify reuse semantics.
const instanceId = (() => {
  // Use high-resolution time plus random component; determinism across a single process lifetime only.
  const t = Date.now().toString(36);
  const r = Math.random().toString(36).slice(2, 10);
  return `${t}-${r}`;
})();
const startedAt = new Date().toISOString();

const server = serve({
  port,
  async fetch(req) {
    const url = new URL(req.url);
    const pathname = url.pathname;

    // API endpoints
    if (pathname === "/api/ping") {
      return Response.json({ ok: true, timestamp: "2024-01-01T00:00:00Z" }); // Fixed timestamp for determinism
    }

    if (pathname === "/api/posts") {
      return Response.json(postsData);
    }

    if (pathname === "/api/slow") {
      // Deterministic delay - always 500ms for consistent testing
      await new Promise((resolve) => setTimeout(resolve, 500));
      return Response.json({
        message: "This endpoint is intentionally slow",
        delay: 500,
      });
    }

    if (pathname === "/api/instance") {
      return Response.json({ id: instanceId, startedAt });
    }

    // Search index endpoint (intentionally ignored by crawler as a non-page JSON surface).
    if (pathname === "/api/search.json") {
      const searchIndex = {
        version: 1,
        generatedAt: "2024-01-01T00:00:00Z",
        entries: [
          { url: "/", title: "Home" },
          { url: "/about", title: "About" },
          { url: "/docs/getting-started", title: "Getting Started" },
        ],
      };
      return Response.json(searchIndex, {
        headers: { "Cache-Control": "no-store" },
      });
    }

    // Dynamic robots.txt based on environment
    if (pathname === "/robots.txt") {
      const robotsContent =
        robotsMode === "deny"
          ? "User-agent: *\nDisallow: /"
          : "User-agent: *\nAllow: /\n\nSitemap: http://localhost:" +
            port +
            "/sitemap.xml";
      return new Response(robotsContent, {
        headers: { "Content-Type": "text/plain" },
      });
    }

    // Sitemap.xml
    if (pathname === "/sitemap.xml") {
      const sitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>http://localhost:${port}/</loc><changefreq>daily</changefreq></url>
  <url><loc>http://localhost:${port}/about</loc><changefreq>weekly</changefreq></url>
  <url><loc>http://localhost:${port}/docs/getting-started</loc><changefreq>weekly</changefreq></url>
  <url><loc>http://localhost:${port}/blog/post-1</loc><changefreq>monthly</changefreq></url>
  <url><loc>http://localhost:${port}/tags</loc><changefreq>weekly</changefreq></url>
</urlset>`;
      return new Response(sitemap, {
        headers: { "Content-Type": "application/xml" },
      });
    }

    // Static assets
    if (pathname.startsWith("/static/")) {
      const filePath = join(process.cwd(), "public", pathname);
      try {
        return new Response(file(filePath));
      } catch {
        return new Response("Not Found", { status: 404 });
      }
    }

    // Handle TypeScript/JavaScript modules
    if (
      pathname.endsWith(".tsx") ||
      pathname.endsWith(".ts") ||
      pathname.endsWith(".jsx") ||
      pathname.endsWith(".js")
    ) {
      const filePath = resolve(process.cwd(), "src", pathname.slice(1));
      try {
        const result = await Bun.build({
          entrypoints: [filePath],
          format: "esm",
          target: "browser",
        });

        if (result.success && result.outputs[0]) {
          const jsContent = await result.outputs[0].text();
          return new Response(jsContent, {
            headers: { "Content-Type": "application/javascript" },
          });
        } else {
          console.error("Build failed:", result.logs);
          return new Response("Build Error", { status: 500 });
        }
      } catch (error) {
        console.error("Module error:", error);
        return new Response("Module Not Found", { status: 404 });
      }
    }

    // Serve React app for all other routes
    const indexPath = resolve(process.cwd(), "src/index.html");
    return new Response(file(indexPath), {
      headers: { "Content-Type": "text/html" },
    });
  },
  development: process.env.NODE_ENV !== "production",
});

// Deterministic startup message for test harness detection
console.log(`TESTSITE: listening on http://127.0.0.1:${port}`);
console.log(`Robots mode: ${robotsMode}`);
