package assets

import (
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "time"
)

// AssetDownloader downloads assets from URLs
type AssetDownloader struct {
    BaseDir string
    timeout time.Duration
}

// NewAssetDownloader creates a new AssetDownloader
func NewAssetDownloader(baseDir string) *AssetDownloader {
    return &AssetDownloader{
        BaseDir: baseDir,
        timeout: 30 * time.Second,
    }
}

// DownloadAsset downloads a single asset
func (ad *AssetDownloader) DownloadAsset(asset *AssetInfo) (*AssetInfo, error) {
    if asset.URL == "" {
        return nil, fmt.Errorf("asset URL is empty")
    }

    // Generate local path if not set
    if asset.LocalPath == "" {
        asset.LocalPath = filepath.Join(ad.BaseDir, asset.Filename)
    }

    // Create the output directory if it doesn't exist
    outputDir := filepath.Dir(asset.LocalPath)
    err := os.MkdirAll(outputDir, 0755)
    if err != nil {
        return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
    }

    // Check if file already exists
    if _, err := os.Stat(asset.LocalPath); err == nil {
        // File exists, mark as downloaded and return
        asset.Downloaded = true
        if info, err := os.Stat(asset.LocalPath); err == nil {
            asset.Size = info.Size()
        }
        return asset, nil
    }

    // Create HTTP request
    client := &http.Client{
        Timeout: ad.timeout,
    }

    resp, err := client.Get(asset.URL)
    if err != nil {
        return nil, fmt.Errorf("failed to download asset from %s: %w", asset.URL, err)
    }
    defer func() {
        if closeErr := resp.Body.Close(); closeErr != nil {
            // Log error but don't override the main error
            fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
        }
    }()

    // Check for successful response
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to download asset from %s: HTTP %d", asset.URL, resp.StatusCode)
    }

    // Create the local file
    file, err := os.Create(asset.LocalPath)
    if err != nil {
        return nil, fmt.Errorf("failed to create local file %s: %w", asset.LocalPath, err)
    }
    defer func() {
        if closeErr := file.Close(); closeErr != nil {
            fmt.Printf("Warning: failed to close file: %v\n", closeErr)
        }
    }()

    // Copy the response body to the file
    size, err := io.Copy(file, resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to write asset to file %s: %w", asset.LocalPath, err)
    }

    // Update asset info
    asset.Downloaded = true
    asset.Size = size
    asset.DownloadedAt = time.Now()

    // Return the updated asset info
    return asset, nil
}
