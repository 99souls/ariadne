package engine

// Phase 5D Iteration 1: Asset strategy interface & foundational types.
// This file introduces additive types; no runtime wiring yet. Tests will
// assert structural presence and basic default behaviors.

import (
    "context"
    engmodels "ariadne/packages/engine/models"
)

// AssetRef represents a discovered asset reference inside a page.
type AssetRef struct {
    URL      string
    Type     string // e.g. img, script, stylesheet
    Attr     string // attribute name (src, href, data-src)
    Original string // original raw attribute value
}

// AssetMode describes the handling decision for an asset.
type AssetMode int

const (
    AssetModeDownload AssetMode = iota
    AssetModeSkip
    AssetModeInline
    AssetModeRewrite
)

// AssetAction couples a reference with a decided handling mode.
type AssetAction struct {
    Ref  AssetRef
    Mode AssetMode
}

// MaterializedAsset represents an asset after execution (download / inline / optimization).
type MaterializedAsset struct {
    Ref           AssetRef
    Bytes         []byte
    Hash          string   // sha256
    Path          string   // stable relative path
    Size          int      // original size in bytes
    Optimizations []string // applied optimization identifiers
}

// AssetStrategy defines the pluggable asset handling pipeline lifecycle.
type AssetStrategy interface {
    Discover(ctx context.Context, page *engmodels.Page) ([]AssetRef, error)
    Decide(ctx context.Context, refs []AssetRef, policy AssetPolicy) ([]AssetAction, error)
    Execute(ctx context.Context, actions []AssetAction, policy AssetPolicy) ([]MaterializedAsset, error)
    Rewrite(ctx context.Context, page *engmodels.Page, assets []MaterializedAsset, policy AssetPolicy) (*engmodels.Page, error)
    Name() string
}

// DefaultAssetStrategy is a placeholder stub that performs no operations. It will
// be replaced in later iterations with migrated logic from internal/assets. For now
// it allows early wiring & tests.
type DefaultAssetStrategy struct{}

func (s *DefaultAssetStrategy) Name() string { return "noop" }

func (s *DefaultAssetStrategy) Discover(ctx context.Context, page *engmodels.Page) ([]AssetRef, error) {
    return nil, nil
}
func (s *DefaultAssetStrategy) Decide(ctx context.Context, refs []AssetRef, policy AssetPolicy) ([]AssetAction, error) {
    // No policy application yet: return empty decisions (later iterations will implement).
    return nil, nil
}
func (s *DefaultAssetStrategy) Execute(ctx context.Context, actions []AssetAction, policy AssetPolicy) ([]MaterializedAsset, error) {
    return nil, nil
}
func (s *DefaultAssetStrategy) Rewrite(ctx context.Context, page *engmodels.Page, assets []MaterializedAsset, policy AssetPolicy) (*engmodels.Page, error) {
    return page, nil
}
