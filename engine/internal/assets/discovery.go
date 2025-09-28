package assets

import (
    "fmt"
    "net/url"
    "regexp"
    "strings"
    "time"

    "github.com/PuerkitoBio/goquery"
)

// AssetDiscoverer discovers assets from HTML content
type AssetDiscoverer struct {
    // Configuration for asset discovery
}

// NewAssetDiscoverer creates a new AssetDiscoverer
func NewAssetDiscoverer() *AssetDiscoverer {
    return &AssetDiscoverer{}
}

// DiscoverAssets discovers all assets from HTML content
func (ad *AssetDiscoverer) DiscoverAssets(html, baseURL string) ([]*AssetInfo, error) {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
    if err != nil {
        return nil, fmt.Errorf("failed to parse HTML: %w", err)
    }

    assets := []*AssetInfo{}

    // Parse base URL
    parsedBase, err := url.Parse(baseURL)
    if err != nil {
        return nil, fmt.Errorf("invalid base URL: %w", err)
    }

    // Discover images
    doc.Find("img").Each(func(i int, s *goquery.Selection) {
        src, exists := s.Attr("src")
        if exists && src != "" {
            asset := createAssetFromURL(src, AssetTypeImage, parsedBase)
            if asset != nil {
                assets = append(assets, asset)
            }
        }
    })

    // Discover srcset images (responsive images)
    doc.Find("source[srcset], img[srcset]").Each(func(i int, s *goquery.Selection) {
        srcset, exists := s.Attr("srcset")
        if exists && srcset != "" {
            // Parse srcset format: "url1 1x, url2 2x" or "url1 100w, url2 200w"
            sources := strings.Split(srcset, ",")
            for _, source := range sources {
                parts := strings.Fields(strings.TrimSpace(source))
                if len(parts) > 0 {
                    asset := createAssetFromURL(parts[0], AssetTypeImage, parsedBase)
                    if asset != nil {
                        assets = append(assets, asset)
                    }
                }
            }
        }
    })

    // Discover CSS files
    doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
        href, exists := s.Attr("href")
        if exists && href != "" {
            asset := createAssetFromURL(href, AssetTypeCSS, parsedBase)
            if asset != nil {
                assets = append(assets, asset)
            }
        }
    })

    // Discover JavaScript files
    doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
        src, exists := s.Attr("src")
        if exists && src != "" {
            asset := createAssetFromURL(src, AssetTypeJavaScript, parsedBase)
            if asset != nil {
                assets = append(assets, asset)
            }
        }
    })

    // Discover document links (PDFs, DOCs, etc.)
    doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
        href, exists := s.Attr("href")
        if exists && href != "" && isDocumentURL(href) {
            asset := createAssetFromURL(href, AssetTypeDocument, parsedBase)
            if asset != nil {
                assets = append(assets, asset)
            }
        }
    })

    // Discover media files (video, audio)
    doc.Find("video[src], audio[src]").Each(func(i int, s *goquery.Selection) {
        src, exists := s.Attr("src")
        if exists && src != "" {
            asset := createAssetFromURL(src, AssetTypeMedia, parsedBase)
            if asset != nil {
                assets = append(assets, asset)
            }
        }
    })

    return assets, nil
}

// createAssetFromURL creates an AssetInfo from a URL
func createAssetFromURL(rawURL, assetType string, baseURL *url.URL) *AssetInfo {
    // Resolve URL to absolute
    parsedURL, err := baseURL.Parse(rawURL)
    if err != nil {
        return nil
    }

    // Extract filename from URL path
    filename := getFilenameFromURL(parsedURL.String())
    if filename == "" {
        filename = generateFilenameFromURL(parsedURL.String(), assetType)
    }

    return &AssetInfo{
        URL:          parsedURL.String(),
        Type:         assetType,
        Filename:     filename,
        DiscoveredAt: time.Now(),
    }
}

// isDocumentURL determines if a URL points to a document
func isDocumentURL(href string) bool {
    lowerHref := strings.ToLower(href)

    for _, ext := range DocumentExtensions {
        if strings.Contains(lowerHref, ext) {
            return true
        }
    }
    return false
}

// getFilenameFromURL extracts filename from URL
func getFilenameFromURL(rawURL string) string {
    parsedURL, err := url.Parse(rawURL)
    if err != nil {
        return ""
    }

    path := parsedURL.Path
    if path == "" || path == "/" {
        return ""
    }

    // Extract last segment of path
    segments := strings.Split(strings.Trim(path, "/"), "/")
    if len(segments) > 0 {
        filename := segments[len(segments)-1]
        // Only return if it looks like a filename (has extension)
        if strings.Contains(filename, ".") {
            return filename
        }
    }

    return ""
}

// generateFilenameFromURL generates a filename when none exists in URL
func generateFilenameFromURL(rawURL, assetType string) string {
    parsedURL, err := url.Parse(rawURL)
    if err != nil {
        return "unknown"
    }

    // Use host and a hash of the URL
    host := parsedURL.Host
    if host == "" {
        host = "local"
    }

    // Clean host for filename
    host = regexp.MustCompile(`[^a-zA-Z0-9\-]`).ReplaceAllString(host, "-")

    // Generate extension based on type
    ext := getExtensionForAssetType(assetType)

    return fmt.Sprintf("%s-asset%s", host, ext)
}

// getExtensionForAssetType returns appropriate file extension for asset type
func getExtensionForAssetType(assetType string) string {
    switch assetType {
    case AssetTypeImage:
        return ".jpg"
    case AssetTypeCSS:
        return ".css"
    case AssetTypeJavaScript:
        return ".js"
    case AssetTypeDocument:
        return ".pdf"
    case AssetTypeMedia:
        return ".mp4"
    default:
        return ".bin"
    }
}
