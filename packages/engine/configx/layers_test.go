package configx

import "testing"

func TestLayerPrecedenceOrder(t *testing.T) {
	order := LayerPrecedenceOrder()
	expected := []int{LayerGlobal, LayerEnvironment, LayerDomain, LayerSite, LayerEphemeral}
	if len(order) != len(expected) {
		t.Fatalf("unexpected length: got %d want %d", len(order), len(expected))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Fatalf("order mismatch at %d: got %d want %d", i, order[i], v)
		}
	}
	// Ensure each layer name resolves and is non-empty / not unknown
	for _, layer := range order {
		name := LayerName(layer)
		if name == "unknown" || name == "" {
			t.Fatalf("layer %d produced invalid name '%s'", layer, name)
		}
	}
}

func TestLayerNameUnknown(t *testing.T) {
	if got := LayerName(999); got != "unknown" {
		t.Fatalf("expected unknown for invalid layer, got %s", got)
	}
}
