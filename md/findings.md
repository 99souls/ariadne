# Web Scraper for Wiki-Style Sites: Research Findings

## Project Overview

Building a web scraper to navigate through internal links on a website (like an Obsidian Quartz wiki) and assemble content into PDF/MD/HTML documents.

## Best Approaches & Architecture

### 1. **Crawling Strategy**

- **Breadth-First Search (BFS)**: Start from a root page and systematically visit all internal links
- **Depth-Limited Crawling**: Set maximum depth to avoid infinite loops
- **Visited Set**: Track visited URLs to prevent duplicate processing
- **Link Filtering**: Only follow internal links (same domain/subdomain)
- **Respect robots.txt**: Check site's crawling policies

### 2. **Core Technology Stack (Go)**

#### Web Scraping Framework

- **Colly v2** (`github.com/gocolly/colly/v2`)
  - Pros: Fast, concurrent, built-in rate limiting, domain filtering
  - Perfect for link traversal with `OnHTML("a[href]")` handlers
  - Built-in support for duplicate URL detection
  - Excellent for structured crawling

#### HTML Parsing & Content Extraction

- **goquery** (`github.com/PuerkitoBio/goquery`)
  - jQuery-like syntax for Go
  - Excellent for extracting content from specific elements
  - Works well with Colly for content processing

#### Content Processing & Output Generation

- **HTML to Markdown**: `github.com/JohannesKaufmann/html-to-markdown/v2`

  - Excellent for converting scraped HTML to clean Markdown
  - Handles complex HTML structures, tables, links, images
  - Configurable plugins for different output styles

- **PDF Generation**: Multiple options
  - `github.com/unidoc/unipdf` - Pure Go PDF library (commercial license)
  - `github.com/signintech/gopdf` - Open source, good for simple PDFs
  - `github.com/sebastiaanklippert/go-wkhtmltopdf` - HTML to PDF via wkhtmltopdf

### 3. **Recommended Implementation Flow**

```
1. Initialize Colly crawler with domain restrictions
2. Set up concurrent workers with rate limiting
3. For each page:
   - Extract main content (remove navigation, headers, footers)
   - Clean and normalize HTML
   - Extract internal links for crawling queue
   - Store content with metadata (URL, title, timestamp)
4. Post-processing:
   - Sort/organize content by URL structure
   - Convert to desired formats (MD/HTML/PDF)
   - Generate table of contents
   - Create consolidated document
```

### 4. **Content Extraction Strategies**

#### For Wiki-Style Sites (Obsidian Quartz)

- Target main content areas: `article`, `.content`, `#main-content`
- Remove navigation: `nav`, `.sidebar`, `.header`, `.footer`
- Preserve internal links and convert to local references
- Handle images: download and reference locally or convert to base64

#### Content Cleaning

- Remove script tags, style elements
- Normalize whitespace and line breaks
- Handle relative URLs (convert to absolute)
- Extract and preserve metadata (titles, dates, authors)

### 5. **Output Format Generation**

#### Markdown Output

- Use html-to-markdown library with smart escaping
- Preserve code blocks, tables, and formatting
- Create index/TOC with internal links
- Support for wiki-style cross-references

#### PDF Output

- Generate from cleaned HTML using wkhtmltopdf wrapper
- Or use markdown-to-HTML-to-PDF pipeline
- Include proper page breaks between sections
- Custom CSS for professional formatting

#### HTML Output

- Template-based generation
- Include navigation between sections
- Responsive design for different devices
- Search functionality (optional)

### 6. **Performance & Reliability Features**

#### Crawling Best Practices

- Rate limiting: 1-2 requests per second to be respectful
- User-Agent identification
- Retry logic for failed requests
- Timeout handling
- Concurrent processing (2-4 workers for small sites)

#### Error Handling

- Graceful handling of 404s, timeouts
- Logging of failed URLs for manual review
- Partial completion support (resume from checkpoint)
- Validation of extracted content

### 7. **Advanced Features**

#### Content Organization

- Automatic categorization by URL structure
- Chronological sorting for blog-style content
- Tag extraction and grouping
- Hierarchical organization for nested content

#### Quality Improvements

- Content deduplication
- Image optimization and downloading
- Link validation and correction
- Metadata extraction (titles, descriptions, keywords)

## Implementation Recommendations

### Phase 1: Core Scraper

1. Set up Colly with domain restrictions and rate limiting
2. Implement basic link traversal and content extraction
3. Store raw content with metadata

### Phase 2: Content Processing

1. Clean HTML content and extract main text
2. Implement html-to-markdown conversion
3. Handle images and assets

### Phase 3: Output Generation

1. Markdown compilation with TOC
2. HTML template generation
3. PDF generation pipeline

### Phase 4: Polish

1. Error handling and retry logic
2. Progress reporting and logging
3. Configuration file support
4. CLI interface improvements

## Alternative Tools & Libraries

### Go Alternatives

- **Soup** (`github.com/anaskhan96/soup`) - BeautifulSoup-like HTML parsing
- **gofeed** (`github.com/mmcdole/gofeed`) - For RSS/feed-based sites
- **chromedp** - For JavaScript-heavy sites requiring browser rendering

### Non-Go Solutions

- **Scrapy** (Python) - Industrial-strength scraping framework
- **Puppeteer/Playwright** (Node.js) - For JavaScript-heavy sites
- **Crawlee** (Node.js/Python) - Modern scraping framework with anti-detection

## Estimated Development Time

- **Basic functional scraper**: 2-3 days
- **With all output formats**: 4-5 days
- **Production-ready with error handling**: 6-8 days

## Orchestration Flow & Concurrency Architecture

### **Multi-Stage Pipeline Orchestration**

The scraping process should be designed as a multi-stage pipeline with clear separation of concerns and optimal resource utilization:

```
[URL Discovery] → [Content Extraction] → [Processing] → [Output Generation]
     ↓                    ↓                 ↓              ↓
  Link Queue         Raw Content        Clean Data      Final Files
```

#### **Stage 1: URL Discovery & Queueing**

```go
type URLQueue struct {
    discovered chan string     // New URLs found during crawling
    pending    chan string     // URLs ready for processing
    visited    sync.Map        // Thread-safe visited URL tracking
    inProgress sync.Map        // Currently processing URLs
}
```

- **Single Crawler Worker**: One dedicated goroutine for link discovery
- **Breadth-First Traversal**: Maintains site structure and prevents deep rabbit holes
- **Duplicate Detection**: Thread-safe visited set using `sync.Map`
- **URL Normalization**: Clean and canonicalize URLs before queueing

#### **Stage 2: Concurrent Content Extraction**

```go
type ContentExtractor struct {
    workers    int
    client     *colly.Collector
    queue      <-chan string
    results    chan<- RawContent
    rateLimiter *time.Ticker
}
```

**Worker Pool Pattern**:

- **8-16 concurrent workers** for large sites (1000+ pages)
- **2-4 workers** for small sites (< 100 pages)
- **Rate limiting**: 1-2 requests/second per worker to be respectful
- **Circuit breaker**: Auto-pause on repeated failures
- **Timeout handling**: 30s timeout per request with exponential backoff

#### **Stage 3: Content Processing Pipeline**

```go
type ProcessingPipeline struct {
    rawContent    <-chan RawContent
    cleanContent  chan CleanContent
    processors    int                // CPU-bound workers
    htmlCleaner   *html2markdown.Converter
    imageHandler  *AssetDownloader
}
```

**CPU-Bound Processing**:

- **Worker count**: `runtime.NumCPU()` for optimal CPU utilization
- **HTML cleaning**: Remove nav, ads, scripts in parallel
- **Content extraction**: Target main article/content areas
- **Image processing**: Download and optimize assets concurrently
- **Markdown conversion**: Clean HTML to markdown with proper escaping

#### **Stage 4: Output Generation**

```go
type OutputGenerator struct {
    content     <-chan CleanContent
    formats     []OutputFormat      // MD, HTML, PDF
    assembler   *DocumentAssembler
    concurrent  bool                // Generate formats in parallel
}
```

### **Concurrency Scaling Strategies**

#### **Small Sites (< 100 pages)**

```go
config := ScraperConfig{
    CrawlWorkers:    1,           // Single crawler to maintain order
    ExtractWorkers:  2,           // Light extraction workload
    ProcessWorkers:  2,           // Light processing workload
    RateLimit:       1 * time.Second,
    Timeout:         15 * time.Second,
}
```

#### **Medium Sites (100-1000 pages)**

```go
config := ScraperConfig{
    CrawlWorkers:    1,           // Still single crawler
    ExtractWorkers:  4,           // More extraction concurrency
    ProcessWorkers:  runtime.NumCPU(), // CPU-bound processing
    RateLimit:       500 * time.Millisecond,
    Timeout:         30 * time.Second,
    BatchSize:       50,          // Process in batches
}
```

#### **Large Sites (1000+ pages)**

```go
config := ScraperConfig{
    CrawlWorkers:    2,           // Two crawlers for different sections
    ExtractWorkers:  8,           // High extraction concurrency
    ProcessWorkers:  runtime.NumCPU() * 2, // Utilize hyperthreading
    RateLimit:       200 * time.Millisecond,
    Timeout:         45 * time.Second,
    BatchSize:       100,         // Larger batches
    CheckpointInterval: 1000,     // Progress checkpointing
}
```

### **Advanced Orchestration Features**

#### **Dynamic Rate Limiting**

```go
type AdaptiveRateLimiter struct {
    baseDelay     time.Duration
    currentDelay  time.Duration
    errorCount    int32
    successCount  int32
    adjustment    sync.RWMutex
}
```

- **Adaptive throttling**: Slow down on 429/503 responses
- **Server health monitoring**: Adjust rates based on response times
- **Burst handling**: Allow controlled bursts, then back off
- **Per-domain limits**: Different rates for different hosts/subdomains

#### **Intelligent Work Distribution**

```go
type WorkScheduler struct {
    priorityQueue  *heap.Interface    // Priority-based URL processing
    domainLimits   map[string]int     // Per-domain concurrency limits
    backoffMap     sync.Map           // Domain-specific backoff timers
}
```

- **Priority-based processing**: Process important pages first (index, navigation)
- **Domain-aware scheduling**: Distribute load across subdomains
- **Backoff coordination**: Share backoff state across workers
- **Content-aware prioritization**: Process shorter pages first

#### **Memory & Resource Management**

```go
type ResourceManager struct {
    maxMemory      uint64
    contentCache   *lru.Cache         // LRU cache for processed content
    tempFiles      *os.File           // Spill to disk for large sites
    compressionPool *sync.Pool        // Reusable compression buffers
}
```

- **Memory monitoring**: Track RSS usage, spill to disk when needed
- **Content caching**: LRU cache for processed content to avoid reprocessing
- **Garbage collection tuning**: Optimize GC for high-throughput processing
- **Buffer pools**: Reuse buffers for HTML processing and conversion

### **Error Handling & Recovery**

#### **Graceful Degradation**

```go
type ErrorHandler struct {
    failedURLs     chan string       // Queue for retry processing
    maxRetries     int               // Exponential backoff retries
    circuitBreaker *breaker.CircuitBreaker
    checkpoint     *ProgressState    // Resumable state persistence
}
```

- **Retry queues**: Separate queue for failed URLs with exponential backoff
- **Circuit breaker**: Auto-pause crawling on sustained failures
- **Checkpoint persistence**: Save progress to resume interrupted sessions
- **Partial completion**: Generate outputs from successfully processed content

#### **Monitoring & Observability**

```go
type MetricsCollector struct {
    processedPages   int64
    failedRequests   int64
    avgResponseTime  time.Duration
    memoryUsage      uint64
    workersActive    int32
    progressReporter chan ProgressUpdate
}
```

- **Real-time progress**: Live updates on processing status
- **Performance metrics**: Response times, throughput, error rates
- **Resource utilization**: Memory, CPU, goroutine tracking
- **ETA calculation**: Estimate completion time based on current rates

### **Production Considerations**

#### **Configuration Management**

```yaml
# scraper.yaml
scraper:
  workers:
    crawl: 1
    extract: 4
    process: 8
  limits:
    rate: "500ms"
    timeout: "30s"
    memory: "1GB"
  output:
    formats: ["markdown", "pdf"]
    batch_size: 50
```

#### **Deployment Patterns**

- **Horizontal scaling**: Multiple scraper instances for different site sections
- **Resource limits**: Container resource constraints for predictable performance
- **Queue persistence**: Redis/database backing for URL queues in production
- **Distributed coordination**: Coordination between multiple scraper instances

### **Performance Benchmarks (Estimated)**

| Site Size    | Pages/min | Memory Usage | Completion Time |
| ------------ | --------- | ------------ | --------------- |
| Small (50)   | 30-60     | 50-100MB     | 1-2 minutes     |
| Medium (500) | 100-200   | 200-500MB    | 3-5 minutes     |
| Large (5000) | 200-400   | 1-2GB        | 15-25 minutes   |

_Benchmarks assume typical wiki-style content with mixed text/images and respectful rate limiting_

## Conclusion

For a Go-based solution targeting wiki-style sites like Obsidian Quartz, the combination of **Colly v2** for crawling, **goquery** for HTML parsing, and **html-to-markdown v2** for content conversion provides an excellent foundation. This stack offers the right balance of performance, reliability, and ease of use for this specific use case.

The multi-stage pipeline architecture with intelligent concurrency scaling ensures optimal performance across different site sizes while maintaining respectful crawling practices and robust error handling.
