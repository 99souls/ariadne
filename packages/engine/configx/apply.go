package configx

import "errors"

// Applier orchestrates validation, optional simulation, commit, and rollback.
type Applier struct {
	Store      *VersionedStore
	Simulator  *Simulator
}

func NewApplier(store *VersionedStore, sim *Simulator) *Applier {
	return &Applier{Store: store, Simulator: sim}
}

// ApplyResult captures outcome of an apply attempt.
type ApplyResult struct {
	Version   int64
	Hash      string
	SimImpact *SimulationImpact
}

var (
	ErrSimulationRejected = errors.New("simulation rejected change")
)

// Apply executes the pipeline: validate -> simulate (if requested) -> (commit unless dry-run) -> return result.
func (a *Applier) Apply(current *EngineConfigSpec, candidate *EngineConfigSpec, opts ApplyOptions) (*ApplyResult, error) {
	if err := ValidateSpec(candidate); err != nil { return nil, err }
	var impact *SimulationImpact
	if a.Simulator != nil {
		impact = a.Simulator.Simulate(current, candidate)
		if !impact.Acceptable && !opts.Force && !opts.DryRun {
			return nil, ErrSimulationRejected
		}
	}
	if opts.DryRun { return &ApplyResult{Version: 0, SimImpact: impact}, nil }
    parent := a.Store.NextVersion() - 1
    vc, err := a.Store.Append(candidate, opts.Actor, "", parent)
    if err != nil { return nil, err }
    return &ApplyResult{Version: vc.Version, Hash: vc.Hash, SimImpact: impact}, nil
}

// Rollback re-applies a previous version's spec as a new version with a rollback diff summary.
func (a *Applier) Rollback(targetVersion int64, actor string) (*ApplyResult, error) {
	vc, ok := a.Store.Get(targetVersion)
    if !ok { return nil, errors.New("target version not found") }
    parent := a.Store.NextVersion() - 1
    newVC, err := a.Store.Append(vc.Spec, actor, "rollback("+itoa64(targetVersion)+")", parent)
    if err != nil { return nil, err }
    return &ApplyResult{Version: newVC.Version, Hash: newVC.Hash}, nil
}

// Lightweight itoa for int64 without importing strconv (micro-optimization not required but simple).
func itoa64(n int64) string {
    if n == 0 { return "0" }
    neg := n < 0
    if neg { n = -n }
    var buf [20]byte
    i := len(buf)
    for n > 0 {
        i--
        buf[i] = byte('0' + n%10)
        n /= 10
    }
    if neg { i--; buf[i] = '-' }
    return string(buf[i:])
}
