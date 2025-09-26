// Deprecated shim: use site-scraper/packages/engine/resources instead.
package resources

import engres "site-scraper/packages/engine/resources"

type (
    Config = engres.Config
    Manager = engres.Manager
    Stats = engres.Stats
)

var (
    NewManager = engres.NewManager
)
