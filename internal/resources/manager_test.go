package resources

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"site-scraper/pkg/models"
)

func TestManagerCacheStoreAndGet(t *testing.T) {
	tmp := t.TempDir()
	cfg := Config{
		CacheCapacity:      2,
		SpillDirectory:     filepath.Join(tmp, "spill"),
		CheckpointPath:     filepath.Join(tmp, "checkpoint.log"),
		CheckpointInterval: 5 * time.Millisecond,
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	pageURL, _ := url.Parse("https://example.com/test")
	page := &models.Page{URL: pageURL, Title: "test"}

	if err := mgr.StorePage(pageURL.String(), page); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	got, hit, err := mgr.GetPage(pageURL.String())
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !hit {
		t.Fatalf("expected cache hit")
	}
	if got.Title != "test" {
		t.Fatalf("expected title test, got %s", got.Title)
	}
}

func TestManagerSpillover(t *testing.T) {
	tmp := t.TempDir()
	spillDir := filepath.Join(tmp, "spill")
	cfg := Config{
		CacheCapacity:      1,
		SpillDirectory:     spillDir,
		CheckpointInterval: 5 * time.Millisecond,
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	u1, _ := url.Parse("https://example.com/1")
	u2, _ := url.Parse("https://example.com/2")

	if err := mgr.StorePage(u1.String(), &models.Page{URL: u1, Title: "one"}); err != nil {
		t.Fatalf("store1 failed: %v", err)
	}
	if err := mgr.StorePage(u2.String(), &models.Page{URL: u2, Title: "two"}); err != nil {
		t.Fatalf("store2 failed: %v", err)
	}

	entries, err := os.ReadDir(spillDir)
	if err != nil {
		t.Fatalf("read spill dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected spill entries")
	}

	// Recovery from spill should succeed
	page, hit, err := mgr.GetPage(u1.String())
	if err != nil {
		t.Fatalf("get spilled: %v", err)
	}
	if !hit {
		t.Fatalf("expected hit from spill")
	}
	if page.Title != "one" {
		t.Fatalf("expected recovered title 'one', got %s", page.Title)
	}
}

func TestManagerCheckpoint(t *testing.T) {
	tmp := t.TempDir()
	checkpoint := filepath.Join(tmp, "checkpoint.log")

	cfg := Config{
		CacheCapacity:      1,
		CheckpointPath:     checkpoint,
		CheckpointInterval: 1 * time.Millisecond,
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	mgr.Checkpoint("https://example.com/a")
	mgr.Checkpoint("https://example.com/b")

	mgr.Close()

	data, err := os.ReadFile(checkpoint)
	if err != nil {
		t.Fatalf("expected checkpoint file, got error: %v", err)
	}

	contents := string(data)
	if !containsLine(contents, "https://example.com/a") || !containsLine(contents, "https://example.com/b") {
		t.Fatalf("missing checkpoint entries: %s", contents)
	}
}

func TestManagerAcquireRelease(t *testing.T) {
	cfg := Config{MaxInFlight: 1}
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	if err := mgr.Acquire(context.Background()); err != nil {
		t.Fatalf("expected acquire success: %v", err)
	}

	acquireDone := make(chan error, 1)
	go func() {
		acquireDone <- mgr.Acquire(context.Background())
	}()

	select {
	case <-acquireDone:
		t.Fatalf("expected acquire to block until release")
	case <-time.After(20 * time.Millisecond):
		// still blocked as expected
	}

	mgr.Release()

	select {
	case err := <-acquireDone:
		if err != nil {
			t.Fatalf("expected acquire to succeed after release: %v", err)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("acquire did not complete after release")
	}
}

func containsLine(contents, target string) bool {
	for _, line := range strings.Split(strings.TrimSpace(contents), "\n") {
		if line == target {
			return true
		}
	}
	return false
}
