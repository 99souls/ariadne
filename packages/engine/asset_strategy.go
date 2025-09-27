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
	"sort"
	"strings"
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

// DefaultAssetStrategy is a placeholder stub that performs no operations. It will
// be replaced in later iterations with migrated logic from internal/assets. For now
// it allows early wiring & tests.
type DefaultAssetStrategy struct{}

func (s *DefaultAssetStrategy) Name() string { return "noop" }

// Discover parses the HTML and extracts candidate asset references.
func (s *DefaultAssetStrategy) Discover(ctx context.Context, page *engmodels.Page) ([]AssetRef, error) {
    if page == nil || page.Content == "" || page.URL == nil { return nil, nil }
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(page.Content))
    if err != nil { return nil, err }
    var refs []AssetRef
    base := page.URL
    resolve := func(raw string) string {
        u, err := base.Parse(raw); if err != nil { return "" }
        return u.String()
    }
    doc.Find("img[src]").Each(func(_ int, sel *goquery.Selection) {
        v,_ := sel.Attr("src"); if v=="" { return }
        abs := resolve(v); if abs=="" { return }
        refs = append(refs, AssetRef{URL: abs, Type: "img", Attr: "src", Original: v})
    })
    doc.Find("link[rel='stylesheet'][href]").Each(func(_ int, sel *goquery.Selection) {
        v,_ := sel.Attr("href"); if v=="" { return }
        abs := resolve(v); if abs=="" { return }
        refs = append(refs, AssetRef{URL: abs, Type: "stylesheet", Attr: "href", Original: v})
    })
    doc.Find("script[src]").Each(func(_ int, sel *goquery.Selection) {
        v,_ := sel.Attr("src"); if v=="" { return }
        abs := resolve(v); if abs=="" { return }
        refs = append(refs, AssetRef{URL: abs, Type: "script", Attr: "src", Original: v})
    })
    return refs, nil
}
func (s *DefaultAssetStrategy) Decide(ctx context.Context, refs []AssetRef, policy AssetPolicy) ([]AssetAction, error) {
    if len(refs) == 0 { return nil, nil }
    if !policy.Enabled { return nil, nil }
    allow := map[string]struct{}{}
    if len(policy.AllowTypes) > 0 { for _, t := range policy.AllowTypes { allow[t]=struct{}{} } }
    block := map[string]struct{}{}
    for _, t := range policy.BlockTypes { block[t]=struct{}{} }
    var actions []AssetAction
    for _, r := range refs {
        if _, blocked := block[r.Type]; blocked { continue }
        if len(allow) > 0 { if _, ok := allow[r.Type]; !ok { continue } }
        mode := AssetModeDownload
        if policy.InlineMaxBytes > 0 && looksInlineCandidate(r.URL) { mode = AssetModeInline }
        actions = append(actions, AssetAction{Ref: r, Mode: mode})
        if policy.MaxPerPage > 0 && len(actions) >= policy.MaxPerPage { break }
    }
    return actions, nil
}
func (s *DefaultAssetStrategy) Execute(ctx context.Context, actions []AssetAction, policy AssetPolicy) ([]MaterializedAsset, error) {
    if !policy.Enabled || len(actions) == 0 { return nil, nil }
    client := http.DefaultClient
    var out []MaterializedAsset
    var total int64
    for _, a := range actions {
        if a.Mode != AssetModeDownload && a.Mode != AssetModeInline { continue }
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.Ref.URL, nil)
        if err != nil { return out, err }
        resp, err := client.Do(req)
        if err != nil { continue }
        if resp.StatusCode != 200 { _ = resp.Body.Close(); continue }
        b, err := io.ReadAll(io.LimitReader(resp.Body, policy.MaxBytes-total))
        _ = resp.Body.Close()
        if err != nil { continue }
        total += int64(len(b))
        optim := []string{}
        if policy.Optimize {
            b2, applied := optimizeBytes(a.Ref.Type, b)
            if len(applied) > 0 { b = b2; optim = applied }
        }
        hash := hashBytesHex(b)
        path := computeAssetPath(policy.RewritePrefix, hash, a.Ref.URL)
        out = append(out, MaterializedAsset{Ref: a.Ref, Bytes: b, Hash: hash, Path: path, Size: len(b), Optimizations: optim})
        if policy.MaxBytes > 0 && total >= policy.MaxBytes { break }
    }
    return out, nil
}
func (s *DefaultAssetStrategy) Rewrite(ctx context.Context, page *engmodels.Page, assets []MaterializedAsset, policy AssetPolicy) (*engmodels.Page, error) {
    if page == nil || len(assets) == 0 || !policy.Enabled { return page, nil }
    content := page.Content
    sort.Slice(assets, func(i,j int) bool { return assets[i].Hash < assets[j].Hash })
    for _, a := range assets {
        if a.Ref.Original == "" { continue }
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
    parsed, err := url.Parse(u); if err != nil { return "" }
    p := parsed.Path
    if idx := strings.LastIndex(p, "."); idx != -1 && idx+1 < len(p) {
        ext := p[idx:]
        if len(ext) <= 10 && regexp.MustCompile(`^[a-zA-Z0-9\.]+$`).MatchString(ext) { return ext }
    }
    return ""
}

// Validation placeholder: ensure rewrite prefix has leading & trailing slash semantics.
func (p AssetPolicy) Validate() error {
    if p.Enabled && !strings.HasPrefix(p.RewritePrefix, "/") { return errors.New("asset rewrite prefix must start with /") }
    return nil
}

// looksInlineCandidate provides a cheap heuristic for likely small assets that are safe to inline.
// Future iterations will replace with actual size probing or HEAD requests.
func looksInlineCandidate(u string) bool {
	lu := strings.ToLower(u)
	if strings.HasSuffix(lu, ".svg") { return true }
	if strings.Contains(lu, "icon") { return true }
	if strings.Contains(lu, "logo") { return true }
	return false
}

// Helpers (Iteration 4)
func hashBytesHex(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func computeAssetPath(prefix, hash, urlStr string) string {
	if prefix == "" { prefix = "/assets/" }
	if !strings.HasSuffix(prefix, "/") { prefix += "/" }
	if !strings.HasPrefix(prefix, "/") { prefix = "/" + prefix }
	ext := guessExtFromURL(urlStr)
	return prefix + hash[:2] + "/" + hash + ext
}
func optimizeBytes(assetType string, in []byte) ([]byte, []string) {
	t := strings.ToLower(assetType)
	switch t {
	case "stylesheet", "css":
		collapsed := collapseSpaces(in)
		if len(collapsed) < len(in) { return collapsed, []string{"css_minify"} }
	case "script", "js":
		collapsed := collapseSpaces(in)
		if len(collapsed) < len(in) { return collapsed, []string{"js_minify"} }
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
			if lastSpace { continue }
			lastSpace = true
			b.WriteByte(' ')
			continue
		}
		lastSpace = false
		b.WriteRune(r)
	}
	return []byte(b.String())
}
