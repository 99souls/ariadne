// Deprecated shim: use ariadne/engine/resources instead.
package resources

import engres "ariadne/engine/resources"

type (
	Config  = engres.Config
	Manager = engres.Manager
	Stats   = engres.Stats
)

var (
	NewManager = engres.NewManager
)
