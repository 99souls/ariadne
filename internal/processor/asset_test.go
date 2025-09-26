package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	httpmock "ariadne/internal/test/httpmock"
)

// Phase 2.3 TDD Tests: Asset Management Pipeline
// Following our proven RED → GREEN → REFACTOR methodology

func TestAssetDiscovery(t *testing.T) {
	discoverer := NewAssetDiscoverer()

	t.Run("should discover images from HTML content", func(t *testing.T) {
		html := `<div>
			<img src="/static/logo.png" alt="Company Logo">
			<img src="https://cdn.example.com/banner.jpg" alt="Banner">
			<picture>
				<source srcset="/images/hero-mobile.webp" media="(max-width: 600px)">
				<img src="/images/hero-desktop.jpg" alt="Hero Image">
			</picture>
		</div>`

		assets, err := discoverer.DiscoverAssets(html, "https://example.com/")
		if err != nil {
			t.Fatalf("DiscoverAssets failed: %v", err)
		}

		imageAssets := filterAssetsByType(assets, "image")
		if len(imageAssets) != 4 {
			t.Errorf("Expected 4 image assets, found %d", len(imageAssets))
		}

		// Validate asset URLs are absolute
		for _, asset := range imageAssets {
			if !strings.HasPrefix(asset.URL, "http") {
				t.Errorf("Asset URL should be absolute: %s", asset.URL)
			}
		}
	})

	t.Run("should discover CSS and JavaScript assets", func(t *testing.T) {
		html := `<head>
			<link rel="stylesheet" href="/assets/main.css">
			<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Inter">
			<script src="/js/app.js"></script>
			<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/js/bootstrap.bundle.min.js"></script>
		</head>`

		assets, err := discoverer.DiscoverAssets(html, "https://example.com/")
		if err != nil {
			t.Fatalf("DiscoverAssets failed: %v", err)
		}

		cssAssets := filterAssetsByType(assets, "css")
		jsAssets := filterAssetsByType(assets, "javascript")

		if len(cssAssets) != 2 {
			t.Errorf("Expected 2 CSS assets, found %d", len(cssAssets))
		}
		if len(jsAssets) != 2 {
			t.Errorf("Expected 2 JavaScript assets, found %d", len(jsAssets))
		}
	})

	t.Run("should discover document and media assets", func(t *testing.T) {
		html := `<div>
			<a href="/downloads/manual.pdf">Download Manual</a>
			<a href="/docs/api-reference.doc">API Reference</a>
			<video src="/media/demo.mp4" controls></video>
			<audio src="/sounds/notification.mp3"></audio>
		</div>`

		assets, err := discoverer.DiscoverAssets(html, "https://example.com/")
		if err != nil {
			t.Fatalf("DiscoverAssets failed: %v", err)
		}

		docAssets := filterAssetsByType(assets, "document")
		mediaAssets := filterAssetsByType(assets, "media")

		if len(docAssets) != 2 {
			t.Errorf("Expected 2 document assets, found %d", len(docAssets))
		}
		if len(mediaAssets) != 2 {
			t.Errorf("Expected 2 media assets, found %d", len(mediaAssets))
		}
	})
}

func TestAssetDownloader(t *testing.T) {
	downloader := NewAssetDownloader("/tmp/test-assets")
	defer os.RemoveAll("/tmp/test-assets")

	t.Run("should download and store assets locally", func(t *testing.T) {
		// Create a test asset
		mock := httpmock.NewServer([]httpmock.RouteSpec{{Pattern: "/robots.txt", Body: "User-agent: *", Status: 200}})
		defer mock.Close()
		asset := &AssetInfo{URL: mock.URL() + "/robots.txt", Type: "document", Filename: "robots.txt", Size: 0}

		result, err := downloader.DownloadAsset(asset)
		if err != nil {
			t.Fatalf("DownloadAsset failed: %v", err)
		}

		// Verify file was created
		if !fileExists(result.LocalPath) {
			t.Error("Downloaded file should exist")
		}

		// Verify asset info was updated
		if result.Size == 0 {
			t.Error("Asset size should be updated after download")
		}

		if result.Downloaded != true {
			t.Error("Asset should be marked as downloaded")
		}
	})

	t.Run("should handle download errors gracefully", func(t *testing.T) {
		asset := &AssetInfo{
			URL:      "https://invalid-domain-that-does-not-exist.fake/image.jpg",
			Type:     "image",
			Filename: "invalid.jpg",
		}

		result, err := downloader.DownloadAsset(asset)
		if err == nil {
			t.Error("Expected download to fail for invalid URL")
		}

		if result != nil && result.Downloaded {
			t.Error("Failed download should not be marked as downloaded")
		}
	})

	t.Run("should skip download if file already exists", func(t *testing.T) {
		// First download
		mock2 := httpmock.NewServer([]httpmock.RouteSpec{{Pattern: "/robots.txt", Body: "User-agent: dup", Status: 200}})
		defer mock2.Close()
		asset1 := &AssetInfo{URL: mock2.URL() + "/robots.txt", Type: "document", Filename: "robots-duplicate.txt"}

		result1, err := downloader.DownloadAsset(asset1)
		if err != nil {
			t.Fatalf("First download failed: %v", err)
		}

		originalModTime := getFileModTime(result1.LocalPath)

		// Wait a bit then try to download again
		time.Sleep(100 * time.Millisecond)

		result2, err := downloader.DownloadAsset(asset1)
		if err != nil {
			t.Fatalf("Second download failed: %v", err)
		}

		newModTime := getFileModTime(result2.LocalPath)

		if !originalModTime.Equal(newModTime) {
			t.Error("File should not be re-downloaded if it already exists")
		}
	})
}

func TestAssetOptimizer(t *testing.T) {
	optimizer := NewAssetOptimizer()

	t.Run("should optimize image assets", func(t *testing.T) {
		// Create temp test image file
		testImagePath := createTestImageFile(t)
		defer os.Remove(testImagePath)

		asset := &AssetInfo{
			LocalPath: testImagePath,
			Type:      "image",
			Filename:  "test.jpg",
		}

		optimized, err := optimizer.OptimizeAsset(asset)
		if err != nil {
			t.Fatalf("OptimizeAsset failed: %v", err)
		}

		if optimized.OptimizedSize >= optimized.OriginalSize {
			t.Error("Optimized image should be smaller than original")
		}

		if !optimized.Optimized {
			t.Error("Asset should be marked as optimized")
		}
	})

	t.Run("should minify CSS assets", func(t *testing.T) {
		testCSSContent := `
		/* This is a CSS comment */
		.header {
			background-color: #ffffff;
			padding: 10px 20px;
			margin: 0;
		}
		
		.content    {
			font-size:   16px;
			line-height:  1.5;
		}`

		testCSSPath := createTestFile(t, "test.css", testCSSContent)
		defer os.Remove(testCSSPath)

		asset := &AssetInfo{
			LocalPath: testCSSPath,
			Type:      "css",
			Filename:  "test.css",
		}

		optimized, err := optimizer.OptimizeAsset(asset)
		if err != nil {
			t.Fatalf("OptimizeAsset failed: %v", err)
		}

		if optimized.OptimizedSize >= optimized.OriginalSize {
			t.Error("Minified CSS should be smaller than original")
		}
	})

	t.Run("should skip optimization for unsupported asset types", func(t *testing.T) {
		asset := &AssetInfo{
			Type:     "document",
			Filename: "manual.pdf",
		}

		result, err := optimizer.OptimizeAsset(asset)
		if err != nil {
			t.Fatalf("OptimizeAsset should not fail for unsupported types: %v", err)
		}

		if result.Optimized {
			t.Error("PDF assets should not be marked as optimized")
		}
	})
}

func TestAssetURLRewriter(t *testing.T) {
	rewriter := NewAssetURLRewriter("/assets/")

	t.Run("should rewrite asset URLs in HTML content", func(t *testing.T) {
		html := `<div>
			<img src="https://example.com/images/logo.png" alt="Logo">
			<link rel="stylesheet" href="https://example.com/css/main.css">
			<script src="https://example.com/js/app.js"></script>
		</div>`

		assets := []*AssetInfo{
			{URL: "https://example.com/images/logo.png", LocalPath: "/tmp/logo.png", Filename: "logo.png"},
			{URL: "https://example.com/css/main.css", LocalPath: "/tmp/main.css", Filename: "main.css"},
			{URL: "https://example.com/js/app.js", LocalPath: "/tmp/app.js", Filename: "app.js"},
		}

		rewrittenHTML, err := rewriter.RewriteAssetURLs(html, assets)
		if err != nil {
			t.Fatalf("RewriteAssetURLs failed: %v", err)
		}

		if strings.Contains(rewrittenHTML, "https://example.com/") {
			t.Error("Original URLs should be replaced with local paths")
		}

		if !strings.Contains(rewrittenHTML, "/assets/logo.png") {
			t.Error("Image URL should be rewritten to local path")
		}

		if !strings.Contains(rewrittenHTML, "/assets/main.css") {
			t.Error("CSS URL should be rewritten to local path")
		}
	})

	t.Run("should preserve relative URLs that don't match assets", func(t *testing.T) {
		html := `<a href="/page1">Internal Link</a><a href="#section">Anchor</a>`

		rewrittenHTML, err := rewriter.RewriteAssetURLs(html, []*AssetInfo{})
		if err != nil {
			t.Fatalf("RewriteAssetURLs failed: %v", err)
		}

		if !strings.Contains(rewrittenHTML, `href="/page1"`) {
			t.Error("Internal links should be preserved")
		}

		if !strings.Contains(rewrittenHTML, `href="#section"`) {
			t.Error("Anchor links should be preserved")
		}
	})
}

func TestAssetPipeline(t *testing.T) {
	t.Run("Phase 2.3: Complete asset management pipeline", func(t *testing.T) {
		// Create temporary directory for test assets
		assetDir := "/tmp/test-pipeline-assets"
		os.RemoveAll(assetDir)
		defer os.RemoveAll(assetDir)

		pipeline := NewAssetPipeline(assetDir)

		mock := httpmock.NewServer([]httpmock.RouteSpec{{Pattern: "/robots.txt", Body: "User-agent: pipeline", Status: 200}})
		defer mock.Close()
		// Test HTML with various asset types referencing mock server
		html := `<!DOCTYPE html>
		<html>
		<head>
			<title>Asset Pipeline Test</title>
			<link rel="stylesheet" href="` + mock.URL() + `/robots.txt">
		</head>
		<body>
			<img src="` + mock.URL() + `/robots.txt" alt="Test">
			<script src="` + mock.URL() + `/robots.txt"></script>
		</body>
		</html>`

		baseURL := "https://example.com/"

		// Process through complete pipeline
		result, err := pipeline.ProcessAssets(html, baseURL)
		if err != nil {
			t.Fatalf("ProcessAssets failed: %v", err)
		}

		// Validate assets were discovered
		if len(result.Assets) == 0 {
			t.Error("Assets should be discovered from HTML")
		}

		// Validate some assets were downloaded (using reliable test endpoint)
		downloadedCount := 0
		for _, asset := range result.Assets {
			if asset.Downloaded {
				downloadedCount++
			}
		}
		if downloadedCount == 0 {
			t.Error("At least some assets should be downloaded")
		}

		// Validate HTML was updated with local paths
		if strings.Contains(result.UpdatedHTML, mock.URL()) {
			t.Error("HTML should have asset URLs rewritten to local paths")
		}

		// Validate processing statistics
		if result.TotalAssets == 0 {
			t.Error("TotalAssets should be counted")
		}

		if result.ProcessingTime == 0 {
			t.Error("ProcessingTime should be recorded")
		}

		t.Logf("🔥 PHASE 2.3 SUCCESS: Processed %d assets in %v!",
			result.TotalAssets, result.ProcessingTime)
	})
}

// Helper functions
func filterAssetsByType(assets []*AssetInfo, assetType string) []*AssetInfo {
	filtered := []*AssetInfo{}
	for _, asset := range assets {
		if asset.Type == assetType {
			filtered = append(filtered, asset)
		}
	}
	return filtered
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func createTestImageFile(t *testing.T) string {
	// Create a minimal valid JPEG file for testing
	testPath := filepath.Join("/tmp", "test-image.jpg")

	// Minimal JPEG header - just enough to be recognized as an image
	jpegHeader := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0xFF, 0xD9, // EOI marker
	}

	err := os.WriteFile(testPath, jpegHeader, 0644)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	return testPath
}

func createTestFile(t *testing.T, filename, content string) string {
	testPath := filepath.Join("/tmp", filename)
	err := os.WriteFile(testPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return testPath
}
