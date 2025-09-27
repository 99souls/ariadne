package resources

import (
	engmodels "ariadne/packages/engine/models"
	"bufio"
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Config controls resource management features such as caching, spillover, and checkpoints.
type Config struct {
	CacheCapacity      int
	MaxInFlight        int
	SpillDirectory     string
	CheckpointPath     string
	CheckpointInterval time.Duration
}

// Manager coordinates resource usage across the pipeline.
type Manager struct {
	cfg          Config
	slots        chan struct{}
	mu           sync.Mutex
	lru          *list.List
	cache        map[string]*list.Element
	spill        map[string]string
	checkpointCh chan string
	wg           sync.WaitGroup
}

// Stats provides lightweight insight into current resource manager state.
type Stats struct {
	CacheEntries     int `json:"cache_entries"`
	SpillFiles       int `json:"spill_files"`
	InFlight         int `json:"in_flight"`
	CheckpointQueued int `json:"checkpoint_queued"`
}

func (m *Manager) Stats() Stats {
	var s Stats
	m.mu.Lock()
	s.CacheEntries = len(m.cache)
	s.SpillFiles = len(m.spill)
	m.mu.Unlock()
	if m.slots != nil {
		s.InFlight = len(m.slots)
	}
	if m.checkpointCh != nil {
		s.CheckpointQueued = len(m.checkpointCh)
	}
	return s
}

type cacheEntry struct {
	url  string
	page *engmodels.Page
}

// NewManager constructs a resource manager according to the provided configuration.
func NewManager(cfg Config) (*Manager, error) {
	manager := &Manager{
		cfg:   cfg,
		lru:   list.New(),
		cache: make(map[string]*list.Element),
		spill: make(map[string]string),
	}

	if cfg.MaxInFlight > 0 {
		manager.slots = make(chan struct{}, cfg.MaxInFlight)
	}

	if cfg.SpillDirectory != "" {
		if err := os.MkdirAll(cfg.SpillDirectory, 0o755); err != nil {
			return nil, fmt.Errorf("create spill directory: %w", err)
		}
	}

	if cfg.CheckpointPath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.CheckpointPath), 0o755); err != nil {
			return nil, fmt.Errorf("create checkpoint directory: %w", err)
		}
		manager.checkpointCh = make(chan string, 1024)
		manager.wg.Add(1)
		go manager.checkpointLoop()
	}

	return manager, nil
}

// Close flushes and stops background goroutines.
func (m *Manager) Close() error {
	if m.checkpointCh != nil {
		close(m.checkpointCh)
		m.wg.Wait()
	}
	return nil
}

// Acquire reserves an in-flight slot; blocks when capacity reached.
func (m *Manager) Acquire(ctx context.Context) error {
	if m.slots == nil {
		return nil
	}
	select {
	case m.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release frees an in-flight slot.
func (m *Manager) Release() {
	if m.slots == nil {
		return
	}
	select {
	case <-m.slots:
	default:
	}
}

// StorePage caches a page, evicting oldest to spill if needed.
func (m *Manager) StorePage(key string, page *engmodels.Page) error {
	if key == "" || page == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	pageCopy := m.deepCopyPage(page)
	if element, ok := m.cache[key]; ok {
		entry := element.Value.(*cacheEntry)
		entry.page = pageCopy
		m.lru.MoveToFront(element)
		return nil
	}
	element := m.lru.PushFront(&cacheEntry{url: key, page: pageCopy})
	m.cache[key] = element
	if m.cfg.CacheCapacity > 0 {
		for len(m.cache) > m.cfg.CacheCapacity {
			m.evictOldest()
		}
	}
	return nil
}

// GetPage retrieves from cache or spill.
func (m *Manager) GetPage(key string) (*engmodels.Page, bool, error) {
	if key == "" {
		return nil, false, nil
	}
	m.mu.Lock()
	if element, ok := m.cache[key]; ok {
		m.lru.MoveToFront(element)
		entry := element.Value.(*cacheEntry)
		page := m.deepCopyPage(entry.page)
		m.mu.Unlock()
		return page, true, nil
	}
	path, spilled := m.spill[key]
	m.mu.Unlock()
	if !spilled {
		return nil, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("read spill file: %w", err)
	}
	var page engmodels.Page
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, false, fmt.Errorf("decode spill file: %w", err)
	}
	pagePtr := &page
	if err := m.StorePage(key, pagePtr); err != nil {
		return nil, false, err
	}
	m.mu.Lock()
	delete(m.spill, key)
	m.mu.Unlock()
	return pagePtr, true, nil
}

// Checkpoint records completion.
func (m *Manager) Checkpoint(u string) {
	if m.checkpointCh == nil || u == "" {
		return
	}
	select {
	case m.checkpointCh <- u:
	default:
		return
	}
}

func (m *Manager) checkpointLoop() {
	defer m.wg.Done()
	interval := m.cfg.CheckpointInterval
	if interval <= 0 {
		interval = 50 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	buffer := make([]string, 0, 64)
	flush := func() {
		if len(buffer) == 0 {
			return
		}
		file, err := os.OpenFile(m.cfg.CheckpointPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return
		}
		writer := bufio.NewWriter(file)
		for _, entry := range buffer {
			_, _ = fmt.Fprintln(writer, entry)
		}
		_ = writer.Flush()
		_ = file.Close()
		buffer = buffer[:0]
	}
	for {
		select {
		case entry, ok := <-m.checkpointCh:
			if !ok {
				flush()
				return
			}
			buffer = append(buffer, entry)
			if len(buffer) >= 64 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (m *Manager) evictOldest() {
	back := m.lru.Back()
	if back == nil {
		return
	}
	entry := back.Value.(*cacheEntry)
	delete(m.cache, entry.url)
	m.lru.Remove(back)
	if m.cfg.SpillDirectory == "" {
		return
	}
	filename := fmt.Sprintf("spill-%d-%s.spill.json", time.Now().UnixNano(), hashKey(entry.url))
	path := filepath.Join(m.cfg.SpillDirectory, filename)
	data, err := json.Marshal(entry.page)
	if err != nil {
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return
	}
	m.spill[entry.url] = path
}

// deepCopyPage creates a deep copy of a Page to prevent shared mutations
func (m *Manager) deepCopyPage(page *engmodels.Page) *engmodels.Page {
	if page == nil {
		return nil
	}
	pageCopy := &engmodels.Page{Title: page.Title, Content: page.Content, CleanedText: page.CleanedText, Markdown: page.Markdown, Images: make([]string, len(page.Images)), CrawledAt: page.CrawledAt, ProcessedAt: page.ProcessedAt}
	if page.URL != nil {
		urlCopy := *page.URL
		pageCopy.URL = &urlCopy
	}
	copy(pageCopy.Images, page.Images)
	if len(page.Links) > 0 {
		pageCopy.Links = make([]*url.URL, len(page.Links))
		for i, link := range page.Links {
			if link != nil {
				linkCopy := *link
				pageCopy.Links[i] = &linkCopy
			}
		}
	}
	pageCopy.Metadata = engmodels.PageMeta{Author: page.Metadata.Author, Description: page.Metadata.Description, Keywords: make([]string, len(page.Metadata.Keywords)), PublishDate: page.Metadata.PublishDate, WordCount: page.Metadata.WordCount, Headers: make(map[string]string), OpenGraph: engmodels.OpenGraphMeta{Title: page.Metadata.OpenGraph.Title, Description: page.Metadata.OpenGraph.Description, Image: page.Metadata.OpenGraph.Image, URL: page.Metadata.OpenGraph.URL, Type: page.Metadata.OpenGraph.Type}}
	copy(pageCopy.Metadata.Keywords, page.Metadata.Keywords)
	for k, v := range page.Metadata.Headers {
		pageCopy.Metadata.Headers[k] = v
	}
	return pageCopy
}

func hashKey(key string) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(key))
	return fmt.Sprintf("%x", hasher.Sum64())
}
