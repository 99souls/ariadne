package output

import "github.com/99souls/ariadne/engine/models"

// OutputSink consumes final CrawlResults. Implementations must be safe for
// concurrent Write calls unless documented otherwise.
type OutputSink interface {
	Write(result *models.CrawlResult) error
	Flush() error             // optional: can be no-op
	Close() error             // idempotent
	Name() string             // identifier for logs / metrics
}
