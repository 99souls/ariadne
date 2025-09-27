package cli

import (
    "context"
    "os/exec"
    "strings"
    "testing"
    "time"
)

// TestCLIBasicRun ensures the binary runs a short crawl with minimal flags.
// Uses `go run` to avoid separate build step; intentionally lightweight.
func TestCLIBasicRun(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    cmd := exec.CommandContext(ctx, "go", "run", "./cmd/ariadne", "-seeds", "https://example.com", "-snapshot-interval", "0")
    out, err := cmd.CombinedOutput()
    if ctx.Err() == context.DeadlineExceeded {
        t.Fatalf("cli run timed out output=%s", string(out))
    }
    if err != nil {
        t.Fatalf("cli run error: %v output=%s", err, string(out))
    }
    if !strings.Contains(string(out), "FINAL SNAPSHOT") {
        t.Fatalf("expected final snapshot marker in output; got: %s", string(out))
    }
}