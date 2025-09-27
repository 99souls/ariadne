package engine

import (
	"context"

	engmodels "github.com/99souls/ariadne/engine/models"
)

// strategies.go consolidates primary extension point interfaces for easier discovery.
// Experimental: Interface set & signatures may evolve prior to v1.0.

// Fetcher defines how pages are fetched.
// Experimental: May add richer metadata result.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (*engmodels.Page, error)
}

// Processor transforms a fetched page into enriched content.
// Experimental: May gain streaming hooks.
type Processor interface {
	Process(ctx context.Context, page *engmodels.Page) (*engmodels.Page, error)
}

// OutputSink consumes processed pages.
// Experimental: Flush/Close semantics may narrow; prefer facade-managed lifecycle.
type OutputSink interface {
	Name() string
	Write(ctx context.Context, page *engmodels.Page) error
	Flush(ctx context.Context) error
	Close(ctx context.Context) error
}

// AssetStrategy manages asset discovery and transformation phases.
// Experimental: Lifecycle may collapse or add batch operations pre-v1.0.
// AssetStrategy (defined in asset_strategy.go) is documented here for consolidated visibility.
// Experimental: Lifecycle & method set may change pre-v1.0.
