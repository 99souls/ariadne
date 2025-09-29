package testsite

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// FetchAndNormalize retrieves a page and applies simple deterministic normalization so
// golden snapshots are stable across minor content/format shifts (whitespace, attribute ordering).
func FetchAndNormalize(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	norm := normalizeHTML(string(b))
	return norm, nil
}

var (
	wsRE       = regexp.MustCompile(`\s+`)
	dataAttrRE = regexp.MustCompile(` data-[a-zA-Z0-9_-]+="[^"]*"`)
)

func normalizeHTML(in string) string {
	// Strip data-* attributes (framework noise) and collapse whitespace
	out := dataAttrRE.ReplaceAllString(in, "")
	out = wsRE.ReplaceAllString(out, " ")
	out = strings.TrimSpace(out)
	// Keep only <title>..</title> and top-level h1..h3 for now (lightweight signal)
	var buf bytes.Buffer
	lower := strings.ToLower(out)
	// naive extraction (good enough for deterministic test site)
	for _, tag := range []string{"title", "h1", "h2", "h3"} {
		startTag := "<" + tag
		endTag := "</" + tag + ">"
		idx := 0
		for {
			s := strings.Index(lower[idx:], startTag)
			if s == -1 {
				break
			}
			s += idx
			e := strings.Index(lower[s:], endTag)
			if e == -1 {
				break
			}
			e += s + len(endTag)
			fragment := out[s:e]
			buf.WriteString(fragment)
			buf.WriteByte('\n')
			idx = e
		}
	}
	res := buf.String()
	if res == "" {
		return out
	}
	return strings.TrimSpace(res)
}
