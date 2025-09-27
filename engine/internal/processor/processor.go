package processor

import (
    "fmt"
    "net/url"
    "regexp"
    "strings"
    "time"


    "github.com/99souls/ariadne/engine/models"

    "github.com/JohannesKaufmann/html-to-markdown/v2/converter"
    "github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
    "github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
    "github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
    "github.com/PuerkitoBio/goquery"
)

// (Asset pipeline dependencies removed during internalization; will be reintroduced later if needed)

// ContentProcessor handles HTML content cleaning and processing
type ContentProcessor struct{}

func NewContentProcessor() *ContentProcessor { return &ContentProcessor{} }

// ExtractContent extracts main content using selectors
func (cp *ContentProcessor) ExtractContent(html string, selectors []string) (string, error) {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
    if err != nil { return "", err }
    for _, selector := range selectors {
        selection := doc.Find(selector)
        if selection.Length() > 0 {
            content, err := selection.Html(); if err != nil { continue }
            return strings.TrimSpace(content), nil
        }
    }
    bodySelection := doc.Find("body")
    if bodySelection.Length() > 0 {
        bodySelection.Find("script, style, nav, footer, aside, header").Remove()
        bodySelection.Find(".advertisement, .ad, .ads").Remove()
        bodySelection.Find("img[width='1'][height='1']").Remove()
        bodyContent, err := bodySelection.Html(); if err != nil { return "", fmt.Errorf("could not extract any content: %w", err) }
        return strings.TrimSpace(bodyContent), nil
    }
    return "", fmt.Errorf("could not extract any content: no body found")
}

func (cp *ContentProcessor) RemoveUnwantedElements(html string) (string, error) {
    re := regexp.MustCompile(`<!--[\s\S]*?-->`)
    html = re.ReplaceAllString(html, "")
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html)); if err != nil { return "", err }
    unwantedTags := []string{"script", "style", "nav", "footer", "aside", "header"}
    for _, tag := range unwantedTags { doc.Find(tag).Remove() }
    unwantedSelectors := []string{".advertisement", ".ad", ".ads", ".sidebar", ".nav", ".navigation", ".footer", ".header", "#comments", ".comments"}
    for _, selector := range unwantedSelectors { doc.Find(selector).Remove() }
    doc.Find("img[width='1'][height='1']").Remove()
    bodyContent := doc.Find("body")
    if bodyContent.Length() > 0 { result, err := bodyContent.Html(); if err != nil { return "", err }; return result, nil }
    result, err := doc.Html(); if err != nil { return "", err }; return result, nil
}

func (cp *ContentProcessor) ConvertRelativeURLs(html, baseURL string) (string, error) {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html)); if err != nil { return "", err }
    base, err := url.Parse(baseURL); if err != nil { return "", fmt.Errorf("invalid base URL: %w", err) }
    doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
        href, exists := s.Attr("href"); if !exists { return }
        if strings.HasPrefix(href, "http") || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "mailto:") { return }
        absoluteURL, err := base.Parse(href); if err != nil { return }
        s.SetAttr("href", absoluteURL.String())
    })
    doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
        src, exists := s.Attr("src"); if !exists { return }
        if strings.HasPrefix(src, "http") || strings.HasPrefix(src, "data:") { return }
        absoluteURL, err := base.Parse(src); if err != nil { return }
        s.SetAttr("src", absoluteURL.String())
    })
    result, err := doc.Html(); if err != nil { return "", err }
    return result, nil
}

func (cp *ContentProcessor) ExtractMetadata(html string) (string, *models.PageMeta, error) {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html)); if err != nil { return "", nil, err }
    meta := &models.PageMeta{}
    title := strings.TrimSpace(doc.Find("title").Text())
    if title == "" { title = strings.TrimSpace(doc.Find("h1").First().Text()) }
    description, _ := doc.Find("meta[name='description']").Attr("content")
    if description == "" { description, _ = doc.Find("meta[property='og:description']").Attr("content") }
    meta.Description = strings.TrimSpace(description)
    keywords, _ := doc.Find("meta[name='keywords']").Attr("content")
    if keywords != "" { keywordList := strings.Split(keywords, ","); for i, k := range keywordList { keywordList[i] = strings.TrimSpace(k) }; meta.Keywords = keywordList }
    author, _ := doc.Find("meta[name='author']").Attr("content"); meta.Author = strings.TrimSpace(author)
    ogTitle, _ := doc.Find("meta[property='og:title']").Attr("content")
    ogDesc, _ := doc.Find("meta[property='og:description']").Attr("content")
    ogImage, _ := doc.Find("meta[property='og:image']").Attr("content")
    ogURL, _ := doc.Find("meta[property='og:url']").Attr("content")
    ogType, _ := doc.Find("meta[property='og:type']").Attr("content")
    meta.OpenGraph = models.OpenGraphMeta{Title: strings.TrimSpace(ogTitle), Description: strings.TrimSpace(ogDesc), Image: strings.TrimSpace(ogImage), URL: strings.TrimSpace(ogURL), Type: strings.TrimSpace(ogType)}
    return title, meta, nil
}

func (cp *ContentProcessor) ExtractImages(html, baseURL string) ([]string, error) {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html)); if err != nil { return nil, err }
    base, err := url.Parse(baseURL); if err != nil { return nil, fmt.Errorf("invalid base URL: %w", err) }
    var images []string
    doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
        src, exists := s.Attr("src"); if !exists || src == "" { return }
        if strings.HasPrefix(src, "data:") || (s.AttrOr("width", "") == "1" && s.AttrOr("height", "") == "1") { return }
        if !strings.HasPrefix(src, "http") { absoluteURL, err := base.Parse(src); if err != nil { return }; src = absoluteURL.String() }
        images = append(images, src)
    })
    return images, nil
}

func (cp *ContentProcessor) ProcessPage(page *models.Page, baseURL string) error {
    if page == nil { return fmt.Errorf("page cannot be nil") }
    if strings.TrimSpace(page.Content) == "" { return fmt.Errorf("page content is empty") }
    if strings.HasPrefix(strings.TrimSpace(page.Content), "<<") { return fmt.Errorf("content appears to be malformed HTML") }
    cleaned, err := cp.RemoveUnwantedElements(page.Content); if err != nil { return fmt.Errorf("failed to clean content: %w", err) }
    withAbsolute, err := cp.ConvertRelativeURLs(cleaned, baseURL); if err != nil { return fmt.Errorf("failed to convert URLs: %w", err) }
    selectors := []string{"main", "article", ".content", "#content", ".post", ".entry", ".article-content"}
    extracted, err := cp.ExtractContent(withAbsolute, selectors); if err != nil { extracted = withAbsolute }
    converter := NewHTMLToMarkdownConverter(); markdown, err := converter.Convert(extracted); if err != nil { return fmt.Errorf("failed to convert to markdown: %w", err) }
    title, meta, err := cp.ExtractMetadata(page.Content); if err != nil { meta = &models.PageMeta{}; title = "" }
    images, err := cp.ExtractImages(extracted, baseURL); if err != nil { images = []string{} }
    cleanText := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(extracted, " "); words := strings.Fields(cleanText); meta.WordCount = len(words)
    page.Content = extracted; page.Markdown = markdown; page.Title = title; page.Images = images; page.Metadata = *meta; page.ProcessedAt = time.Now()
    return nil
}

type WorkerPool struct{ workerCount int }
type HTMLToMarkdownConverter struct{}
type ContentValidator struct{}
type ValidationResult struct { IsValid bool `json:"is_valid"`; Score float64 `json:"score"`; Issues []string `json:"issues"`; WordCount int `json:"word_count"`; HasContent bool `json:"has_content"`; HasHeadings bool `json:"has_headings"` }

func NewWorkerPool(workerCount int) *WorkerPool { return &WorkerPool{workerCount: workerCount} }
func NewHTMLToMarkdownConverter() *HTMLToMarkdownConverter { return &HTMLToMarkdownConverter{} }
func NewContentValidator() *ContentValidator { return &ContentValidator{} }
func (wp *WorkerPool) WorkerCount() int { return wp.workerCount }
func (wp *WorkerPool) Stop()            {}

func (wp *WorkerPool) ProcessPages(pages []*models.Page, baseURL string) <-chan *models.CrawlResult {
    results := make(chan *models.CrawlResult, len(pages))
    go func() {
        defer close(results)
        for _, page := range pages {
            processor := NewContentProcessor(); err := processor.ProcessPage(page, baseURL)
            resultURL := ""; if page != nil && page.URL != nil { resultURL = page.URL.String() }
            res := &models.CrawlResult{ URL: resultURL, Page: page, Success: err == nil, Stage: "processing" }
            if err != nil { res.Error = err }
            results <- res
        }
    }()
    return results
}

func (c *HTMLToMarkdownConverter) Convert(html string) (string, error) {
    if strings.TrimSpace(html) == "" { return "", fmt.Errorf("HTML content is empty") }
    conv := converter.NewConverter(converter.WithPlugins(base.NewBasePlugin(), commonmark.NewCommonmarkPlugin(), table.NewTablePlugin()))
    markdown, err := conv.ConvertString(html); if err != nil { return "", fmt.Errorf("conversion failed: %w", err) }
    cleaned := cleanMarkdown(markdown); return cleaned, nil
}

func cleanMarkdown(markdown string) string {
    re := regexp.MustCompile(`<!--[\s\S]*?-->`); cleaned := re.ReplaceAllString(markdown, "")
    re = regexp.MustCompile(`\n{3,}`); cleaned = re.ReplaceAllString(cleaned, "\n\n")
    cleaned = strings.ReplaceAll(cleaned, "\\n", "\n")
    cleaned = strings.ReplaceAll(cleaned, `\"`, `"`)
    lines := strings.Split(cleaned, "\n")
    for i, line := range lines {
        if strings.Contains(line, "|") && !strings.HasPrefix(strings.TrimSpace(line), "|--") {
            parts := strings.Split(line, "|")
            for j, part := range parts { parts[j] = strings.TrimSpace(part) }
            if len(parts) > 2 && parts[0] == "" && parts[len(parts)-1] == "" {
                var cleanParts []string
                for k := 1; k < len(parts)-1; k++ { cleanParts = append(cleanParts, parts[k]) }
                lines[i] = "| " + strings.Join(cleanParts, " | ") + " |"
            }
        } else { lines[i] = strings.TrimRight(line, " ") }
    }
    return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (cv *ContentValidator) ValidateContent(page *models.Page) *ValidationResult {
    result := &ValidationResult{ IsValid: true, Score: 1.0, Issues: []string{} }
    if page == nil { result.IsValid = false; result.Score = 0; result.Issues = append(result.Issues, "page_is_nil"); return result }
    content := strings.TrimSpace(page.Content); if content == "" { result.IsValid = false; result.Score = 0; result.Issues = append(result.Issues, "no_content"); result.HasContent = false; return result }
    result.HasContent = true
    wordCount := page.Metadata.WordCount
    if wordCount == 0 { words := strings.Fields(strings.ReplaceAll(content, "<", " <")); for _, w := range words { if !strings.HasPrefix(w, "<") { wordCount++ } } }
    result.WordCount = wordCount
    titleMissing := strings.TrimSpace(page.Title) == ""; titleTooShort := !titleMissing && len(strings.TrimSpace(page.Title)) < 10
    contentTooShort := wordCount < 5
    lowContentDensity := false
    if wordCount >= 5 && wordCount <= 15 { htmlTagCount := strings.Count(content, "<"); if htmlTagCount > 0 { contentDensity := float64(wordCount)/float64(htmlTagCount); lowContentDensity = contentDensity < 1.0 } }
    if titleMissing { result.Issues = append(result.Issues, "missing_title"); result.Score -= 0.4 } else if titleTooShort { result.Issues = append(result.Issues, "title_too_short"); result.Score -= 0.3 }
    if contentTooShort { result.Issues = append(result.Issues, "content_too_short"); result.Score -= 0.4 } else if lowContentDensity { result.Issues = append(result.Issues, "low_content_density"); result.Score -= 0.3 }
    if wordCount >= 15 && !strings.Contains(content, "<h1") && !strings.Contains(content, "<h2") { result.HasHeadings = false; if len(result.Issues) == 0 { result.Issues = append(result.Issues, "no_headings"); result.Score -= 0.2 } } else { result.HasHeadings = strings.Contains(content, "<h1") || strings.Contains(content, "<h2") }
    if result.Score < 0 { result.Score = 0 }
    if result.Score < 0.6 || len(result.Issues) > 0 { result.IsValid = false }
    return result
}
