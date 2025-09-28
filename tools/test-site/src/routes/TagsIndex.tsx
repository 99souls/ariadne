import React from "react";
import { Link } from "react-router-dom";
import { Card } from "../components/ui/card";

export function TagsIndex() {
  const tagData = [
    {
      name: "announcement",
      count: 1,
      description: "Product announcements and major updates",
      color: "bg-blue-100 text-blue-800",
      posts: ["introducing-ariadne"],
    },
    {
      name: "architecture",
      count: 1,
      description: "System design and architectural decisions",
      color: "bg-green-100 text-green-800",
      posts: ["introducing-ariadne"],
    },
    {
      name: "performance",
      count: 1,
      description: "Performance optimization and benchmarking",
      color: "bg-red-100 text-red-800",
      posts: ["performance-optimization"],
    },
    {
      name: "technical",
      count: 1,
      description: "Technical deep-dives and implementation details",
      color: "bg-purple-100 text-purple-800",
      posts: ["performance-optimization"],
    },
    {
      name: "markdown",
      count: 1,
      description: "Markdown processing and content conversion",
      color: "bg-yellow-100 text-yellow-800",
      posts: ["markdown-processing"],
    },
    {
      name: "processing",
      count: 1,
      description: "Content processing and data transformation",
      color: "bg-indigo-100 text-indigo-800",
      posts: ["markdown-processing"],
    },
  ];

  const totalPosts = tagData.reduce((sum, tag) => sum + tag.count, 0);
  const popularTags = tagData
    .filter((tag) => tag.count >= 1)
    .sort((a, b) => b.count - a.count);

  return (
    <div className="space-y-8 max-w-4xl mx-auto">
      <div className="text-center space-y-4">
        <h1 className="text-4xl font-bold tracking-tight">Tags</h1>
        <p className="text-xl text-muted-foreground">
          Explore our content organized by topics and themes
        </p>
        <div className="flex justify-center space-x-6 text-sm text-muted-foreground">
          <span>{tagData.length} tags total</span>
          <span>•</span>
          <span>{totalPosts} posts tagged</span>
        </div>
      </div>

      {/* Tag Cloud */}
      <Card className="p-6">
        <h2 className="text-2xl font-bold mb-4">Popular Tags</h2>
        <div className="flex flex-wrap gap-3">
          {popularTags.map((tag) => (
            <Link
              key={tag.name}
              to={`/tags/${tag.name}`}
              className={`px-4 py-2 rounded-full text-sm font-medium transition-all hover:scale-105 hover:shadow-md ${tag.color}`}
            >
              #{tag.name} ({tag.count})
            </Link>
          ))}
        </div>
      </Card>

      {/* Detailed Tag List */}
      <div className="space-y-4">
        <h2 className="text-2xl font-bold">All Tags</h2>
        <div className="grid gap-4 md:grid-cols-2">
          {tagData.map((tag) => (
            <Card key={tag.name} className="p-4">
              <div className="flex items-start justify-between mb-3">
                <Link
                  to={`/tags/${tag.name}`}
                  className="text-lg font-semibold hover:text-blue-600 transition-colors"
                >
                  #{tag.name}
                </Link>
                <span className="text-sm text-muted-foreground bg-muted px-2 py-1 rounded">
                  {tag.count} post{tag.count !== 1 ? "s" : ""}
                </span>
              </div>

              <p className="text-muted-foreground text-sm mb-3">
                {tag.description}
              </p>

              <div className="space-y-2">
                <div className="text-sm font-medium">Recent posts:</div>
                {tag.posts.slice(0, 2).map((postSlug) => (
                  <Link
                    key={postSlug}
                    to={`/blog/${postSlug}`}
                    className="block text-sm text-blue-600 hover:underline"
                  >
                    {postSlug
                      .replace(/-/g, " ")
                      .replace(/\b\w/g, (l) => l.toUpperCase())}
                  </Link>
                ))}
              </div>
            </Card>
          ))}
        </div>
      </div>

      {/* Tag Statistics */}
      <Card className="p-6">
        <h2 className="text-2xl font-bold mb-4">Tag Statistics</h2>
        <div className="grid gap-4 md:grid-cols-3">
          <div className="text-center">
            <div className="text-3xl font-bold text-blue-600">
              {tagData.length}
            </div>
            <div className="text-sm text-muted-foreground">Total Tags</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-green-600">
              {totalPosts}
            </div>
            <div className="text-sm text-muted-foreground">Tagged Posts</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-purple-600">
              {Math.round((totalPosts / tagData.length) * 10) / 10}
            </div>
            <div className="text-sm text-muted-foreground">
              Avg Posts per Tag
            </div>
          </div>
        </div>
      </Card>

      {/* Search and Navigation */}
      <div className="bg-muted p-6 rounded-lg">
        <h3 className="text-lg font-semibold mb-4">Find Content</h3>
        <div className="space-y-4">
          <div className="relative">
            <input
              type="text"
              placeholder="Search tags..."
              className="w-full px-4 py-2 border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <div className="absolute right-3 top-2.5 text-muted-foreground">
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            <span className="text-sm text-muted-foreground">Quick access:</span>
            <Link to="/blog" className="text-sm text-blue-600 hover:underline">
              All Posts
            </Link>
            <span className="text-muted-foreground">•</span>
            <Link
              to="/docs/getting-started"
              className="text-sm text-blue-600 hover:underline"
            >
              Documentation
            </Link>
            <span className="text-muted-foreground">•</span>
            <Link to="/about" className="text-sm text-blue-600 hover:underline">
              About
            </Link>
          </div>
        </div>
      </div>

      {/* Browse by Category */}
      <div className="text-center space-y-4">
        <h3 className="text-lg font-semibold">Browse by Category</h3>
        <div className="flex flex-wrap justify-center gap-3">
          <Link
            to="/tags/technical"
            className="bg-purple-100 text-purple-800 px-4 py-2 rounded-full text-sm hover:bg-purple-200 transition-colors"
          >
            Technical Articles
          </Link>
          <Link
            to="/tags/announcement"
            className="bg-blue-100 text-blue-800 px-4 py-2 rounded-full text-sm hover:bg-blue-200 transition-colors"
          >
            Announcements
          </Link>
          <Link
            to="/tags/performance"
            className="bg-red-100 text-red-800 px-4 py-2 rounded-full text-sm hover:bg-red-200 transition-colors"
          >
            Performance
          </Link>
        </div>
      </div>
    </div>
  );
}
