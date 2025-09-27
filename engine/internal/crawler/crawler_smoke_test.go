package crawler

import (
    "net/http"
    "net/http/httptest"
    "net/url"
    "testing"
    "time"
    "github.com/99souls/ariadne/engine/models"
)

func TestCrawlerBasic(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.URL.Path { case "/": w.Write([]byte(`<html><body><a href="/a">A</a></body></html>`)); case "/a": w.Write([]byte(`<html><body><h1>A</h1></body></html>`)); default: http.NotFound(w,r) }
    }))
    defer srv.Close()
    cfg := models.DefaultConfig()
    cfg.StartURL = srv.URL
    u, _ := url.Parse(srv.URL)
    cfg.AllowedDomains = []string{u.Host}
    cfg.MaxPages = 2
    cfg.RequestDelay = 5 * time.Millisecond
    c := New(&cfg)
    if err := c.Start(cfg.StartURL); err != nil { t.Fatalf("start: %v", err) }
    timeout := time.After(2 * time.Second)
    got := 0
    for got < 2 {
        select {
        case <-timeout: t.Fatalf("timeout waiting for results; got=%d", got)
        case _, ok := <-c.Results(): if !ok { t.Fatalf("results closed early") }; got++
        }
    }
    c.Stop()
    st := c.Stats()
    if st.ProcessedPages == 0 { t.Fatalf("expected processed pages") }
}
