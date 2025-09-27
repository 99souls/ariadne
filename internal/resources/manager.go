// Deprecated shim: use github.com/99souls/ariadne/engine/resources instead.
package resources

import engres "github.com/99souls/ariadne/engine/resources"

type (
	Config  = engres.Config
	Manager = engres.Manager
	Stats   = engres.Stats
)

var (
	NewManager = engres.NewManager
)
