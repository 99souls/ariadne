package assets

import (
    "fmt"
    "strings"

    "github.com/PuerkitoBio/goquery"
)

// AssetURLRewriter rewrites asset URLs in HTML content
type AssetURLRewriter struct {
    BaseURL string
}

// NewAssetURLRewriter creates a new AssetURLRewriter
func NewAssetURLRewriter(baseURL string) *AssetURLRewriter {
    return &AssetURLRewriter{BaseURL: baseURL}
}

// RewriteAssetURLs rewrites asset URLs in HTML content
func (aur *AssetURLRewriter) RewriteAssetURLs(html string, assets []*AssetInfo) (string, error) {
    if html == "" {
        return "", fmt.Errorf("HTML content is empty")
    }

    // Parse HTML content
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
    if err != nil {
        return "", fmt.Errorf("failed to parse HTML: %w", err)
    }

    // Create a map of original URLs to local paths for quick lookup
    urlToLocal := make(map[string]string)
    for _, asset := range assets {
        if asset.LocalPath != "" {
            // Use the base URL + filename for the rewritten URL
            localURL := strings.TrimSuffix(aur.BaseURL, "/") + "/" + asset.Filename
            urlToLocal[asset.URL] = localURL
        }
    }

    // Rewrite image sources
    doc.Find("img").Each(func(i int, s *goquery.Selection) {
        if src, exists := s.Attr("src"); exists {
            if localPath, found := urlToLocal[src]; found {
                s.SetAttr("src", localPath)
            }
        }
        // Handle srcset attribute
        if srcset, exists := s.Attr("srcset"); exists {
            newSrcset := aur.rewriteSrcset(srcset, urlToLocal)
            s.SetAttr("srcset", newSrcset)
        }
    })

    // Rewrite CSS links
    doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
        if href, exists := s.Attr("href"); exists {
            if localPath, found := urlToLocal[href]; found {
                s.SetAttr("href", localPath)
            }
        }
    })

    // Rewrite JavaScript sources
    doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
        if src, exists := s.Attr("src"); exists {
            if localPath, found := urlToLocal[src]; found {
                s.SetAttr("src", localPath)
            }
        }
    })

    // Rewrite other media elements (audio, video, etc.)
    doc.Find("audio[src], video[src]").Each(func(i int, s *goquery.Selection) {
        if src, exists := s.Attr("src"); exists {
            if localPath, found := urlToLocal[src]; found {
                s.SetAttr("src", localPath)
            }
        }
    })

    // Return the modified HTML
    modifiedHTML, err := doc.Html()
    if err != nil {
        return "", fmt.Errorf("failed to generate HTML: %w", err)
    }

    return modifiedHTML, nil
}

// rewriteSrcset rewrites URLs in a srcset attribute
func (aur *AssetURLRewriter) rewriteSrcset(srcset string, urlToLocal map[string]string) string {
    if srcset == "" {
        return srcset
    }

    // Split srcset by commas and process each source
    sources := strings.Split(srcset, ",")
    for i, source := range sources {
        source = strings.TrimSpace(source)
        // Extract URL (everything before the first space)
        parts := strings.Fields(source)
        if len(parts) > 0 {
            url := parts[0]
            if localPath, found := urlToLocal[url]; found {
                // Replace the URL with the local path, keeping any descriptors
                if len(parts) > 1 {
                    sources[i] = localPath + " " + strings.Join(parts[1:], " ")
                } else {
                    sources[i] = localPath
                }
            }
        }
    }

    return strings.Join(sources, ", ")
}
