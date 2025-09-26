package models

import engmodels "ariadne/packages/engine/models"

// Deprecated: use engine/models errors. Aliases maintained for migration.
var (
	ErrMissingStartURL       = engmodels.ErrMissingStartURL
	ErrMissingAllowedDomains = engmodels.ErrMissingAllowedDomains
	ErrInvalidMaxDepth       = engmodels.ErrInvalidMaxDepth
	ErrURLNotAllowed         = engmodels.ErrURLNotAllowed
	ErrMaxDepthExceeded      = engmodels.ErrMaxDepthExceeded
	ErrMaxPagesExceeded      = engmodels.ErrMaxPagesExceeded
	ErrContentNotFound       = engmodels.ErrContentNotFound
	ErrHTTPError             = engmodels.ErrHTTPError
	ErrHTMLParsingFailed     = engmodels.ErrHTMLParsingFailed
	ErrMarkdownConversion    = engmodels.ErrMarkdownConversion
	ErrAssetDownloadFailed   = engmodels.ErrAssetDownloadFailed
	ErrOutputDirCreation     = engmodels.ErrOutputDirCreation
	ErrFileWriteFailed       = engmodels.ErrFileWriteFailed
	ErrTemplateExecution     = engmodels.ErrTemplateExecution
)

type CrawlError = engmodels.CrawlError

var NewCrawlError = engmodels.NewCrawlError
