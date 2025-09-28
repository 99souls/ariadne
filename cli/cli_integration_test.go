package cli

import (
    "context"
    "net/http"
    "net/http/httptest"
    "os/exec"
    "regexp"
    "strings"
    "testing"
    "time"
)

// TestCLIBasicRun ensures the binary runs a short crawl with minimal flags.
// Uses `go run` to avoid separate build step; intentionally lightweight.
func TestCLIBasicRun(t *testing.T) {
	// Local test server eliminates external network dependency.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>t</title></head><body><a href="/next">n</a></body></html>`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/ariadne", "-seeds", srv.URL, "-snapshot-interval", "0")
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("cli run timed out output=%s", string(out))
	}
	if err != nil {
		t.Fatalf("cli run error: %v output=%s", err, string(out))
	}
	output := string(out)
	if !strings.Contains(output, "FINAL SNAPSHOT") {
		t.Fatalf("expected final snapshot marker in output; got: %s", output)
	}
	// Heuristic: at least one JSON object line (result) should appear (starts with '{').
	re := regexp.MustCompile(`(?m)^{.*}$`)
	matches := re.FindAllString(output, -1)
	if len(matches) == 0 {
		t.Fatalf("expected at least one JSON result line in output; got: %s", output)
	}
}

// TestCLIMetricsAndHealth ensures metrics and health endpoints start when flags provided.
func TestCLIMetricsAndHealth(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        _, _ = w.Write([]byte(`<html><body><a href="/next">n</a></body></html>`))
    }))
    defer srv.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
    defer cancel()
    cmd := exec.CommandContext(ctx, "go", "run", "./cmd/ariadne", "-seeds", srv.URL, "-snapshot-interval", "0", "-enable-metrics", "-metrics", ":19111", "-health", ":19112")
    out, err := cmd.CombinedOutput()
    if ctx.Err() == context.DeadlineExceeded {
        t.Fatalf("cli metrics/health run timed out output=%s", string(out))
    }
    if err != nil {
        t.Fatalf("cli run error: %v output=%s", err, string(out))
    }
    o := string(out)
    if !strings.Contains(o, "metrics listening on") {
        t.Fatalf("expected metrics server log line, output=%s", o)
    }
    if !strings.Contains(o, "health endpoint listening") {
        t.Fatalf("expected health endpoint log line, output=%s", o)
    }
    if !strings.Contains(o, "FINAL SNAPSHOT") {
        t.Fatalf("expected final snapshot marker, output=%s", o)
    }
}
