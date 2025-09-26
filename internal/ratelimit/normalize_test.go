package ratelimit

import "testing"

func TestNormalizeDomain(t *testing.T) {
	cases := map[string]string{
		"Example.COM":          "example.com",
		"sub.example.com:80":   "sub.example.com",
		"sub.example.com:443":  "sub.example.com",
		"sub.example.com:8080": "sub.example.com:8080",
		"[2001:db8::1]:443":    "[2001:db8::1]",
		"[2001:db8::1]:8443":   "[2001:db8::1]:8443",
		"xn--exmple-cua.com":   "xn--exmple-cua.com",
	}

	for input, expected := range cases {
		actual, err := normalizeDomain(input)
		if err != nil {
			t.Fatalf("normalizeDomain(%q) returned error: %v", input, err)
		}
		if actual != expected {
			t.Fatalf("normalizeDomain(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestNormalizeDomainErrors(t *testing.T) {
	errInputs := []string{"", ":80", "invalid::domain"}

	for _, input := range errInputs {
		if _, err := normalizeDomain(input); err == nil {
			t.Fatalf("normalizeDomain(%q) expected error", input)
		}
	}
}
