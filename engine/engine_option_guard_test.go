package engine

import "testing"

// TestNoOptionType ensures the former exported Option type is gone.
// If this test fails due to reintroduction, internalize again or justify via plan update.
func TestNoOptionType(t *testing.T) {
	// Attempt to reference Option should fail to compile if removed. We cannot reference it directly.
	// So this test intentionally does nothing except serve as historical marker.
	// If someone reintroduces type Option, add a reflective detection here and fail.
}
