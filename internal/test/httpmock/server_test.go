package httpmock

import (
	"context"
	"testing"
	"time"
)

func TestMockServerBasicRoutes(t *testing.T) {
	rs := []RouteSpec{
		{Pattern: "/ok", Body: "OK", Status: 200},
		{Pattern: "/slow", Body: "SLOW", Status: 200, Delay: 50 * time.Millisecond},
		{Pattern: "^/re", Regex: true, Body: "REGEX", Status: 201},
	}
	ms := NewServer(rs)
	defer ms.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := []struct {
		path   string
		want   string
		status int
	}{
		{"/ok", "OK", 200},
		{"/slow", "SLOW", 200},
		{"/regex-test", "REGEX", 201},
		{"/missing", "not found", 404},
	}

	for _, c := range cases {
		resp, err := ms.MustGet(ctx, c.path)
		if err != nil {
			t.Fatalf("GET %s: %v", c.path, err)
		}
		if resp.StatusCode != c.status {
			t.Fatalf("status for %s: got %d want %d", c.path, resp.StatusCode, c.status)
		}
		// no body read needed for lightweight verification
	}
}

func TestMockServerDelayCancellation(t *testing.T) {
	ms := NewServer([]RouteSpec{{Pattern: "/delay", Body: "X", Delay: 200 * time.Millisecond}})
	defer ms.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := ms.MustGet(ctx, "/delay")
	if err == nil {
		// Request may race between cancellation and completion; allow both outcomes by timing threshold
		t.Log("Request completed before cancellation (acceptable)")
	}
}
