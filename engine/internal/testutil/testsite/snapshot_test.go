package testsite

import (
	"os"
	"path/filepath"
	"testing"
	"strings"
)

// TestGenerateSnapshots launches the test site (reuse capable) and writes one normalized
// snapshot to a golden directory. In future we can assert against existing goldens;
// for now this simply (re)writes to keep the workflow lightweight.
func TestGenerateSnapshots(t *testing.T) {
	WithLiveTestSite(t, func(root string) {
		targetURL := root + "/docs/getting-started"
		norm, err := FetchAndNormalize(targetURL)
		if err != nil {
			t.Fatalf("fetch normalize: %v", err)
		}

		goldDir := filepath.Join("testdata", "snapshots")
		if err := os.MkdirAll(goldDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		goldenPath := filepath.Join(goldDir, "docs_getting_started.golden.txt")
		if os.Getenv("UPDATE_SNAPSHOTS") == "1" {
			if err := os.WriteFile(goldenPath, []byte(norm+"\n"), 0o644); err != nil {
				t.Fatalf("write golden: %v", err)
			}
			t.Logf("updated snapshot %s (%d bytes)", goldenPath, len(norm))
			return
		}
		// Read existing golden
		data, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("read golden: %v (set UPDATE_SNAPSHOTS=1 to create/update)", err)
		}
		expected := strings.TrimSpace(string(data))
		actual := strings.TrimSpace(norm)
		if expected != actual {
			// Simple diff (first differing line)
			expLines := strings.Split(expected, "\n")
			actLines := strings.Split(actual, "\n")
			max := len(expLines)
			if len(actLines) > max { max = len(actLines) }
			diffLine := -1
			for i := 0; i < max; i++ {
				var eLine, aLine string
				if i < len(expLines) { eLine = expLines[i] }
				if i < len(actLines) { aLine = actLines[i] }
				if eLine != aLine { diffLine = i; break }
			}
			if diffLine >= 0 {
				t.Fatalf("snapshot drift detected at line %d\nEXPECTED: %q\nACTUAL:   %q\nRun with UPDATE_SNAPSHOTS=1 to accept changes.", diffLine+1, expLines[diffLine], actLines[diffLine])
			}
			t.Fatalf("snapshot drift detected (length differs) run with UPDATE_SNAPSHOTS=1 to accept changes")
		}
	})
}
