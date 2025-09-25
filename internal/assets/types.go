package assets

import (
	"time"
)

// AssetInfo represents information about a discovered asset
type AssetInfo struct {
	URL            string        `json:"url"`
	Type           string        `json:"type"` // image, css, javascript, document, media
	Filename       string        `json:"filename"`
	LocalPath      string        `json:"local_path"`
	Size           int64         `json:"size"`
	OriginalSize   int64         `json:"original_size"`
	OptimizedSize  int64         `json:"optimized_size"`
	Downloaded     bool          `json:"downloaded"`
	Optimized      bool          `json:"optimized"`
	DiscoveredAt   time.Time     `json:"discovered_at"`
	DownloadedAt   time.Time     `json:"downloaded_at"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// AssetPipelineResult represents the result of asset processing
type AssetPipelineResult struct {
	Assets          []*AssetInfo  `json:"assets"`
	UpdatedHTML     string        `json:"updated_html"`
	TotalAssets     int           `json:"total_assets"`
	DownloadedCount int           `json:"downloaded_count"`
	OptimizedCount  int           `json:"optimized_count"`
	ProcessingTime  time.Duration `json:"processing_time"`
}

// Asset type constants
const (
	AssetTypeImage      = "image"
	AssetTypeCSS        = "css"
	AssetTypeJavaScript = "javascript"
	AssetTypeDocument   = "document"
	AssetTypeMedia      = "media"
)

// Common file extensions for asset type detection
var DocumentExtensions = []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".zip", ".rar"}
