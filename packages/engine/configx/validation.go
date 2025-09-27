package configx

import "errors"

// Validation errors
var (
	ErrInvalidRolloutMode    = errors.New("invalid rollout mode")
	ErrPercentageOutOfRange  = errors.New("rollout percentage out of range")
	ErrNegativeConcurrency   = errors.New("negative max concurrency")
	ErrNegativeRetryConfig   = errors.New("negative retry config")
)

// ValidateSpec performs structural + semantic validation.
// This is intentionally minimal in Iteration 4; future iterations will add deeper policy checks.
func ValidateSpec(spec *EngineConfigSpec) error {
	if spec == nil { return errors.New("nil spec") }
	if spec.Rollout != nil {
		mode := spec.Rollout.Mode
		if mode == "" { mode = "full" }
		switch mode {
		case "full":
		case "percentage":
			if spec.Rollout.Percentage < 0 || spec.Rollout.Percentage > 100 { return ErrPercentageOutOfRange }
		case "cohort":
			// no further checks yet (future: ensure at least one domain/glob)
		default:
			return ErrInvalidRolloutMode
		}
	}
	if spec.Global != nil {
		if spec.Global.MaxConcurrency < 0 { return ErrNegativeConcurrency }
		if spec.Global.RetryPolicy != nil {
			if spec.Global.RetryPolicy.MaxRetries < 0 || spec.Global.RetryPolicy.InitialDelay < 0 { return ErrNegativeRetryConfig }
		}
	}
	return nil
}
