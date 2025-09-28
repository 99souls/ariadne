import React from "react";
import { Link } from "react-router-dom";
import { Card } from "../components/ui/card";

export function BlogIndex() {
  const blogPosts = [
    {
      slug: "introducing-ariadne",
      title: "Introducing Ariadne: The Web Documentation Crawler",
      excerpt:
        "Learn about the motivation and architecture behind Ariadne, our new tool for extracting and converting web documentation.",
      date: "2024-01-15",
      author: "Development Team",
      tags: ["announcement", "architecture"],
      readTime: "5 min read",
    },
    {
      slug: "performance-optimization",
      title: "Performance Optimization in Web Crawling",
      excerpt:
        "Deep dive into the techniques we use to achieve high-performance crawling while respecting rate limits and server resources.",
      date: "2024-01-08",
      author: "Performance Team",
      tags: ["performance", "technical"],
      readTime: "8 min read",
    },
    {
      slug: "markdown-processing",
      title: "Advanced Markdown Processing Techniques",
      excerpt:
        "Explore how Ariadne handles complex markdown processing, including code blocks, tables, and custom extensions.",
      date: "2024-01-01",
      author: "Content Team",
      tags: ["markdown", "processing"],
      readTime: "6 min read",
    },
  ];

  return (
    <div className="space-y-8 max-w-4xl mx-auto">
      <div className="text-center space-y-4">
        <h1 className="text-4xl font-bold tracking-tight">Blog</h1>
        <p className="text-xl text-muted-foreground">
          Insights, updates, and technical deep-dives from the Ariadne team
        </p>
      </div>

      <div className="grid gap-6">
        {blogPosts.map((post) => (
          <Card key={post.slug} className="p-6">
            <article>
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center space-x-2 text-sm text-muted-foreground">
                  <time dateTime={post.date}>
                    {new Date(post.date).toLocaleDateString("en-US", {
                      year: "numeric",
                      month: "long",
                      day: "numeric",
                    })}
                  </time>
                  <span>•</span>
                  <span>{post.author}</span>
                  <span>•</span>
                  <span>{post.readTime}</span>
                </div>
              </div>

              <Link to={`/blog/${post.slug}`} className="group">
                <h2 className="text-2xl font-bold mb-3 group-hover:text-blue-600 transition-colors">
                  {post.title}
                </h2>
              </Link>

              <p className="text-muted-foreground mb-4 leading-relaxed">
                {post.excerpt}
              </p>

              <div className="flex items-center justify-between">
                <div className="flex flex-wrap gap-2">
                  {post.tags.map((tag) => (
                    <Link
                      key={tag}
                      to={`/tags/${tag}`}
                      className="bg-muted hover:bg-muted/80 px-3 py-1 rounded-full text-sm transition-colors"
                    >
                      #{tag}
                    </Link>
                  ))}
                </div>

                <Link
                  to={`/blog/${post.slug}`}
                  className="text-blue-600 hover:underline text-sm font-medium"
                >
                  Read more →
                </Link>
              </div>
            </article>
          </Card>
        ))}
      </div>

      <div className="bg-muted p-6 rounded-lg text-center">
        <h3 className="text-lg font-semibold mb-2">Stay Updated</h3>
        <p className="text-muted-foreground mb-4">
          Subscribe to our newsletter for the latest updates and insights.
        </p>
        <div className="flex max-w-md mx-auto">
          <input
            type="email"
            placeholder="Enter your email"
            className="flex-1 px-4 py-2 border border-border rounded-l-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button className="bg-blue-600 text-white px-6 py-2 rounded-r-md hover:bg-blue-700 transition-colors">
            Subscribe
          </button>
        </div>
        <p className="text-xs text-muted-foreground mt-2">
          We respect your privacy. Unsubscribe at any time.
        </p>
      </div>

      <div className="text-center">
        <p className="text-muted-foreground mb-4">Explore more content:</p>
        <div className="space-x-4">
          <Link to="/tags" className="text-blue-600 hover:underline">
            Browse All Tags
          </Link>
          <span className="text-muted-foreground">•</span>
          <Link
            to="/docs/getting-started"
            className="text-blue-600 hover:underline"
          >
            Documentation
          </Link>
          <span className="text-muted-foreground">•</span>
          <Link to="/about" className="text-blue-600 hover:underline">
            About Us
          </Link>
        </div>
      </div>
    </div>
  );
}
