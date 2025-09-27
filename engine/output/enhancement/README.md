# Enhancement Engine

Content enhancement layer for Ariadne's output pipeline. Transforms raw crawled content into navigable, searchable documentation with intelligent cross-referencing.

## What it does

**Table of Contents Generation**

- Extracts document hierarchy and generates multi-level navigation
- Creates cross-references between related pages based on content analysis
- Supports configurable depth limits and anchor generation

**Content Indexing**

- Builds alphabetical and topic-based indexes of all content
- Extracts keywords automatically from page metadata and structure
- Separates page-level and section-level entries for granular navigation

**Search Infrastructure**

- Full-text search with relevance scoring based on title and content matching
- Generates client-side JavaScript for real-time search functionality
- Configurable search parameters and result ranking

**Adaptive Styling**

- Theme system supporting professional and dark modes
- Responsive layouts optimized for desktop, mobile, and print
- Syntax highlighting integration for code documentation

## Architecture

The enhancement engine operates on pages collected during the crawling phase, building navigation and search structures that integrate with Ariadne's markdown and HTML output systems.

```go
type ContentEnhancer struct {
    config      ContentEnhancementConfig
    styleConfig StylingConfig
    pages       []*models.Page
    searchIndex *SearchIndex
    stats       ContentEnhancementStats
    mutex       sync.RWMutex
}
```

## Configuration

```go
enhancer := NewContentEnhancer()

// Custom settings
config := ContentEnhancementConfig{
    GenerateTOC:     true,
    GenerateIndex:   true,
    EnableSearch:    true,
    TOCMaxDepth:     4,
    SearchMinLength: 2,
}
enhancer.SetConfig(config)

// Styling options
styleConfig := StylingConfig{
    Theme:          "professional",
    PrimaryColor:   "#2c3e50",
    SecondaryColor: "#3498db",
    FontFamily:     "Inter, sans-serif",
}
enhancer.SetStylingConfig(styleConfig)
```

## Usage

```go
// Add pages from crawler
for _, page := range crawledPages {
    enhancer.AddPage(page)
}

// Generate enhanced output
output := enhancer.GenerateEnhancedOutput(documentHierarchy)

// Access components
toc := output.TOC           // Enhanced table of contents
index := output.Index       // Content indexes
searchJS := output.SearchJS // Client-side search
css := output.CustomCSS     // Themed styles
```

## Output Structure

**Enhanced TOC**

```go
type EnhancedTOC struct {
    Sections []*TOCSection
    MaxDepth int
}

type TOCSection struct {
    Title           string
    Level           int
    Anchor          string
    URL             string
    Subsections     []*TOCSection
    CrossReferences []CrossReference
}
```

**Search System**

```go
type SearchIndex struct {
    Entries []*SearchEntry
}

// Perform searches
results := enhancer.Search("authentication")
```

## Integration

Works with Ariadne's output pipeline:

- Receives pages from the crawler and processor
- Integrates with document assembly for hierarchy context
- Provides enhanced content to markdown and HTML renderers

## Quality

- 91.1% test coverage
- Thread-safe concurrent operations
- Performance-optimized algorithms
- Memory-efficient data structures
