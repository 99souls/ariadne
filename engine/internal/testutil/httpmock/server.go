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

type RouteSpec struct {
	Pattern     string
	Regex       bool
	Status      int
	Body        string
	Headers     map[string]string
	Delay       time.Duration
	MatchPrefix bool
}

type MockServer struct {
	server  *httptest.Server
	mux     sync.RWMutex
	ordered []*RouteSpec
}

func NewServer(routes []RouteSpec) *MockServer {
	ms := &MockServer{}
	ms.ordered = make([]*RouteSpec, 0, len(routes))
	for i := range routes { r := routes[i]; if r.Status == 0 { r.Status = http.StatusOK }; ms.ordered = append(ms.ordered, &r) }
	sort.SliceStable(ms.ordered, func(i, j int) bool { return len(ms.ordered[i].Pattern) > len(ms.ordered[j].Pattern) })
	ms.server = httptest.NewServer(http.HandlerFunc(ms.handle))
	return ms
}

func (m *MockServer) URL() string { return m.server.URL }
func (m *MockServer) Close() { m.server.Close() }

func (m *MockServer) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	m.mux.RLock(); defer m.mux.RUnlock()
	for _, spec := range m.ordered {
		if spec.Regex { matched, _ := regexp.MatchString(spec.Pattern, path); if !matched { continue } } else if spec.MatchPrefix { if !strings.HasPrefix(path, spec.Pattern) { continue } } else { if !strings.Contains(path, spec.Pattern) { continue } }
		if spec.Delay > 0 { select { case <-r.Context().Done(): return; case <-time.After(spec.Delay): } }
		for k,v := range spec.Headers { w.Header().Set(k,v) }
		w.WriteHeader(spec.Status); _, _ = w.Write([]byte(spec.Body)); return
	}
	log.Printf("httpmock: unmatched path %s", path)
	w.WriteHeader(http.StatusNotFound); _, _ = w.Write([]byte("not found"))
}

func (m *MockServer) MustGet(ctx context.Context, path string) (*http.Response, error) { req, _ := http.NewRequestWithContext(ctx, http.MethodGet, m.URL()+path, nil); return http.DefaultClient.Do(req) }
