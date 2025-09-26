package ratelimit

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var errInvalidDomain = errors.New("ratelimit: invalid domain")

func normalizeDomain(value string) (string, error) {
    host := strings.TrimSpace(value)
    if host == "" { return "", errInvalidDomain }
    host = strings.ToLower(host)
    if strings.Contains(host, "://") { u, err := url.Parse(host); if err != nil || u.Host == "" { return "", errInvalidDomain }; host = strings.ToLower(u.Host) }
    if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") { return host, nil }
    base := host; port := ""
    if strings.ContainsRune(host, ':') { h, p, err := net.SplitHostPort(host); if err != nil { return "", errInvalidDomain }; base = strings.ToLower(h); port = p }
    if base == "" { return "", errInvalidDomain }
    if strings.Contains(base, ":") && !strings.HasPrefix(base, "[") { base = fmt.Sprintf("[%s]", base) }
    switch port { case "", "0": return base, nil; case "80": if base != "" && !strings.Contains(base, ":") { return base, nil }; case "443": return base, nil }
    if port != "" { return fmt.Sprintf("%s:%s", base, port), nil }
    return base, nil
}
