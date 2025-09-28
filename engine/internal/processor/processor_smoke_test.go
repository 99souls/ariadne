package processor

import (
	"net/url"
	"testing"

	"github.com/99souls/ariadne/engine/models"
)

func TestProcessPageSmoke(t *testing.T) {
	cp := NewContentProcessor()
	u, _ := url.Parse("https://example.com/post")
	p := &models.Page{URL: u, Content: `<html><head><title>Hello</title><meta name="description" content="d"></head><body><h1>Hello</h1><p>World</p><img src="/x.png"></body></html>`}
	if err := cp.ProcessPage(p, "https://example.com"); err != nil {
		t.Fatalf("process: %v", err)
	}
	if p.Markdown == "" {
		t.Fatalf("expected markdown output")
	}
	if p.Metadata.WordCount == 0 {
		t.Fatalf("expected word count")
	}
}
