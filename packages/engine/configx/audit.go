package configx

import "time"

// AuditRecord captures immutable metadata about a committed configuration version.
type AuditRecord struct {
	Version     int64     `json:"version"`
	Hash        string    `json:"hash"`
	Actor       string    `json:"actor"`
	AppliedAt   time.Time `json:"applied_at"`
	Parent      int64     `json:"parent"`
	DiffSummary string    `json:"diff_summary,omitempty"`
}
