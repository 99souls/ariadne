import React from "react";
import { useParams, Link } from "react-router-dom";
import { Card } from "../components/ui/card";

export function BlogPost() {
  const { slug } = useParams<{ slug: string }>();

  // Simulate blog post data based on slug
  const getBlogPost = (slug: string) => {
    const posts: Record<string, any> = {
      "introducing-ariadne": {
        title: "Introducing Ariadne: The Web Documentation Crawler",
        date: "2024-01-15",
        author: "Development Team",
        readTime: "5 min read",
        tags: ["announcement", "architecture"],
        content: `
# Introducing Ariadne

We're excited to announce **Ariadne**, a powerful web crawler designed specifically for extracting and converting web documentation into structured formats.

## The Problem

Web documentation is scattered across countless sites, each with different structures, styling, and organization. Developers often need to:

- Extract documentation from legacy sites
- Migrate content between platforms
- Archive important technical knowledge
- Create offline documentation bundles

## Our Solution

Ariadne addresses these challenges with:

### 🚀 High Performance
Built in Go with concurrent processing and intelligent rate limiting.

\`\`\`go
type CrawlerConfig struct {
    MaxDepth     int
    Workers      int 
    RateLimit    time.Duration
    OutputFormat string
}
\`\`\`

### 🎯 Smart Content Extraction  
Advanced algorithms to identify and extract meaningful content while filtering noise.

### 📝 Multiple Output Formats
Support for Markdown, HTML, and PDF output formats.

## Architecture Overview

Ariadne follows a pipeline architecture:

1. **URL Discovery** - Find and queue URLs to crawl
2. **Content Fetching** - Download pages with rate limiting
3. **Content Processing** - Extract and clean content
4. **Format Conversion** - Convert to target format
5. **Output Generation** - Write structured files

## Getting Started

Install Ariadne and start crawling:

\`\`\`bash
go install github.com/99souls/ariadne@latest
ariadne crawl --seeds https://docs.example.com
\`\`\`

## What's Next

We're actively working on:
- Enhanced content detection algorithms
- Plugin system for custom processors  
- Web UI for crawl management
- Cloud deployment options

Stay tuned for more updates!`,
      },
      "performance-optimization": {
        title: "Performance Optimization in Web Crawling",
        date: "2024-01-08",
        author: "Performance Team",
        readTime: "8 min read",
        tags: ["performance", "technical"],
        content: `
# Performance Optimization in Web Crawling

Web crawling at scale requires careful attention to performance. Here's how we optimized Ariadne for high-throughput crawling.

## The Challenge

Modern websites can have thousands of pages. Crawling them efficiently requires:

- **Concurrent Processing** - Multiple workers processing URLs simultaneously
- **Rate Limiting** - Respecting server resources and avoiding blocks
- **Memory Management** - Handling large crawls without exhausting system resources
- **Network Efficiency** - Minimizing redundant requests and bandwidth usage

## Our Approach

### Worker Pool Architecture

We implemented a worker pool pattern with configurable concurrency:

\`\`\`go
type WorkerPool struct {
    workers   int
    jobs      chan CrawlJob
    results   chan CrawlResult
    semaphore chan struct{}
}

func (wp *WorkerPool) Start() {
    for i := 0; i < wp.workers; i++ {
        go wp.worker(i)
    }
}
\`\`\`

### Intelligent Rate Limiting

Our rate limiter uses a token bucket algorithm with per-domain tracking:

\`\`\`go
type DomainLimiter struct {
    limits map[string]*TokenBucket
    mutex  sync.RWMutex
}

func (dl *DomainLimiter) Allow(domain string) bool {
    bucket := dl.getBucket(domain)
    return bucket.TakeToken()
}
\`\`\`

### Memory Optimization

For large crawls, we implemented:
- **Streaming Processing** - Process pages as they're downloaded
- **Disk Spillover** - Move inactive data to disk when memory is low  
- **Content Deduplication** - Avoid storing duplicate content
- **Garbage Collection Tuning** - Optimized GC settings for our workload

## Benchmarking Results

Our optimizations yielded significant improvements:

| Metric | Before | After | Improvement |
|--------|---------|-------|-------------|
| Pages/sec | 12 | 45 | 275% |
| Memory Usage | 2.1GB | 800MB | 62% reduction |
| CPU Usage | 85% | 45% | 47% reduction |

## Best Practices

Based on our experience, here are key recommendations:

### 1. Right-size Your Workers
More workers isn't always better. Find the sweet spot for your hardware:

\`\`\`bash
# Start conservative
ariadne crawl --workers 5

# Gradually increase while monitoring
ariadne crawl --workers 10
\`\`\`

### 2. Respect Rate Limits
Always configure appropriate rate limits:

\`\`\`yaml
crawler:
  rate_limit: 100ms  # 10 requests/second
  burst_limit: 5     # Allow short bursts
\`\`\`

### 3. Monitor Resource Usage
Use system monitoring to identify bottlenecks:

\`\`\`bash
# Monitor during crawl
htop
iostat 1
\`\`\`

### 4. Optimize Content Processing
Process content efficiently:

- Use streaming parsers for large documents
- Filter unwanted content early in the pipeline
- Cache expensive computations

## Future Optimizations

We're exploring additional optimizations:

- **HTTP/2 Connection Pooling** - Reuse connections more efficiently
- **Adaptive Rate Limiting** - Dynamically adjust rates based on server response
- **Content-Based Prioritization** - Crawl important pages first
- **Distributed Crawling** - Scale across multiple machines

## Conclusion

Performance optimization in web crawling requires balancing throughput, resource usage, and server respect. By implementing these techniques, Ariadne can efficiently crawl large sites while maintaining system stability.

Try these optimizations in your own crawling projects and let us know your results!`,
      },
      "markdown-processing": {
        title: "Advanced Markdown Processing Techniques",
        date: "2024-01-01",
        author: "Content Team",
        readTime: "6 min read",
        tags: ["markdown", "processing"],
        content: `
# Advanced Markdown Processing Techniques

Converting web content to Markdown is more complex than it initially appears. Here's how Ariadne handles advanced Markdown processing.

## The Markdown Challenge

Web pages contain rich formatting that doesn't directly map to standard Markdown:

- **Complex Tables** - With merged cells, styling, and nested content
- **Code Blocks** - Various syntax highlighting and language detection
- **Media Elements** - Images, videos, and interactive content  
- **Custom Elements** - Site-specific widgets and components

## Our Processing Pipeline

### 1. HTML Parsing and Cleaning

We start with robust HTML parsing:

\`\`\`go
type HTMLProcessor struct {
    parser *html.Parser
    clean  *bluemonday.Policy
}

func (p *HTMLProcessor) Clean(input []byte) *html.Node {
    doc := p.parser.Parse(input)
    return p.clean.SanitizeDOM(doc)
}
\`\`\`

### 2. Semantic Analysis

Identify content structure and meaning:

\`\`\`go
type ContentAnalyzer struct {
    headingDetector HeadingDetector
    tableDetector   TableDetector  
    codeDetector    CodeDetector
}

func (ca *ContentAnalyzer) Analyze(node *html.Node) *ContentStructure {
    return &ContentStructure{
        Headings: ca.headingDetector.Extract(node),
        Tables:   ca.tableDetector.Extract(node),
        Code:     ca.codeDetector.Extract(node),
    }
}
\`\`\`

### 3. Markdown Generation

Convert semantic content to Markdown:

## Advanced Features

### Table Processing

Complex tables require special handling:

| Feature | Challenge | Solution |
|---------|-----------|----------|
| Merged Cells | No native MD support | Convert to nested lists |
| Styling | CSS styling lost | Extract semantic meaning |
| Large Tables | Readability issues | Split or summarize |

### Code Block Detection

We use multiple heuristics to detect code:

\`\`\`go
type CodeDetector struct {
    patterns []regexp.Regexp
    languages map[string]LanguageConfig
}

func (cd *CodeDetector) DetectLanguage(code string) string {
    for lang, config := range cd.languages {
        if config.matches(code) {
            return lang
        }
    }
    return "text"
}
\`\`\`

### Image Processing

Images require special attention:

- **Alt Text Extraction** - For accessibility and context
- **Size Detection** - To determine if inline or block
- **URL Resolution** - Convert relative to absolute paths
- **Format Optimization** - Choose appropriate display format

### Link Processing

Link handling involves:

1. **URL Resolution** - Convert relative to absolute
2. **Link Validation** - Check for broken links  
3. **Anchor Processing** - Handle internal page links
4. **Link Context** - Preserve link context and meaning

## Custom Extensions

We support several Markdown extensions:

### Admonitions

\`\`\`markdown
!!! note "Important Note"
    This is an important note that stands out.

!!! warning "Careful!"
    Be careful with this operation.
\`\`\`

### Math Expressions

Inline math: \$E = mc^2\$

Block math:
\$\$
\\int_{-\\infty}^{\\infty} e^{-x^2} dx = \\sqrt{\\pi}
\$\$

### Footnotes

Content with footnote reference[^1].

[^1]: This is the footnote content.

## Quality Assurance

We validate output quality through:

- **Round-trip Testing** - Convert MD back to HTML and compare
- **Reference Preservation** - Ensure all links are maintained
- **Content Integrity** - Verify no content is lost
- **Format Validation** - Check Markdown syntax correctness

## Performance Considerations

Processing large documents efficiently:

\`\`\`go
type StreamingProcessor struct {
    chunkSize int
    buffer    []byte
}

func (sp *StreamingProcessor) Process(reader io.Reader) <-chan MarkdownChunk {
    chunks := make(chan MarkdownChunk)
    go sp.processChunks(reader, chunks)
    return chunks
}
\`\`\`

## Future Enhancements

Planned improvements include:

- **AI-Assisted Processing** - Use ML for better content understanding
- **Plugin Architecture** - Custom processors for specific sites
- **Interactive Elements** - Better handling of dynamic content
- **Multi-format Output** - Generate multiple formats simultaneously

## Conclusion

Advanced Markdown processing requires deep understanding of both source content and target format limitations. Ariadne's approach balances fidelity with readability to produce high-quality Markdown output.

What Markdown processing challenges have you encountered? Share your experiences!`,
      },
    };

    return posts[slug || ""] || null;
  };

  const post = getBlogPost(slug || "");

  if (!post) {
    return (
      <div className="max-w-4xl mx-auto text-center py-16">
        <h1 className="text-4xl font-bold mb-4">Post Not Found</h1>
        <p className="text-muted-foreground mb-6">
          The blog post you're looking for doesn't exist.
        </p>
        <Link
          to="/blog"
          className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
        >
          ← Back to Blog
        </Link>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto">
      {/* Breadcrumb */}
      <nav className="mb-6 text-sm">
        <Link to="/" className="text-blue-600 hover:underline">
          Home
        </Link>
        <span className="text-muted-foreground mx-2">→</span>
        <Link to="/blog" className="text-blue-600 hover:underline">
          Blog
        </Link>
        <span className="text-muted-foreground mx-2">→</span>
        <span className="text-muted-foreground">{post.title}</span>
      </nav>

      {/* Article Header */}
      <header className="mb-8">
        <h1 className="text-4xl font-bold tracking-tight mb-4">{post.title}</h1>

        <div className="flex items-center space-x-4 text-muted-foreground mb-6">
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

        <div className="flex flex-wrap gap-2">
          {post.tags.map((tag: string) => (
            <Link
              key={tag}
              to={`/tags/${tag}`}
              className="bg-muted hover:bg-muted/80 px-3 py-1 rounded-full text-sm transition-colors"
            >
              #{tag}
            </Link>
          ))}
        </div>
      </header>

      {/* Article Content */}
      <article className="prose max-w-none mb-12">
        <div
          className="markdown-content"
          dangerouslySetInnerHTML={{
            __html: post.content
              .split("\n")
              .map((line: string) => {
                // Basic markdown-to-HTML conversion for demo purposes
                if (line.startsWith("# ")) {
                  return `<h1 class="text-3xl font-bold mb-4">${line.slice(
                    2
                  )}</h1>`;
                } else if (line.startsWith("## ")) {
                  return `<h2 class="text-2xl font-bold mb-3">${line.slice(
                    3
                  )}</h2>`;
                } else if (line.startsWith("### ")) {
                  return `<h3 class="text-xl font-semibold mb-2">${line.slice(
                    4
                  )}</h3>`;
                } else if (line.startsWith("```")) {
                  return line.includes("```go")
                    ? `<pre class="bg-muted p-4 rounded-lg overflow-x-auto mb-4"><code class="language-go">`
                    : line.includes("```bash")
                    ? `<pre class="bg-muted p-4 rounded-lg overflow-x-auto mb-4"><code class="language-bash">`
                    : line.includes("```yaml")
                    ? `<pre class="bg-muted p-4 rounded-lg overflow-x-auto mb-4"><code class="language-yaml">`
                    : line.includes("```markdown")
                    ? `<pre class="bg-muted p-4 rounded-lg overflow-x-auto mb-4"><code class="language-markdown">`
                    : line === "```"
                    ? `</code></pre>`
                    : `<pre class="bg-muted p-4 rounded-lg overflow-x-auto mb-4"><code>`;
                } else if (line.includes("**") && !line.startsWith("#")) {
                  return `<p class="mb-4">${line.replace(
                    /\*\*(.*?)\*\*/g,
                    "<strong>$1</strong>"
                  )}</p>`;
                } else if (line.startsWith("- ")) {
                  return `<li class="ml-4">${line.slice(2)}</li>`;
                } else if (line.startsWith("| ")) {
                  // Simple table handling
                  const cells = line.split("|").slice(1, -1);
                  if (line.includes("-----")) {
                    return ""; // Skip separator rows
                  }
                  return `<tr>${cells
                    .map(
                      (cell) =>
                        `<td class="border border-border p-3">${cell.trim()}</td>`
                    )
                    .join("")}</tr>`;
                } else if (
                  line.trim() &&
                  !line.startsWith("<") &&
                  !line.startsWith("```")
                ) {
                  return `<p class="mb-4">${line}</p>`;
                } else {
                  return line;
                }
              })
              .join("")
              .replace(
                /(<li.*?>.*<\/li>\n?)/g,
                '<ul class="list-disc ml-6 space-y-1 mb-4">$1</ul>'
              )
              .replace(
                /(<tr>.*<\/tr>\n?)+/g,
                '<table class="min-w-full border-collapse border border-border mb-6"><tbody>$&</tbody></table>'
              ),
          }}
        />
      </article>

      {/* Article Footer */}
      <footer className="border-t pt-8">
        <div className="flex justify-between items-center mb-6">
          <Link
            to="/blog"
            className="inline-flex items-center px-4 py-2 border border-border rounded-md hover:bg-muted transition-colors"
          >
            ← Back to Blog
          </Link>

          <div className="text-sm text-muted-foreground">
            Share this post:
            <button className="ml-2 text-blue-600 hover:underline">
              Twitter
            </button>
            <button className="ml-2 text-blue-600 hover:underline">
              LinkedIn
            </button>
          </div>
        </div>

        <Card className="p-6">
          <h3 className="text-lg font-semibold mb-2">Related Posts</h3>
          <p className="text-muted-foreground mb-4">
            Explore more content related to this topic.
          </p>
          <div className="space-x-4">
            <Link to="/blog" className="text-blue-600 hover:underline">
              View All Posts
            </Link>
            <Link
              to={`/tags/${post.tags[0]}`}
              className="text-blue-600 hover:underline"
            >
              #{post.tags[0]} Posts
            </Link>
          </div>
        </Card>
      </footer>
    </div>
  );
}
