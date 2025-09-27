package assets

import "testing"

func TestDiscoverAssetsBasic(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>T</title><link rel="stylesheet" href="/styles.css"></head><body><img src="/img.png"><a href="/file.pdf">PDF</a><script src="/app.js"></script></body></html>`
	d := NewAssetDiscoverer()
	assets, err := d.DiscoverAssets(html, "https://example.com")
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(assets) == 0 {
		t.Fatalf("expected assets discovered")
	}
	var haveCSS, haveImg, havePDF, haveJS bool
	for _, a := range assets {
		switch a.Type {
		case AssetTypeCSS:
			haveCSS = true
		case AssetTypeImage:
			haveImg = true
		case AssetTypeDocument:
			havePDF = true
		case AssetTypeJavaScript:
			haveJS = true
		}
	}
	if !(haveCSS && haveImg && havePDF && haveJS) {
		t.Fatalf("missing expected asset types: css=%v img=%v pdf=%v js=%v", haveCSS, haveImg, havePDF, haveJS)
	}
}
