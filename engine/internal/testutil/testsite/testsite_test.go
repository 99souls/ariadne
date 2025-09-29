package testsite

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestLiveTestSiteHarness(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	WithLiveTestSite(t, func(baseURL string) {
		resp, err := http.Get(baseURL + "/api/ping")
		if err != nil {
			t.Fatalf("ping request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected ping status: %d", resp.StatusCode)
		}
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode ping JSON: %v", err)
		}
		if ok, _ := body["ok"].(bool); !ok {
			t.Fatalf("expected ok=true in ping response, got %#v", body)
		}
	})
}
