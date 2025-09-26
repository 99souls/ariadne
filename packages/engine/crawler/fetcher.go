package crawler

import (
	"context"
	"net/url"
)

// FetchedPage is the normalized representation of a fetched resource.
type FetchedPage struct {
	URL       *url.URL
	Content   []byte
	MediaType string
	Status    int
	Links     []*url.URL
	Metadata  map[string]string
}

// Fetcher abstracts the act of retrieving a page + discovering outbound links.
type Fetcher interface {
	Fetch(ctx context.Context, rawURL string) (*FetchedPage, error)
}
