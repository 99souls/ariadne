package resources

import (
	"bufio"
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sync"
	"time"

	"site-scraper/pkg/models"
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
	cfg Config

	slots chan struct{}

	mu    sync.Mutex
	lru   *list.List
	cache map[string]*list.Element
	spill map[string]string

	checkpointCh chan string
	wg           sync.WaitGroup
}

type cacheEntry struct {
	url  string
	page *models.Page
}

// NewManager constructs a resource manager according to the provided configuration.
func NewManager(cfg Config) (*Manager, error) {
	manager := &Manager{
		cfg:          cfg,
		lru:          list.New(),
		cache:        make(map[string]*list.Element),
		spill:        make(map[string]string),
		checkpointCh: nil,
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

// Close shuts down the manager and flushes any pending checkpoint data.
func (m *Manager) Close() error {
	if m.checkpointCh != nil {
		close(m.checkpointCh)
		m.wg.Wait()
	}
	return nil
}

// Acquire reserves a memory slot for in-flight work; it blocks when MaxInFlight is reached.
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

// Release frees an in-flight slot previously acquired.
func (m *Manager) Release() {
	if m.slots == nil {
		return
	}

	select {
	case <-m.slots:
	default:
	}
}

// StorePage stores a page in the in-memory cache, evicting to disk if capacity is exceeded.
func (m *Manager) StorePage(key string, page *models.Page) error {
	if key == "" || page == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if element, ok := m.cache[key]; ok {
		entry := element.Value.(*cacheEntry)
		entry.page = page
		m.lru.MoveToFront(element)
		return nil
	}

	element := m.lru.PushFront(&cacheEntry{url: key, page: page})
	m.cache[key] = element

	if m.cfg.CacheCapacity > 0 {
		for len(m.cache) > m.cfg.CacheCapacity {
			m.evictOldest()
		}
	}

	return nil
}

// GetPage returns a cached page if available, loading from spillover if necessary.
func (m *Manager) GetPage(key string) (*models.Page, bool, error) {
	if key == "" {
		return nil, false, nil
	}

	m.mu.Lock()
	if element, ok := m.cache[key]; ok {
		m.lru.MoveToFront(element)
		entry := element.Value.(*cacheEntry)
		page := entry.page
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
	var page models.Page
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

// Checkpoint records the completion of a URL to durable storage.
func (m *Manager) Checkpoint(url string) {
	if m.checkpointCh == nil || url == "" {
		return
	}

	select {
	case m.checkpointCh <- url:
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
			fmt.Fprintln(writer, entry)
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

func hashKey(key string) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(key))
	return fmt.Sprintf("%x", hasher.Sum64())
}
