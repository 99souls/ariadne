// Deprecated shim: use ariadne/packages/engine/resources instead.
package resources

import engres "ariadne/packages/engine/resources"

type (
	Config  = engres.Config
	Manager = engres.Manager
	Stats   = engres.Stats
)

var (
	NewManager = engres.NewManager
)
