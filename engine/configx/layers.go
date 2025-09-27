package configx

// Configuration layer precedence.
const (
	LayerGlobal = iota
	LayerEnvironment
	LayerDomain
	LayerSite
	LayerEphemeral
)

var layerNames = map[int]string{
	LayerGlobal:      "global",
	LayerEnvironment: "environment",
	LayerDomain:      "domain",
	LayerSite:        "site",
	LayerEphemeral:   "ephemeral",
}

// LayerName returns the human-readable name for a layer constant.
func LayerName(layer int) string {
	if name, ok := layerNames[layer]; ok {
		return name
	}
	return "unknown"
}

// LayerPrecedenceOrder returns the merge order from lowest to highest priority.
func LayerPrecedenceOrder() []int {
	return []int{LayerGlobal, LayerEnvironment, LayerDomain, LayerSite, LayerEphemeral}
}
