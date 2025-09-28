package httpmock

import (
	"context"
	"testing"
	"time"
)

func TestBasicRoutesAndRegex(t *testing.T) {
	routes := []RouteSpec{
		{Pattern: "/ok", Body: "OK", Status: 200},
		{Pattern: "^/re", Regex: true, Body: "REGEX", Status: 201},
	}
	ms := NewServer(routes)
	defer ms.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cases := map[string]int{"/ok": 200, "/regex-test": 201, "/missing": 404}
	for p, want := range cases {
		resp, err := ms.MustGet(ctx, p)
		if err != nil {
			t.Fatalf("get %s: %v", p, err)
		}
		if resp.StatusCode != want {
			t.Fatalf("status %s got %d want %d", p, resp.StatusCode, want)
		}
	}
}

func TestDelayAndCancellation(t *testing.T) {
	ms := NewServer([]RouteSpec{{Pattern: "/slow", Body: "SLOW", Delay: 200 * time.Millisecond}})
	defer ms.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	// Request may time out; ensure no panic occurs.
	_, _ = ms.MustGet(ctx, "/slow")
}
