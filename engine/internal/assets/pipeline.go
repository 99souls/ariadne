package assets

import (
    "fmt"
    "time"
)

// AssetPipeline coordinates the complete asset management process
type AssetPipeline struct {
    BaseDir    string
    Discoverer *AssetDiscoverer
    Downloader *AssetDownloader
    Optimizer  *AssetOptimizer
    Rewriter   *AssetURLRewriter
}

// NewAssetPipeline creates a new AssetPipeline
func NewAssetPipeline(baseDir string) *AssetPipeline {
    return &AssetPipeline{
        BaseDir:    baseDir,
        Discoverer: NewAssetDiscoverer(),
        Downloader: NewAssetDownloader(baseDir),
        Optimizer:  NewAssetOptimizer(),
        Rewriter:   NewAssetURLRewriter("/assets/"),
    }
}

// ProcessAssets runs the complete asset management pipeline
func (ap *AssetPipeline) ProcessAssets(html, baseURL string) (*AssetPipelineResult, error) {
    if html == "" {
        return nil, fmt.Errorf("HTML content is empty")
    }

    if baseURL == "" {
        return nil, fmt.Errorf("base URL is empty")
    }

    // Track processing time
    startTime := time.Now()

    // Initialize result
    result := &AssetPipelineResult{}

    // Step 1: Discover assets in HTML
    assets, err := ap.Discoverer.DiscoverAssets(html, baseURL)
    if err != nil {
        return nil, fmt.Errorf("asset discovery failed: %w", err)
    }

    result.TotalAssets = len(assets)

    // Step 2: Download assets
    var downloadedAssets []*AssetInfo
    downloadedCount := 0

    for _, asset := range assets {
        downloaded, err := ap.Downloader.DownloadAsset(asset)
        if err != nil {
            // Continue with other assets if one fails
            continue
        }
        downloadedAssets = append(downloadedAssets, downloaded)
        downloadedCount++
    }

    result.DownloadedCount = downloadedCount

    // Step 3: Optimize assets
    var optimizedAssets []*AssetInfo
    optimizedCount := 0

    for _, asset := range downloadedAssets {
        optimized, err := ap.Optimizer.OptimizeAsset(asset)
        if err != nil {
            // Still include the asset even if optimization failed
            optimizedAssets = append(optimizedAssets, asset)
            continue
        }
        optimizedAssets = append(optimizedAssets, optimized)
        optimizedCount++
    }

    result.OptimizedCount = optimizedCount

    // Step 4: Rewrite HTML URLs
    rewrittenHTML, err := ap.Rewriter.RewriteAssetURLs(html, optimizedAssets)
    if err != nil {
        return nil, fmt.Errorf("URL rewriting failed: %w", err)
    }

    // Populate final result
    result.UpdatedHTML = rewrittenHTML
    result.Assets = optimizedAssets
    result.ProcessingTime = time.Since(startTime)

    return result, nil
}
