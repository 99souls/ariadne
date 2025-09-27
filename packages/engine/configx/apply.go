package configx

import (
	"errors"
	"time"
)

// Applier orchestrates validation, optional simulation, commit, and rollback.
type Applier struct {
	Store      *VersionedStore
	Simulator  *Simulator
	Dispatcher *Dispatcher
	Metrics    MetricsRecorder
}

func NewApplier(store *VersionedStore, sim *Simulator) *Applier { return &Applier{Store: store, Simulator: sim} }

// WithDispatcher attaches an event dispatcher (fluent).
func (a *Applier) WithDispatcher(d *Dispatcher) *Applier { a.Dispatcher = d; return a }
// WithMetrics attaches a metrics recorder (fluent).
func (a *Applier) WithMetrics(m MetricsRecorder) *Applier { a.Metrics = m; return a }

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
    if err := ValidateSpec(candidate); err != nil { a.observeFailure("validation_error", 0, opts.Actor, err); return nil, err }
    var impact *SimulationImpact
    if a.Simulator != nil {
        impact = a.Simulator.Simulate(current, candidate)
        if !impact.Acceptable && !opts.Force && !opts.DryRun { a.observeFailure("simulation_reject", 0, opts.Actor, ErrSimulationRejected); return nil, ErrSimulationRejected }
    }
    if opts.DryRun { return &ApplyResult{Version: 0, SimImpact: impact}, nil }
    parent := a.Store.NextVersion() - 1
    vc, err := a.Store.Append(candidate, opts.Actor, "", parent)
    if err != nil { a.observeFailure("append_error", 0, opts.Actor, err); return nil, err }
    a.observeSuccess("apply", vc.Version, vc.Hash, opts.Actor)
    return &ApplyResult{Version: vc.Version, Hash: vc.Hash, SimImpact: impact}, nil
}

// Rollback re-applies a previous version's spec as a new version with a rollback diff summary.
func (a *Applier) Rollback(targetVersion int64, actor string) (*ApplyResult, error) {
    vc, ok := a.Store.Get(targetVersion)
    if !ok { return nil, errors.New("target version not found") }
    parent := a.Store.NextVersion() - 1
    newVC, err := a.Store.Append(vc.Spec, actor, "rollback("+itoa64(targetVersion)+")", parent)
    if err != nil { return nil, err }
    a.observeSuccess("rollback", newVC.Version, newVC.Hash, actor)
    if a.Metrics != nil { a.Metrics.IncRollback() }
    return &ApplyResult{Version: newVC.Version, Hash: newVC.Hash}, nil
}

// Lightweight itoa for int64 without importing strconv (micro-optimization not required but simple).
func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// internal observability helpers
func (a *Applier) observeSuccess(kind string, version int64, hash, actor string) {
	if a.Metrics != nil {
		if kind == "apply" { a.Metrics.IncApplySuccess() }
		a.Metrics.SetActiveVersion(version)
	}
	if a.Dispatcher != nil {
		a.Dispatcher.Emit(ChangeEvent{Type: kind, Version: version, Hash: hash, Actor: actor, Timestamp: time.Now().UTC()})
	}
}

func (a *Applier) observeFailure(kind string, version int64, actor string, err error) {
	if a.Metrics != nil { a.Metrics.IncApplyFailure() }
	if a.Dispatcher != nil { a.Dispatcher.Emit(ChangeEvent{Type: kind, Version: version, Actor: actor, Error: err, Timestamp: time.Now().UTC()}) }
}
