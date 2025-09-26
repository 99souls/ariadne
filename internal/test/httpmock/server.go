package httpmock

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// RouteSpec defines a mocked HTTP route behavior.
type RouteSpec struct {
	Pattern     string            // substring or regex if Regex=true
	Regex       bool              // interpret Pattern as regex
	Status      int               // HTTP status code
	Body        string            // Response body
	Headers     map[string]string // Optional headers
	Delay       time.Duration     // Artificial delay
	MatchPrefix bool              // Treat Pattern as prefix if not regex
}

// MockServer wraps an httptest server with dynamic routing.
type MockServer struct {
	server  *httptest.Server
	mux     sync.RWMutex
	ordered []*RouteSpec // stable order for deterministic tests
}

// NewServer creates a new mock HTTP server from a set of routes.
func NewServer(routes []RouteSpec) *MockServer {
	ms := &MockServer{}
	// Copy slice
	ms.ordered = make([]*RouteSpec, 0, len(routes))
	for i := range routes {
		// local copy for address stability
		r := routes[i]
		if r.Status == 0 {
			r.Status = http.StatusOK
		}
		ms.ordered = append(ms.ordered, &r)
	}
	// Sort by pattern length desc to get most specific first (non-regex, non-prefix)
	sort.SliceStable(ms.ordered, func(i, j int) bool {
		return len(ms.ordered[i].Pattern) > len(ms.ordered[j].Pattern)
	})

	hs := httptest.NewServer(http.HandlerFunc(ms.handle))
	ms.server = hs
	return ms
}

// URL returns base URL of the mock server.
func (m *MockServer) URL() string { return m.server.URL }

// Close shuts down the server.
func (m *MockServer) Close() { m.server.Close() }

// handle implements matching logic.
func (m *MockServer) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	m.mux.RLock()
	defer m.mux.RUnlock()
	for _, spec := range m.ordered {
		if spec.Regex {
			matched, _ := regexp.MatchString(spec.Pattern, path)
			if !matched { continue }
		} else if spec.MatchPrefix {
			if !strings.HasPrefix(path, spec.Pattern) { continue }
		} else {
			if !strings.Contains(path, spec.Pattern) { continue }
		}

		if spec.Delay > 0 {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(spec.Delay):
			}
		}
		for k, v := range spec.Headers { w.Header().Set(k, v) }
		w.WriteHeader(spec.Status)
		_, _ = w.Write([]byte(spec.Body))
		return
	}
	log.Printf("httpmock: unmatched path %s", path)
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("not found"))
}

// MustGet allows quick GET calls (for debugging) with context.
func (m *MockServer) MustGet(ctx context.Context, path string) (*http.Response, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, m.URL()+path, nil)
	return http.DefaultClient.Do(req)
}
