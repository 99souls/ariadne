import React from "react";
import { Link } from "react-router-dom";

export function DeepLeafPage() {
  return (
    <div className="space-y-8 max-w-4xl mx-auto">
      <div className="text-center space-y-4">
        <h1 className="text-4xl font-bold tracking-tight">Deep Nested Leaf</h1>
        <p className="text-xl text-muted-foreground">
          You've reached the deepest level of our site structure
        </p>
      </div>

      {/* Breadcrumb */}
      <nav className="bg-muted p-4 rounded-lg" aria-label="breadcrumb">
        <ol className="flex flex-wrap items-center space-x-2 text-sm">
          <li>
            <Link to="/" className="text-blue-600 hover:underline">
              Home
            </Link>
          </li>
          <li className="text-muted-foreground">→</li>
          <li>
            <Link to="/labs" className="text-blue-600 hover:underline">
              Labs
            </Link>
          </li>
          <li className="text-muted-foreground">→</li>
          <li>
            <Link to="/labs/depth" className="text-blue-600 hover:underline">
              Depth
            </Link>
          </li>
          <li className="text-muted-foreground">→</li>
          <li>
            <Link
              to="/labs/depth/depth2"
              className="text-blue-600 hover:underline"
            >
              Depth2
            </Link>
          </li>
          <li className="text-muted-foreground">→</li>
          <li>
            <Link
              to="/labs/depth/depth2/depth3"
              className="text-blue-600 hover:underline"
            >
              Depth3
            </Link>
          </li>
          <li className="text-muted-foreground">→</li>
          <li className="text-muted-foreground">Leaf</li>
        </ol>
      </nav>

      {/* Path Information */}
      <div className="bg-blue-50 border border-blue-200 p-6 rounded-lg">
        <h2 className="text-2xl font-bold mb-4 text-blue-900">Path Analysis</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <div>
            <h3 className="font-semibold text-blue-800 mb-2">URL Structure</h3>
            <code className="block bg-white p-3 rounded border text-sm">
              /labs/depth/depth2/depth3/leaf
            </code>
            <p className="text-sm text-blue-700 mt-2">
              This URL has a depth of 5 levels, perfect for testing crawler
              depth limits.
            </p>
          </div>
          <div>
            <h3 className="font-semibold text-blue-800 mb-2">
              Crawler Testing
            </h3>
            <ul className="text-sm text-blue-700 space-y-1">
              <li>✓ Tests max depth configuration</li>
              <li>✓ Validates URL path handling</li>
              <li>✓ Checks link discovery at depth</li>
              <li>✓ Verifies content extraction</li>
            </ul>
          </div>
        </div>
      </div>

      {/* Content Sections */}
      <div className="prose max-w-none">
        <h2 className="text-2xl font-bold mb-4">Deep Content Testing</h2>

        <p className="mb-4">
          This page exists at a deep nesting level to test how web crawlers
          handle deeply nested content. It contains various elements that should
          be properly extracted and processed regardless of the URL depth.
        </p>

        <h3 className="text-xl font-semibold mb-3">Code Example at Depth</h3>
        <pre className="bg-muted p-4 rounded-lg overflow-x-auto mb-6">
          <code className="language-go">{`// Example: Depth-limited crawler configuration
type CrawlerConfig struct {
    MaxDepth     int    // Maximum crawl depth
    FollowDepth  int    // Maximum link follow depth  
    ContentDepth int    // Maximum content extraction depth
}

func (c *Crawler) shouldCrawl(url *url.URL, depth int) bool {
    return depth <= c.config.MaxDepth
}

// This function would return true for this page if MaxDepth >= 5`}</code>
        </pre>

        <h3 className="text-xl font-semibold mb-3">Table at Depth</h3>
        <div className="overflow-x-auto mb-6">
          <table className="min-w-full border-collapse border border-border">
            <thead>
              <tr className="bg-muted">
                <th className="border border-border p-3 text-left">
                  Depth Level
                </th>
                <th className="border border-border p-3 text-left">
                  Path Segment
                </th>
                <th className="border border-border p-3 text-left">Purpose</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td className="border border-border p-3">1</td>
                <td className="border border-border p-3 font-mono">/labs</td>
                <td className="border border-border p-3">
                  Laboratory section entry
                </td>
              </tr>
              <tr>
                <td className="border border-border p-3">2</td>
                <td className="border border-border p-3 font-mono">/depth</td>
                <td className="border border-border p-3">
                  Depth testing category
                </td>
              </tr>
              <tr>
                <td className="border border-border p-3">3</td>
                <td className="border border-border p-3 font-mono">/depth2</td>
                <td className="border border-border p-3">Second level depth</td>
              </tr>
              <tr>
                <td className="border border-border p-3">4</td>
                <td className="border border-border p-3 font-mono">/depth3</td>
                <td className="border border-border p-3">Third level depth</td>
              </tr>
              <tr>
                <td className="border border-border p-3">5</td>
                <td className="border border-border p-3 font-mono">/leaf</td>
                <td className="border border-border p-3">Terminal leaf node</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div
          className="border-l-4 border-yellow-500 bg-yellow-50 p-4 rounded-r-lg mb-6"
          role="alert"
        >
          <div className="font-semibold text-yellow-800">⚠️ Depth Warning</div>
          <div className="text-yellow-700">
            This page is at significant depth. Crawlers with limited depth
            settings may not reach this content. Consider your max depth
            configuration when crawling sites with deep hierarchies.
          </div>
        </div>

        <h3 className="text-xl font-semibold mb-3">Navigation Links</h3>
        <p className="mb-4">
          These links help test how crawlers handle navigation from deeply
          nested pages:
        </p>

        <ul className="list-disc ml-6 space-y-2 mb-6">
          <li>
            <Link to="/" className="text-blue-600 hover:underline">
              Back to Root
            </Link>{" "}
            (5 levels up)
          </li>
          <li>
            <Link to="/about" className="text-blue-600 hover:underline">
              About Page
            </Link>{" "}
            (cross-section navigation)
          </li>
          <li>
            <Link
              to="/docs/getting-started"
              className="text-blue-600 hover:underline"
            >
              Documentation
            </Link>{" "}
            (different hierarchy)
          </li>
          <li>
            <Link to="/blog" className="text-blue-600 hover:underline">
              Blog Index
            </Link>{" "}
            (lateral navigation)
          </li>
        </ul>

        <h3 className="text-xl font-semibold mb-3">Assets at Depth</h3>
        <div className="grid gap-4 md:grid-cols-2 mb-6">
          <div>
            <h4 className="font-medium mb-2">Working Image</h4>
            <img
              src="/static/img/sample1.svg"
              alt="Sample image for deep page testing"
              className="w-full h-32 object-cover rounded border"
            />
            <p className="text-sm text-muted-foreground mt-1">
              Image loading test at depth level 5
            </p>
          </div>
          <div>
            <h4 className="font-medium mb-2">Broken Image (404 Test)</h4>
            <img
              src="/static/img/deep-missing.png"
              alt="Intentionally broken image for 404 testing"
              className="w-full h-32 object-cover rounded border"
            />
            <p className="text-sm text-muted-foreground mt-1">
              404 handling test at depth level 5
            </p>
          </div>
        </div>

        <h3 className="text-xl font-semibold mb-3">Content Extraction Test</h3>
        <p className="mb-4">
          This paragraph contains various inline elements that should be
          properly extracted: <strong>bold text</strong>, <em>italic text</em>,
          <code className="bg-muted px-1 rounded">inline code</code>, and
          <a href="/about" className="text-blue-600 hover:underline">
            internal links
          </a>
          .
        </p>

        <blockquote className="border-l-4 border-muted pl-4 italic text-muted-foreground mb-6">
          "The depth of your content structure should never compromise the
          quality of your information architecture." - Web Design Principles
        </blockquote>

        <h3 className="text-xl font-semibold mb-3">Footnotes at Depth</h3>
        <p className="mb-4">
          Deep content can still reference footnotes
          <sup>
            <a href="#fn-deep1" id="fnref-deep1" className="text-blue-600">
              1
            </a>
          </sup>
          and maintain proper linking
          <sup>
            <a href="#fn-deep2" id="fnref-deep2" className="text-blue-600">
              2
            </a>
          </sup>
          .
        </p>

        <hr className="my-8" />

        <div className="text-sm space-y-2">
          <div id="fn-deep1" className="flex">
            <a href="#fnref-deep1" className="text-blue-600 mr-2">
              1.
            </a>
            <span>
              Footnotes should work correctly regardless of URL depth or nesting
              level.
            </span>
          </div>
          <div id="fn-deep2" className="flex">
            <a href="#fnref-deep2" className="text-blue-600 mr-2">
              2.
            </a>
            <span>
              This tests anchor link functionality in deeply nested content
              structures.
            </span>
          </div>
        </div>
      </div>

      {/* Navigation Help */}
      <div className="bg-muted p-6 rounded-lg text-center">
        <h3 className="text-lg font-semibold mb-2">Quick Navigation</h3>
        <p className="text-muted-foreground mb-4">
          Use these links to navigate back to main sections:
        </p>
        <div className="space-x-4">
          <Link to="/" className="text-blue-600 hover:underline">
            Home
          </Link>
          <span className="text-muted-foreground">•</span>
          <Link to="/about" className="text-blue-600 hover:underline">
            About
          </Link>
          <span className="text-muted-foreground">•</span>
          <Link
            to="/docs/getting-started"
            className="text-blue-600 hover:underline"
          >
            Docs
          </Link>
          <span className="text-muted-foreground">•</span>
          <Link to="/blog" className="text-blue-600 hover:underline">
            Blog
          </Link>
          <span className="text-muted-foreground">•</span>
          <Link to="/tags" className="text-blue-600 hover:underline">
            Tags
          </Link>
        </div>
      </div>
    </div>
  );
}
