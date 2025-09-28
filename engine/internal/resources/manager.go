package resources

// Package resources contains the internal resource manager implementation that
// was previously exposed as engine/resources. It has been internalized as part
// of Wave 4 pruning (W4-04) to reduce the public API surface. External
// consumers should only rely on high-level Engine configuration and snapshot
// data rather than this implementation.

import (
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

	engmodels "github.com/99souls/ariadne/engine/models"
)

type Config struct {
	CacheCapacity      int
	MaxInFlight        int
	SpillDirectory     string
	CheckpointPath     string
	CheckpointInterval time.Duration
}

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

type Stats struct {
	CacheEntries     int
	SpillFiles       int
	InFlight         int
	CheckpointQueued int
}

func NewManager(cfg Config) (*Manager, error) {
	m := &Manager{cfg: cfg, lru: list.New(), cache: make(map[string]*list.Element), spill: make(map[string]string)}
	if cfg.MaxInFlight > 0 {
		m.slots = make(chan struct{}, cfg.MaxInFlight)
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
		m.checkpointCh = make(chan string, 1024)
		m.wg.Add(1)
		go m.checkpointLoop()
	}
	return m, nil
}

func (m *Manager) Close() error {
	if m.checkpointCh != nil {
		close(m.checkpointCh)
		m.wg.Wait()
	}
	return nil
}

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
func (m *Manager) Release() {
	if m.slots == nil {
		return
	}
	select {
	case <-m.slots:
	default:
	}
}

type cacheEntry struct {
	url  string
	page *engmodels.Page
}

func (m *Manager) StorePage(key string, page *engmodels.Page) error {
	if key == "" || page == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	pc := m.deepCopyPage(page)
	if el, ok := m.cache[key]; ok {
		el.Value.(*cacheEntry).page = pc
		m.lru.MoveToFront(el)
		return nil
	}
	el := m.lru.PushFront(&cacheEntry{url: key, page: pc})
	m.cache[key] = el
	if m.cfg.CacheCapacity > 0 {
		for len(m.cache) > m.cfg.CacheCapacity {
			m.evictOldest()
		}
	}
	return nil
}

func (m *Manager) GetPage(key string) (*engmodels.Page, bool, error) {
	if key == "" {
		return nil, false, nil
	}
	m.mu.Lock()
	if el, ok := m.cache[key]; ok {
		m.lru.MoveToFront(el)
		entry := el.Value.(*cacheEntry)
		pg := m.deepCopyPage(entry.page)
		m.mu.Unlock()
		return pg, true, nil
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
	var pg engmodels.Page
	if err := json.Unmarshal(data, &pg); err != nil {
		return nil, false, fmt.Errorf("decode spill file: %w", err)
	}
	pgPtr := &pg
	if err := m.StorePage(key, pgPtr); err != nil {
		return nil, false, err
	}
	m.mu.Lock()
	delete(m.spill, key)
	m.mu.Unlock()
	return pgPtr, true, nil
}

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

func (m *Manager) checkpointLoop() {
	defer m.wg.Done()
	interval := m.cfg.CheckpointInterval
	if interval <= 0 {
		interval = 50 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	buf := make([]string, 0, 64)
	flush := func() {
		if len(buf) == 0 {
			return
		}
		f, err := os.OpenFile(m.cfg.CheckpointPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return
		}
		w := bufio.NewWriter(f)
		for _, e := range buf {
			_, _ = fmt.Fprintln(w, e)
		}
		_ = w.Flush()
		_ = f.Close()
		buf = buf[:0]
	}
	for {
		select {
		case e, ok := <-m.checkpointCh:
			if !ok {
				flush()
				return
			}
			buf = append(buf, e)
			if len(buf) >= 64 {
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

func (m *Manager) deepCopyPage(p *engmodels.Page) *engmodels.Page {
	if p == nil {
		return nil
	}
	pc := &engmodels.Page{Title: p.Title, Content: p.Content, CleanedText: p.CleanedText, Markdown: p.Markdown, Images: make([]string, len(p.Images)), CrawledAt: p.CrawledAt, ProcessedAt: p.ProcessedAt}
	if p.URL != nil {
		u := *p.URL
		pc.URL = &u
	}
	copy(pc.Images, p.Images)
	if len(p.Links) > 0 {
		pc.Links = make([]*url.URL, len(p.Links))
		for i, l := range p.Links {
			if l != nil {
				lc := *l
				pc.Links[i] = &lc
			}
		}
	}
	pc.Metadata = engmodels.PageMeta{Author: p.Metadata.Author, Description: p.Metadata.Description, Keywords: make([]string, len(p.Metadata.Keywords)), PublishDate: p.Metadata.PublishDate, WordCount: p.Metadata.WordCount, Headers: make(map[string]string), OpenGraph: engmodels.OpenGraphMeta{Title: p.Metadata.OpenGraph.Title, Description: p.Metadata.OpenGraph.Description, Image: p.Metadata.OpenGraph.Image, URL: p.Metadata.OpenGraph.URL, Type: p.Metadata.OpenGraph.Type}}
	copy(pc.Metadata.Keywords, p.Metadata.Keywords)
	for k, v := range p.Metadata.Headers {
		pc.Metadata.Headers[k] = v
	}
	return pc
}

func hashKey(k string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(k))
	return fmt.Sprintf("%x", h.Sum64())
}
