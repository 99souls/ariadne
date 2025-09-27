package models

// Deprecated model aliases: moved to packages/engine/models.
// These aliases preserve backward compatibility; new code should import
// "github.com/99souls/ariadne/engine/models" instead. Removal planned in cleanup phase.

import engmodels "github.com/99souls/ariadne/engine/models"

type Page = engmodels.Page
type PageMeta = engmodels.PageMeta
type OpenGraphMeta = engmodels.OpenGraphMeta
type CrawlResult = engmodels.CrawlResult
type CrawlStats = engmodels.CrawlStats
