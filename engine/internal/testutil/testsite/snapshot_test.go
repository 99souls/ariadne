package testsite

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateSnapshots launches the test site (reuse capable) and writes one normalized
// snapshot to a golden directory. In future we can assert against existing goldens;
// for now this simply (re)writes to keep the workflow lightweight.
func TestGenerateSnapshots(t *testing.T) {
	WithLiveTestSite(t, func(root string) {
		url := root + "/docs/getting-started"
		norm, err := FetchAndNormalize(url)
		if err != nil {
			t.Fatalf("fetch normalize: %v", err)
		}

		goldDir := filepath.Join("testdata", "snapshots")
		if err := os.MkdirAll(goldDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		outPath := filepath.Join(goldDir, "docs_getting_started.golden.txt")
		if err := os.WriteFile(outPath, []byte(norm+"\n"), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("wrote snapshot %s (%d bytes)", outPath, len(norm))
	})
}
