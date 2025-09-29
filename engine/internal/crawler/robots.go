package crawler

import (
	"bufio"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// robotsRules represents a minimal subset of robots.txt directives we enforce.
// Current scope: only User-agent * Disallow lines (no Crawl-delay / Allow patterns).
type robotsRules struct {
	denyAll   bool
	disallows []string // simple prefix matches
	fetchedAt time.Time
}

// robotsCache keeps per-host rules (scheme ignored; host keyed).
type robotsCache struct {
	mu    sync.RWMutex
	rules map[string]*robotsRules
}

func newRobotsCache() *robotsCache { return &robotsCache{rules: make(map[string]*robotsRules)} }

func (rc *robotsCache) get(host string) (*robotsRules, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	r, ok := rc.rules[host]
	return r, ok
}
func (rc *robotsCache) set(host string, r *robotsRules) {
	rc.mu.Lock()
	rc.rules[host] = r
	rc.mu.Unlock()
}

// fetchRobots fetches and parses robots.txt for the provided URL host if not cached.
func (c *Crawler) fetchRobots(u *url.URL) *robotsRules {
	if !c.config.RespectRobots {
		return nil
	}
	host := u.Host
	if r, ok := c.robots.get(host); ok {
		return r
	}
	robotsURL := &url.URL{Scheme: u.Scheme, Host: u.Host, Path: "/robots.txt"}
	resp, err := http.Get(robotsURL.String())
	if err != nil || resp.StatusCode >= 400 { // treat errors as allow-all
		if err == nil {
			_ = resp.Body.Close()
		}
		rr := &robotsRules{denyAll: false, fetchedAt: time.Now()}
		c.robots.set(host, rr)
		return rr
	}
	defer func() { _ = resp.Body.Close() }()
	scanner := bufio.NewScanner(resp.Body)
	active := false
	rr := &robotsRules{fetchedAt: time.Now()}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "user-agent:") {
			ua := strings.TrimSpace(line[len("user-agent:"):])
			active = (ua == "*")
			continue
		}
		if !active {
			continue
		}
		if strings.HasPrefix(lower, "disallow:") {
			path := strings.TrimSpace(line[len("disallow:"):])
			if path == "" { // empty -> allow all (reset denyAll if previously set?)
				continue
			}
			if path == "/" {
				rr.denyAll = true
			} else {
				rr.disallows = append(rr.disallows, path)
			}
		}
	}
	c.robots.set(host, rr)
	return rr
}

// allowedByRobots evaluates path allowance. Assumes domain already allowed.
func (c *Crawler) allowedByRobots(u *url.URL) bool {
	if !c.config.RespectRobots {
		return true
	}
	if u.Path == "/robots.txt" {
		return true
	} // always allow fetching robots
	rules := c.fetchRobots(u)
	if rules == nil {
		return true
	}
	if rules.denyAll {
		return false
	}
	p := u.Path
	for _, d := range rules.disallows {
		if strings.HasPrefix(p, d) {
			return false
		}
	}
	return true
}
