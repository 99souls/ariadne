package engine

// Phase 5D Iteration 1: Asset strategy interface & foundational types.
// This file introduces additive types; no runtime wiring yet. Tests will
// assert structural presence and basic default behaviors.

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
    Size          int      // original size in bytes
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

func (s *DefaultAssetStrategy) Discover(ctx context.Context, page *engmodels.Page) ([]AssetRef, error) {
    if page == nil || page.Content == "" || page.URL == nil {
        return nil, nil
    }
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(page.Content))
    if err != nil {
        return nil, err
    }
    var refs []AssetRef
    base := page.URL
    // helper closure
    resolve := func(raw string) string {
        u, err := base.Parse(raw)
        if err != nil { return "" }
        return u.String()
    }
    // images
    doc.Find("img[src]").Each(func(i int, sel *goquery.Selection) {
        v, _ := sel.Attr("src"); if v=="" {return}
        abs := resolve(v); if abs=="" {return}
        refs = append(refs, AssetRef{URL: abs, Type: "img", Attr: "src", Original: v})
    })
    // stylesheets
    doc.Find("link[rel='stylesheet'][href]").Each(func(i int, sel *goquery.Selection) {
        v,_ := sel.Attr("href"); if v=="" {return}
        abs := resolve(v); if abs=="" {return}
        refs = append(refs, AssetRef{URL: abs, Type: "stylesheet", Attr: "href", Original: v})
    })
    // scripts
    doc.Find("script[src]").Each(func(i int, sel *goquery.Selection) {
        v,_ := sel.Attr("src"); if v=="" {return}
        abs := resolve(v); if abs=="" {return}
        refs = append(refs, AssetRef{URL: abs, Type: "script", Attr: "src", Original: v})
    })
    return refs, nil
}
func (s *DefaultAssetStrategy) Decide(ctx context.Context, refs []AssetRef, policy AssetPolicy) ([]AssetAction, error) {
    if len(refs) == 0 { return nil, nil }
    // If disabled, do nothing
    if !policy.Enabled { return nil, nil }
    // build allow/block sets
    allowSet := map[string]struct{}{}
    if len(policy.AllowTypes) > 0 {
        for _, t := range policy.AllowTypes { allowSet[t] = struct{}{} }
    }
    blockSet := map[string]struct{}{}
    for _, t := range policy.BlockTypes { blockSet[t] = struct{}{} }
    var actions []AssetAction
    for _, r := range refs {
        if _, blocked := blockSet[r.Type]; blocked { continue }
        if len(allowSet) > 0 {
            if _, ok := allowSet[r.Type]; !ok { continue }
        }
        actions = append(actions, AssetAction{Ref: r, Mode: AssetModeDownload})
        if policy.MaxPerPage > 0 && len(actions) >= policy.MaxPerPage { break }
    }
    return actions, nil
}
func (s *DefaultAssetStrategy) Execute(ctx context.Context, actions []AssetAction, policy AssetPolicy) ([]MaterializedAsset, error) {
    if !policy.Enabled || len(actions) == 0 { return nil, nil }
    // basic serial downloader with size cap enforcement
    var out []MaterializedAsset
    var total int64
    client := http.DefaultClient
    for _, a := range actions {
        if a.Mode != AssetModeDownload { continue }
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.Ref.URL, nil)
        if err != nil { return out, err }
        resp, err := client.Do(req)
        if err != nil { continue }
        if resp.StatusCode != 200 { resp.Body.Close(); continue }
        b, err := io.ReadAll(io.LimitReader(resp.Body, policy.MaxBytes-total))
        resp.Body.Close()
        if err != nil { continue }
        total += int64(len(b))
        h := sha256.Sum256(b)
        // derive pseudo path: /assets/<first2>/<hash>[.ext]
        hexhash := hex.EncodeToString(h[:])
        ext := guessExtFromURL(a.Ref.URL)
        path := policy.RewritePrefix + hexhash[:2] + "/" + hexhash + ext
        out = append(out, MaterializedAsset{Ref: a.Ref, Bytes: b, Hash: hexhash, Path: path, Size: len(b)})
        if policy.MaxBytes > 0 && total >= policy.MaxBytes { break }
    }
    return out, nil
}
func (s *DefaultAssetStrategy) Rewrite(ctx context.Context, page *engmodels.Page, assets []MaterializedAsset, policy AssetPolicy) (*engmodels.Page, error) {
    if page == nil || len(assets) == 0 || !policy.Enabled { return page, nil }
    content := page.Content
    // stable order rewrite
    sort.Slice(assets, func(i,j int) bool { return assets[i].Hash < assets[j].Hash })
    for _, a := range assets {
        // naive replacement of original attribute values
        if a.Ref.Original == "" { continue }
        // escape regex special chars in original
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
    if p.Enabled {
        if !strings.HasPrefix(p.RewritePrefix, "/") { return errors.New("asset rewrite prefix must start with /") }
    }
    return nil
}
