package stdout

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/99souls/ariadne/engine/internal/output"
	"github.com/99souls/ariadne/engine/models"
)

// Sink writes each CrawlResult as a compact JSON line to stdout.
type Sink struct {
	enc   *json.Encoder
	mu    sync.Mutex
	write *os.File
}

func New() *Sink { return &Sink{enc: json.NewEncoder(os.Stdout), write: os.Stdout} }

func (s *Sink) Write(r *models.CrawlResult) error {
	if r == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enc.Encode(r)
}

func (s *Sink) Flush() error { return nil }
func (s *Sink) Close() error { return nil }
func (s *Sink) Name() string { return "stdout-jsonl" }

// Ensure interface compliance at compile time
var _ output.OutputSink = (*Sink)(nil)

// Example helper (optional)
func Example() {
	s := New()
	_ = s.Write(&models.CrawlResult{URL: "https://example.com", Success: true})
	fmt.Println("written")
}
