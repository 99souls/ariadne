package engine

// Phase 5D Iterations 1-4: Asset strategy interface + discovery + decision matrix
// + basic download execution + deterministic path + optimization stub.
//
// Later iterations will introduce concurrency, metrics, richer optimization,
// and integration into the engine processor lifecycle wiring.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	engmodels "ariadne/packages/engine/models"

	"github.com/PuerkitoBio/goquery"
)

// AssetRef represents a discovered asset reference inside a page.
type AssetRef struct {
	URL      string
	Type     string // e.g. img, script, stylesheet
	Attr     string // attribute name (src, href, data-src)
	Original string // original raw attribute value
}

// AssetMode describes the handling decision for an asset.
type AssetMode int

const (
	AssetModeDownload AssetMode = iota
	AssetModeSkip
	AssetModeInline
	AssetModeRewrite
)

// AssetAction couples a reference with a decided handling mode.
type AssetAction struct {
	Ref  AssetRef
	Mode AssetMode
}

// MaterializedAsset represents an asset after execution (download / inline / optimization).
type MaterializedAsset struct {
	Ref           AssetRef
	Bytes         []byte
	Hash          string   // sha256
	Path          string   // stable relative path
	Size          int      // size after optimization (if any)
	Optimizations []string // applied optimization identifiers
}

// AssetStrategy defines the pluggable asset handling pipeline lifecycle.
type AssetStrategy interface {
	Discover(ctx context.Context, page *engmodels.Page) ([]AssetRef, error)
	Decide(ctx context.Context, refs []AssetRef, policy AssetPolicy) ([]AssetAction, error)
	Execute(ctx context.Context, actions []AssetAction, policy AssetPolicy) ([]MaterializedAsset, error)
	Rewrite(ctx context.Context, page *engmodels.Page, assets []MaterializedAsset, policy AssetPolicy) (*engmodels.Page, error)
	Name() string
}

// AssetEvent represents a lifecycle occurrence for observability.
type AssetEvent struct {
	Type          string        // e.g. asset_download, asset_stage_error, asset_rewrite
	URL           string        // asset URL (where applicable)
	Stage         string        // discover|decide|execute|rewrite
	BytesIn       int           // pre-optimization bytes
	BytesOut      int           // post-optimization bytes
	Duration      time.Duration // operation latency (download/optimize)
	Error         string        // error message if any
	Optimizations []string      // optimization identifiers applied
}

// AssetEventPublisher publishes events (non-blocking behavior recommended).
type AssetEventPublisher interface{ Publish(AssetEvent) }

// AssetMetrics holds counters for asset processing lifecycle.
type AssetMetrics struct {
	discovered      int64
	selected        int64
	skipped         int64
	downloaded      int64
	failed          int64 // Iteration 7 part 2: failed download attempts
	inlined         int64
	optimized       int64
	bytesIn         int64
	bytesOut        int64
	rewriteFailures int64
}

// Snapshot returns immutable view for assertions / reporting.
type AssetMetricsSnapshot struct {
	Discovered      int64
	Selected        int64
	Skipped         int64
	Downloaded      int64
	Failed          int64
	Inlined         int64
	Optimized       int64
	BytesIn         int64
	BytesOut        int64
	RewriteFailures int64
}

func (m *AssetMetrics) snapshot() AssetMetricsSnapshot {
	if m == nil {
		return AssetMetricsSnapshot{}
	}
	return AssetMetricsSnapshot{
		Discovered:      atomic.LoadInt64(&m.discovered),
		Selected:        atomic.LoadInt64(&m.selected),
		Skipped:         atomic.LoadInt64(&m.skipped),
		Downloaded:      atomic.LoadInt64(&m.downloaded),
		Failed:          atomic.LoadInt64(&m.failed),
		Inlined:         atomic.LoadInt64(&m.inlined),
		Optimized:       atomic.LoadInt64(&m.optimized),
		BytesIn:         atomic.LoadInt64(&m.bytesIn),
		BytesOut:        atomic.LoadInt64(&m.bytesOut),
		RewriteFailures: atomic.LoadInt64(&m.rewriteFailures),
	}
}

// DefaultAssetStrategy implements AssetStrategy with instrumentation hooks.
type DefaultAssetStrategy struct {
	metrics *AssetMetrics
	events  AssetEventPublisher
}

func NewDefaultAssetStrategy(m *AssetMetrics, pub AssetEventPublisher) *DefaultAssetStrategy {
	return &DefaultAssetStrategy{metrics: m, events: pub}
}

func (s *DefaultAssetStrategy) Name() string { return "noop" }

// Discover parses the HTML and extracts candidate asset references.
func (s *DefaultAssetStrategy) Discover(ctx context.Context, page *engmodels.Page) ([]AssetRef, error) {
	if page == nil || page.Content == "" || page.URL == nil {
		return nil, nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(page.Content))
	if err != nil {
		if s.events != nil {
			s.events.Publish(AssetEvent{Type: "asset_stage_error", Stage: "discover", Error: err.Error()})
		}
		return nil, err
	}
	var refs []AssetRef
	seen := make(map[string]struct{})
	add := func(r AssetRef) {
		if r.URL == "" {
			return
		}
		key := r.Type + "|" + r.URL
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		refs = append(refs, r)
	}
	base := page.URL
	resolve := func(raw string) string {
		u, err := base.Parse(raw)
		if err != nil {
			return ""
		}
		return u.String()
	}
	doc.Find("img[src]").Each(func(_ int, sel *goquery.Selection) {
		v, _ := sel.Attr("src")
		if v == "" {
			return
		}
		abs := resolve(v)
		if abs == "" {
			return
		}
		add(AssetRef{URL: abs, Type: "img", Attr: "src", Original: v})
	})
	// Iteration 7: parse srcset (choose first URL for now; future: multiple variants)
	doc.Find("img[srcset]").Each(func(_ int, sel *goquery.Selection) {
		v, _ := sel.Attr("srcset")
		if v == "" {
			return
		}
		// pick first candidate before comma
		first := strings.TrimSpace(strings.Split(v, ",")[0])
		// remove descriptor (e.g., '1x' or '480w')
		parts := strings.Fields(first)
		if len(parts) > 0 {
			first = parts[0]
		}
		abs := resolve(first)
		if abs == "" {
			return
		}
		add(AssetRef{URL: abs, Type: "img", Attr: "srcset", Original: first})
	})
	// <source srcset> inside <picture> or media elements (treat like img)
	doc.Find("source[srcset]").Each(func(_ int, sel *goquery.Selection) {
		v, _ := sel.Attr("srcset")
		if v == "" {
			return
		}
		first := strings.TrimSpace(strings.Split(v, ",")[0])
		parts := strings.Fields(first)
		if len(parts) > 0 {
			first = parts[0]
		}
		abs := resolve(first)
		if abs == "" {
			return
		}
		add(AssetRef{URL: abs, Type: "img", Attr: "srcset", Original: first})
	})
	// video/audio sources
	doc.Find("video source[src], audio source[src]").Each(func(_ int, sel *goquery.Selection) {
		v, _ := sel.Attr("src")
		if v == "" {
			return
		}
		abs := resolve(v)
		if abs == "" {
			return
		}
		add(AssetRef{URL: abs, Type: "media", Attr: "src", Original: v})
	})
	doc.Find("link[rel='stylesheet'][href]").Each(func(_ int, sel *goquery.Selection) {
		v, _ := sel.Attr("href")
		if v == "" {
			return
		}
		abs := resolve(v)
		if abs == "" {
			return
		}
		add(AssetRef{URL: abs, Type: "stylesheet", Attr: "href", Original: v})
	})
	// Preload hints: map underlying type via 'as'
	doc.Find("link[rel='preload'][as][href]").Each(func(_ int, sel *goquery.Selection) {
		v, _ := sel.Attr("href")
		if v == "" {
			return
		}
		asVal, _ := sel.Attr("as")
		asVal = strings.ToLower(asVal)
		var t string
		switch asVal {
		case "image":
			t = "img"
		case "script":
			t = "script"
		case "style", "stylesheet":
			t = "stylesheet"
		default:
			return
		}
		abs := resolve(v)
		if abs == "" {
			return
		}
		add(AssetRef{URL: abs, Type: t, Attr: "href", Original: v})
	})
	doc.Find("script[src]").Each(func(_ int, sel *goquery.Selection) {
		v, _ := sel.Attr("src")
		if v == "" {
			return
		}
		abs := resolve(v)
		if abs == "" {
			return
		}
		add(AssetRef{URL: abs, Type: "script", Attr: "src", Original: v})
	})
	// Document assets via anchors (pdf/doc* etc.) – discovery only (policy may skip)
	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		v, _ := sel.Attr("href")
		if v == "" {
			return
		}
		lower := strings.ToLower(v)
		if !isDocLike(lower) {
			return
		}
		abs := resolve(v)
		if abs == "" {
			return
		}
		add(AssetRef{URL: abs, Type: "doc", Attr: "href", Original: v})
	})
	if s.metrics != nil {
		atomic.AddInt64(&s.metrics.discovered, int64(len(refs)))
	}
	return refs, nil
}
func (s *DefaultAssetStrategy) Decide(ctx context.Context, refs []AssetRef, policy AssetPolicy) ([]AssetAction, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	if !policy.Enabled {
		return nil, nil
	}
	allow := map[string]struct{}{}
	if len(policy.AllowTypes) > 0 {
		for _, t := range policy.AllowTypes {
			allow[t] = struct{}{}
		}
	}
	block := map[string]struct{}{}
	for _, t := range policy.BlockTypes {
		block[t] = struct{}{}
	}
	var actions []AssetAction
	skipped := 0
	for _, r := range refs {
		if _, blocked := block[r.Type]; blocked {
			skipped++
			continue
		}
		if len(allow) > 0 {
			if _, ok := allow[r.Type]; !ok {
				skipped++
				continue
			}
		}
		mode := AssetModeDownload
		if policy.InlineMaxBytes > 0 && looksInlineCandidate(r.URL) {
			mode = AssetModeInline
		}
		actions = append(actions, AssetAction{Ref: r, Mode: mode})
		if policy.MaxPerPage > 0 && len(actions) >= policy.MaxPerPage {
			break
		}
	}
	if s.metrics != nil {
		atomic.AddInt64(&s.metrics.selected, int64(len(actions)))
		atomic.AddInt64(&s.metrics.skipped, int64(skipped))
		inlineCount := int64(0)
		for _, a := range actions {
			if a.Mode == AssetModeInline {
				inlineCount++
			}
		}
		if inlineCount > 0 {
			atomic.AddInt64(&s.metrics.inlined, inlineCount)
		}
	}
	return actions, nil
}
func (s *DefaultAssetStrategy) Execute(ctx context.Context, actions []AssetAction, policy AssetPolicy) ([]MaterializedAsset, error) {
	if !policy.Enabled || len(actions) == 0 {
		return nil, nil
	}
	// Filter to executable actions
	filtered := make([]AssetAction, 0, len(actions))
	for _, a := range actions {
		if a.Mode == AssetModeDownload || a.Mode == AssetModeInline {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == 0 {
		return nil, nil
	}

	workerCount := policy.MaxConcurrent
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
		if workerCount > 8 { // conservative cap
			workerCount = 8
		}
	}
	if workerCount > len(filtered) {
		workerCount = len(filtered)
	}

	jobs := make(chan AssetAction)
	results := make(chan MaterializedAsset, len(filtered))
	var wg sync.WaitGroup
	var totalBytes int64 // cumulative pre-optimization bytes fetched

	worker := func() {
		defer wg.Done()
		for a := range jobs {
			if ctx.Err() != nil { // context cancelled
				return
			}
			if policy.MaxBytes > 0 && atomic.LoadInt64(&totalBytes) >= policy.MaxBytes {
				continue
			}
			var capRemaining int64
			if policy.MaxBytes > 0 {
				capRemaining = policy.MaxBytes - atomic.LoadInt64(&totalBytes)
				if capRemaining <= 0 {
					continue
				}
			}
			start := time.Now()
			b, err := fetchAsset(ctx, a.Ref.URL, capRemaining)
			if err != nil {
				if s.metrics != nil {
					atomic.AddInt64(&s.metrics.failed, 1)
				}
				if s.events != nil {
					s.events.Publish(AssetEvent{Type: "asset_stage_error", Stage: "execute", URL: a.Ref.URL, Error: err.Error()})
				}
				continue
			}
			preLen := len(b)
			// account toward total cap
			atomic.AddInt64(&totalBytes, int64(preLen))

			optimizations := []string{}
			if policy.Optimize {
				b2, applied := optimizeBytes(a.Ref.Type, b)
				if len(applied) > 0 {
					optimizations = applied
					b = b2
				}
			}
			postLen := len(b)

			hash := hashBytesHex(b)
			path := computeAssetPath(policy.RewritePrefix, hash, a.Ref.URL)
			ma := MaterializedAsset{Ref: a.Ref, Bytes: b, Hash: hash, Path: path, Size: postLen, Optimizations: optimizations}

			if s.metrics != nil {
				atomic.AddInt64(&s.metrics.downloaded, 1)
				if len(optimizations) > 0 {
					atomic.AddInt64(&s.metrics.optimized, 1)
				}
				atomic.AddInt64(&s.metrics.bytesIn, int64(preLen))
				atomic.AddInt64(&s.metrics.bytesOut, int64(postLen))
			}
			if s.events != nil {
				s.events.Publish(AssetEvent{Type: "asset_download", Stage: "execute", URL: a.Ref.URL, BytesIn: preLen, BytesOut: postLen, Duration: time.Since(start), Optimizations: optimizations})
			}
			results <- ma
		}
	}

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}
	for _, a := range filtered {
		jobs <- a
	}
	close(jobs)
	wg.Wait()
	close(results)

	assets := make([]MaterializedAsset, 0, len(filtered))
	for m := range results {
		assets = append(assets, m)
	}
	return assets, nil
}
func (s *DefaultAssetStrategy) Rewrite(ctx context.Context, page *engmodels.Page, assets []MaterializedAsset, policy AssetPolicy) (*engmodels.Page, error) {
	if page == nil || len(assets) == 0 || !policy.Enabled {
		return page, nil
	}
	content := page.Content
	sort.Slice(assets, func(i, j int) bool { return assets[i].Hash < assets[j].Hash })
	for _, a := range assets {
		if a.Ref.Original == "" {
			continue
		}
		esc := regexp.QuoteMeta(a.Ref.Original)
		re := regexp.MustCompile(esc)
		content = re.ReplaceAllString(content, a.Path)
	}
	cloned := *page
	cloned.Content = content
	return &cloned, nil
}

// guessExtFromURL returns a best-effort extension from URL path.
func guessExtFromURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	p := parsed.Path
	if idx := strings.LastIndex(p, "."); idx != -1 && idx+1 < len(p) {
		ext := p[idx:]
		if len(ext) <= 10 && regexp.MustCompile(`^[a-zA-Z0-9\.]+$`).MatchString(ext) {
			return ext
		}
	}
	return ""
}

// Validation placeholder: ensure rewrite prefix has leading & trailing slash semantics.
func (p AssetPolicy) Validate() error {
	if p.Enabled && !strings.HasPrefix(p.RewritePrefix, "/") {
		return errors.New("asset rewrite prefix must start with /")
	}
	return nil
}

// looksInlineCandidate provides a cheap heuristic for likely small assets that are safe to inline.
// Future iterations will replace with actual size probing or HEAD requests.
func looksInlineCandidate(u string) bool {
	lu := strings.ToLower(u)
	if strings.HasSuffix(lu, ".svg") {
		return true
	}
	if strings.Contains(lu, "icon") {
		return true
	}
	if strings.Contains(lu, "logo") {
		return true
	}
	return false
}

// Helpers (Iteration 4)
func hashBytesHex(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func computeAssetPath(prefix, hash, urlStr string) string {
	if prefix == "" {
		prefix = "/assets/"
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	ext := guessExtFromURL(urlStr)
	return prefix + hash[:2] + "/" + hash + ext
}
func optimizeBytes(assetType string, in []byte) ([]byte, []string) {
	t := strings.ToLower(assetType)
	switch t {
	case "stylesheet", "css":
		collapsed := collapseSpaces(in)
		if len(collapsed) < len(in) {
			return collapsed, []string{"css_minify"}
		}
	case "script", "js":
		collapsed := collapseSpaces(in)
		if len(collapsed) < len(in) {
			return collapsed, []string{"js_minify"}
		}
	case "img", "image":
		return in, []string{"img_meta"}
	}
	return in, nil
}
func collapseSpaces(in []byte) []byte {
	var b strings.Builder
	b.Grow(len(in))
	lastSpace := false
	for _, r := range string(in) {
		if unicode.IsSpace(r) {
			if lastSpace {
				continue
			}
			lastSpace = true
			b.WriteByte(' ')
			continue
		}
		lastSpace = false
		b.WriteRune(r)
	}
	return []byte(b.String())
}

// isDocLike returns true if the URL (lower-cased) ends with a recognized document extension.
// Extracted to simplify conditional logic (lint: De Morgan's law suggestion).
func isDocLike(lower string) bool {
	return strings.HasSuffix(lower, ".pdf") ||
		strings.HasSuffix(lower, ".doc") ||
		strings.HasSuffix(lower, ".docx") ||
		strings.HasSuffix(lower, ".ppt") ||
		strings.HasSuffix(lower, ".pptx") ||
		strings.HasSuffix(lower, ".xls") ||
		strings.HasSuffix(lower, ".xlsx")
}

// --- Internal fetch layer (Iteration 4 testability enhancement) ---
// fetchAsset is overrideable in tests to avoid real network calls.
var fetchAsset = func(ctx context.Context, rawURL string, capRemaining int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		_ = resp.Body.Close()
		return nil, errors.New("non-200")
	}
	var reader io.Reader = resp.Body
	if capRemaining > 0 {
		reader = io.LimitReader(resp.Body, capRemaining)
	}
	b, err := io.ReadAll(reader)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return b, nil
}
