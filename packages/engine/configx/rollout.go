package configx

import (
    "hash/fnv"
    "strings"
)

// RolloutEvaluator determines which config version should be active for a given domain based on the
// rollout strategy of the latest applied configuration. If a domain is not yet included in a staged
// rollout it falls back to the previous version (if any).
type RolloutEvaluator struct { Store *VersionedStore }

func NewRolloutEvaluator(store *VersionedStore) *RolloutEvaluator { return &RolloutEvaluator{Store: store} }

// ActiveVersionForDomain returns the version number that should be considered active for the provided domain.
// If no versions exist returns 0.
func (r *RolloutEvaluator) ActiveVersionForDomain(domain string) int64 {
    head, ok := r.Store.Head()
    if !ok { return 0 }
    spec := head.Spec
    if spec == nil || spec.Rollout == nil || spec.Rollout.Mode == "full" { return head.Version }
    mode := spec.Rollout.Mode
    switch mode {
    case "percentage":
        if spec.Rollout.Percentage >= 100 { return head.Version }
        if spec.Rollout.Percentage <= 0 { return previousOrHead(r.Store, head) }
        h := fnv.New32a()
        _, _ = h.Write([]byte(strings.ToLower(domain)))
        v := h.Sum32() % 100
        if int(v) < spec.Rollout.Percentage { return head.Version }
        return previousOrHead(r.Store, head)
    case "cohort":
        domLower := strings.ToLower(domain)
        for _, d := range spec.Rollout.CohortDomains { if strings.ToLower(d) == domLower { return head.Version } }
        return previousOrHead(r.Store, head)
    default:
        // Unknown mode treated as full to avoid silent exclusion while still not blocking traffic.
        return head.Version
    }
}

func previousOrHead(store *VersionedStore, head *VersionedConfig) int64 {
    if head.Parent != 0 { return head.Parent }
    return head.Version
}