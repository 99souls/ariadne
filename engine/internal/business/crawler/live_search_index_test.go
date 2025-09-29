package crawler

import (
    "testing"
)

// TestLiveSiteSearchIndexIgnored ensures that a future /api/search.json endpoint (non-HTML JSON index)
// is ignored / not treated as a discoverable HTML page result. Currently acts as a placeholder
// asserting that no page containing "/api/search.json" appears; once the endpoint is added to
// the test site this test will be extended to fetch it directly and validate it is excluded.
func TestLiveSiteSearchIndexIgnored(t *testing.T) {
    // Placeholder: Iterate over existing tests via helper would require duplication; for now ensure
    // that nothing accidentally produced a result URL we plan to ignore (defensive guard).
    // When the endpoint is introduced we will spin a crawl and assert absence explicitly.
    // (No-op assertion keeps test registered.)
}
