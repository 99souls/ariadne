package assets

import (
	"os"
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

		if len(assets) < 3 {
			t.Errorf("Expected at least 3 assets (images), got %d", len(assets))
		}

		// Verify image assets were discovered
		imageCount := 0
		for _, asset := range assets {
			if asset.Type == AssetTypeImage {
				imageCount++
			}
		}

		if imageCount < 3 {
			t.Errorf("Expected at least 3 image assets, got %d", imageCount)
		}
	})

	t.Run("should discover CSS and JavaScript assets", func(t *testing.T) {
		html := `<head>
			<link rel="stylesheet" href="/static/css/main.css">
			<link rel="stylesheet" href="https://cdn.example.com/bootstrap.css">
			<script src="/static/js/app.js"></script>
			<script src="https://code.jquery.com/jquery.min.js"></script>
		</head>`

		assets, err := discoverer.DiscoverAssets(html, "https://example.com/")
		if err != nil {
			t.Fatalf("DiscoverAssets failed: %v", err)
		}

		// Verify CSS and JS assets were discovered
		cssCount := 0
		jsCount := 0
		for _, asset := range assets {
			switch asset.Type {
			case AssetTypeCSS:
				cssCount++
			case AssetTypeJavaScript:
				jsCount++
			}
		}

		if cssCount < 2 {
			t.Errorf("Expected at least 2 CSS assets, got %d", cssCount)
		}

		if jsCount < 2 {
			t.Errorf("Expected at least 2 JS assets, got %d", jsCount)
		}
	})

	t.Run("should discover document and media assets", func(t *testing.T) {
		html := `<div>
			<a href="/downloads/manual.pdf">Download Manual</a>
			<a href="/files/report.docx">Report</a>
			<video src="/media/intro.mp4" controls></video>
			<audio src="/audio/soundtrack.mp3" controls></audio>
		</div>`

		assets, err := discoverer.DiscoverAssets(html, "https://example.com/")
		if err != nil {
			t.Fatalf("DiscoverAssets failed: %v", err)
		}

		// Verify document and media assets were discovered
		docCount := 0
		mediaCount := 0
		for _, asset := range assets {
			switch asset.Type {
			case AssetTypeDocument:
				docCount++
			case AssetTypeMedia:
				mediaCount++
			}
		}

		if docCount < 2 {
			t.Errorf("Expected at least 2 document assets, got %d", docCount)
		}

		if mediaCount < 2 {
			t.Errorf("Expected at least 2 media assets, got %d", mediaCount)
		}
	})
}

func TestAssetDownloader(t *testing.T) {
	downloader := NewAssetDownloader("/tmp/test-assets")
	defer func() {
		if err := os.RemoveAll("/tmp/test-assets"); err != nil {
			t.Logf("Warning: failed to clean up test directory: %v", err)
		}
	}()

		// Use mock server for deterministic asset content
		mock := httpmock.NewServer([]httpmock.RouteSpec{{Pattern: "/robots.txt", Body: "User-agent: *", Status: 200}})
		defer mock.Close()

		t.Run("should download and store assets locally", func(t *testing.T) {
			asset := &AssetInfo{
				URL:      mock.URL() + "/robots.txt",
				Type:     AssetTypeDocument,
				Filename: "robots.txt",
				Size:     0,
			}
			result, err := downloader.DownloadAsset(asset)
			if err != nil {
				 t.Fatalf("DownloadAsset failed: %v", err)
			}
			if !fileExists(result.LocalPath) {
				 t.Error("Downloaded file should exist")
			}
			if result.Size == 0 {
				 t.Error("Asset size should be updated after download")
			}
			if !result.Downloaded {
				 t.Error("Asset should be marked as downloaded")
			}
		})

	t.Run("should handle download errors gracefully", func(t *testing.T) {
		// Test with invalid URL
		asset := &AssetInfo{
			URL:      "https://nonexistent-domain-12345.com/file.txt",
			Type:     AssetTypeDocument,
			Filename: "invalid.txt",
		}

		_, err := downloader.DownloadAsset(asset)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("should skip download if file already exists", func(t *testing.T) {
		// Create asset and download it first time
		mock2 := httpmock.NewServer([]httpmock.RouteSpec{{Pattern: "/robots.txt", Body: "User-agent: mock", Status: 200}})
		defer mock2.Close()
		asset := &AssetInfo{URL: mock2.URL() + "/robots.txt", Type: AssetTypeDocument, Filename: "robots-exists.txt"}

		// First download
		first, err := downloader.DownloadAsset(asset)
		if err != nil {
			t.Fatalf("First download failed: %v", err)
		}

		// Get original modification time
		originalInfo, err := os.Stat(first.LocalPath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		// Wait a bit to ensure different timestamp if file was re-downloaded
		time.Sleep(100 * time.Millisecond)

		// Second download should skip
		second, err := downloader.DownloadAsset(asset)
		if err != nil {
			t.Fatalf("Second download failed: %v", err)
		}

		// Check file wasn't re-downloaded by comparing modification times
		newInfo, err := os.Stat(second.LocalPath)
		if err != nil {
			t.Fatalf("Failed to stat file after second download: %v", err)
		}

		if !newInfo.ModTime().Equal(originalInfo.ModTime()) {
			t.Error("File should not be re-downloaded if it already exists")
		}
	})
}

func TestAssetOptimizer(t *testing.T) {
	optimizer := NewAssetOptimizer()

	t.Run("should optimize image assets", func(t *testing.T) {
		// Create temp test image file
		testImagePath := createTestImageFile(t)
		defer func() {
			if err := os.Remove(testImagePath); err != nil {
				t.Logf("Warning: failed to remove test image: %v", err)
			}
		}()

		asset := &AssetInfo{
			LocalPath: testImagePath,
			Type:      AssetTypeImage,
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
		// Create temp test CSS file
		testCSSPath := createTestCSSFile(t)
		defer func() {
			if err := os.Remove(testCSSPath); err != nil {
				t.Logf("Warning: failed to remove test CSS: %v", err)
			}
		}()

		asset := &AssetInfo{
			LocalPath: testCSSPath,
			Type:      AssetTypeCSS,
			Filename:  "test.css",
		}

		optimized, err := optimizer.OptimizeAsset(asset)
		if err != nil {
			t.Fatalf("OptimizeAsset failed: %v", err)
		}

		if optimized.OptimizedSize >= optimized.OriginalSize {
			t.Error("Optimized CSS should be smaller than original")
		}

		if !optimized.Optimized {
			t.Error("Asset should be marked as optimized")
		}
	})

	t.Run("should skip optimization for unsupported asset types", func(t *testing.T) {
		asset := &AssetInfo{
			Type:     AssetTypeDocument,
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

		// HTML should remain unchanged for non-asset URLs
		if !strings.Contains(rewrittenHTML, `href="/page1"`) {
			t.Error("Internal links should be preserved")
		}

		if !strings.Contains(rewrittenHTML, `href="#section"`) {
			t.Error("Anchor links should be preserved")
		}
	})
}

func TestAssetPipeline(t *testing.T) {
	// Create temporary directory for test assets
	assetDir := "/tmp/test-asset-pipeline"
	if err := os.RemoveAll(assetDir); err != nil {
		t.Logf("Warning: failed to remove existing test directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(assetDir); err != nil {
			t.Logf("Warning: failed to clean up test directory: %v", err)
		}
	}()

	pipeline := NewAssetPipeline(assetDir)

	t.Run("Phase 2.3: Complete asset management pipeline", func(t *testing.T) {
		mock := httpmock.NewServer([]httpmock.RouteSpec{{Pattern: "/robots.txt", Body: "User-agent: asset-pipeline", Status: 200}})
		defer mock.Close()
		html := `<!DOCTYPE html>
		<html>
		<head>
			<title>Test Page</title>
			<link rel="stylesheet" href="` + mock.URL() + `/robots.txt">
		</head>
		<body>
			<h1>Test Content</h1>
			<img src="` + mock.URL() + `/robots.txt" alt="Test Image">
			<script src="` + mock.URL() + `/robots.txt"></script>
		</body>
		</html>`

		// Process assets through complete pipeline
		result, err := pipeline.ProcessAssets(html, "https://example.com/")
		if err != nil {
			t.Fatalf("ProcessAssets failed: %v", err)
		}

		// Validate pipeline results
		if result.TotalAssets < 3 {
			t.Errorf("Expected at least 3 assets discovered, got %d", result.TotalAssets)
		}

		if result.DownloadedCount == 0 {
			t.Error("Expected some assets to be downloaded")
		}

		if result.ProcessingTime == 0 {
			t.Error("Processing time should be recorded")
		}

		if result.UpdatedHTML == "" {
			t.Error("Updated HTML should not be empty")
		}

		// Verify HTML was rewritten
		if strings.Contains(result.UpdatedHTML, mock.URL()+"/robots.txt") {
			t.Error("Original URLs should be replaced in updated HTML")
		}

		t.Logf("🔥 PHASE 2.3 SUCCESS: Processed %d assets in %v!", result.TotalAssets, result.ProcessingTime)
	})
}

// Helper function to check if file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// Helper function to create a test image file
func createTestImageFile(t *testing.T) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-image-*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp image file: %v", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			t.Logf("Warning: failed to close temp file: %v", err)
		}
	}()

	// Write some dummy image data
	imageData := []byte("fake-image-data-for-testing")
	if _, err := tmpFile.Write(imageData); err != nil {
		t.Fatalf("Failed to write image data: %v", err)
	}

	return tmpFile.Name()
}

// Helper function to create a test CSS file
func createTestCSSFile(t *testing.T) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-style-*.css")
	if err != nil {
		t.Fatalf("Failed to create temp CSS file: %v", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			t.Logf("Warning: failed to close temp file: %v", err)
		}
	}()

	// Write some CSS data
	cssData := []byte("body { margin: 0; padding: 0; }")
	if _, err := tmpFile.Write(cssData); err != nil {
		t.Fatalf("Failed to write CSS data: %v", err)
	}

	return tmpFile.Name()
}
