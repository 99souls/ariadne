package configx

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// StoreOption allows future extension of store construction.
type StoreOption func(*VersionedStore)

// VersionedStore maintains an append-only log of versioned configurations in-memory.
// Not safe for persistence; a file adapter may wrap it later.
type VersionedStore struct {
	mu       sync.RWMutex
	versions []*VersionedConfig // index = version-1
	audit    []*AuditRecord
}

// NewVersionedStore constructs an empty store.
func NewVersionedStore(opts ...StoreOption) *VersionedStore {
	vs := &VersionedStore{}
	for _, o := range opts {
		o(vs)
	}
	return vs
}

// NextVersion returns the next version number that would be assigned.
func (s *VersionedStore) NextVersion() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.versions) + 1)
}

// ListAudit returns a snapshot copy of audit records.
func (s *VersionedStore) ListAudit() []*AuditRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*AuditRecord, len(s.audit))
	for i, rec := range s.audit {
		if rec == nil {
			continue
		}
		c := *rec
		out[i] = &c
	}
	return out
}

// Get returns the VersionedConfig for a version number (1-based).
func (s *VersionedStore) Get(version int64) (*VersionedConfig, bool) {
	if version <= 0 {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if int(version) > len(s.versions) {
		return nil, false
	}
	vc := s.versions[version-1]
	return cloneVersioned(vc), true
}

// Head returns the latest versioned config.
func (s *VersionedStore) Head() (*VersionedConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.versions) == 0 {
		return nil, false
	}
	return cloneVersioned(s.versions[len(s.versions)-1]), true
}

var ErrHashMismatch = errors.New("hash mismatch")

// Append stores a new versioned config assigning the next version number.
// If vc.Version is non-zero it must match the expected next version.
func (s *VersionedStore) Append(spec *EngineConfigSpec, actor, diff string, parentExpected int64) (*VersionedConfig, error) {
	if spec == nil {
		return nil, errors.New("nil spec")
	}
	// Prepare canonical JSON for hashing.
	raw, err := canonicalJSON(spec)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(raw)
	hash := hex.EncodeToString(h[:])

	s.mu.Lock()
	defer s.mu.Unlock()
	version := int64(len(s.versions) + 1)
	var parent int64
	if len(s.versions) > 0 {
		parent = s.versions[len(s.versions)-1].Version
	}
	if parent != parentExpected && parentExpected != 0 {
		return nil, errors.New("parent version mismatch")
	}
	vc := &VersionedConfig{
		Version:     version,
		Spec:        cloneSpec(spec),
		Hash:        hash,
		AppliedAt:   time.Now().UTC(),
		Actor:       actor,
		Parent:      parent,
		DiffSummary: diff,
	}
	s.versions = append(s.versions, vc)
	s.audit = append(s.audit, &AuditRecord{Version: version, Hash: hash, Actor: actor, AppliedAt: vc.AppliedAt, Parent: parent, DiffSummary: diff})
	return cloneVersioned(vc), nil
}

// Verify recomputes hash for a stored version and returns error if mismatch.
func (s *VersionedStore) Verify(version int64) error {
	vc, ok := s.Get(version)
	if !ok {
		return errors.New("version not found")
	}
	raw, err := canonicalJSON(vc.Spec)
	if err != nil {
		return err
	}
	h := sha256.Sum256(raw)
	hash := hex.EncodeToString(h[:])
	if hash != vc.Hash {
		return ErrHashMismatch
	}
	return nil
}

func canonicalJSON(spec *EngineConfigSpec) ([]byte, error) {
	// Standard encoding/json with stable key order (Go 1.21+ deterministic)
	return json.Marshal(spec)
}

// clone helpers (shallow + deep for maps/slices reused from resolver patterns)
func cloneSpec(spec *EngineConfigSpec) *EngineConfigSpec {
	if spec == nil {
		return nil
	}
	c := *spec
	if spec.Global != nil {
		g := *spec.Global
		if spec.Global.RetryPolicy != nil {
			rp := *spec.Global.RetryPolicy
			g.RetryPolicy = &rp
		}
		c.Global = &g
	}
	if spec.Crawling != nil {
		cr := *spec.Crawling
		if cr.SiteRules != nil {
			cr.SiteRules = cloneSiteRulesMap(cr.SiteRules)
		}
		if cr.LinkRules != nil {
			lr := *cr.LinkRules
			cr.LinkRules = &lr
		}
		if cr.RateRules != nil {
			rr := *cr.RateRules
			if rr.SiteDelays != nil {
				sd := make(map[string]time.Duration, len(rr.SiteDelays))
				for k, v := range rr.SiteDelays {
					sd[k] = v
				}
				rr.SiteDelays = sd
			}
			cr.RateRules = &rr
		}
		c.Crawling = &cr
	}
	if spec.Processing != nil {
		pr := *spec.Processing
		if len(pr.ExtractionRules) > 0 {
			pr.ExtractionRules = cloneStringSlice(pr.ExtractionRules)
		}
		if len(pr.ProcessingSteps) > 0 {
			pr.ProcessingSteps = cloneStringSlice(pr.ProcessingSteps)
		}
		if pr.ConditionalActions != nil {
			m := make(map[string]string, len(pr.ConditionalActions))
			for k, v := range pr.ConditionalActions {
				m[k] = v
			}
			pr.ConditionalActions = m
		}
		c.Processing = &pr
	}
	if spec.Output != nil {
		o := *spec.Output
		if o.RoutingRules != nil {
			m := make(map[string]string, len(o.RoutingRules))
			for k, v := range o.RoutingRules {
				m[k] = v
			}
			o.RoutingRules = m
		}
		if len(o.QualityGates) > 0 {
			o.QualityGates = cloneStringSlice(o.QualityGates)
		}
		c.Output = &o
	}
	if spec.Policies != nil {
		p := *spec.Policies
		if p.BusinessRules != nil {
			br := make([]*PolicyRuleSpec, 0, len(p.BusinessRules))
			for _, r := range p.BusinessRules {
				if r == nil {
					continue
				}
				rr := *r
				br = append(br, &rr)
			}
			p.BusinessRules = br
		}
		if p.EnabledFlags != nil {
			ef := make(map[string]bool, len(p.EnabledFlags))
			for k, v := range p.EnabledFlags {
				ef[k] = v
			}
			p.EnabledFlags = ef
		}
		c.Policies = &p
	}
	if spec.Rollout != nil {
		r := *spec.Rollout
		if len(r.CohortDomains) > 0 {
			r.CohortDomains = cloneStringSlice(r.CohortDomains)
		}
		if len(r.CohortDomainGlobs) > 0 {
			r.CohortDomainGlobs = cloneStringSlice(r.CohortDomainGlobs)
		}
		c.Rollout = &r
	}
	return &c
}

func cloneSiteRulesMap(m map[string]*SiteCrawlerRule) map[string]*SiteCrawlerRule {
	out := make(map[string]*SiteCrawlerRule, len(m))
	for k, v := range m {
		out[k] = cloneSiteRule(v)
	}
	return out
}

func cloneVersioned(vc *VersionedConfig) *VersionedConfig {
	if vc == nil {
		return nil
	}
	c := *vc
	c.Spec = cloneSpec(vc.Spec)
	return &c
}
