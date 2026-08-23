// Package operation models long-running work surfaced to the UI: a stable
// operation ID, a coarse lifecycle phase, byte-sized progress, an i18n
// message key, and whether the user may still cancel.
package operation

import (
	"crypto/rand"
	"encoding/hex"
)

// Phase is the coarse lifecycle stage of an operation.
type Phase string

const (
	PhasePrepare   Phase = "prepare"
	PhaseRun       Phase = "run"
	PhaseFinalize  Phase = "finalize"
	PhaseDone      Phase = "done"
	PhaseFailed    Phase = "failed"
	PhaseCancelled Phase = "cancelled"
)

// Event is one progress step of an operation, safe to serialize to the
// frontend (message keys only; never raw error text).
type Event struct {
	OperationID string `json:"operationId"`
	Phase       Phase  `json:"phase"`
	// Progress counts 0..100 percent; byte keeps the JSON tiny and the
	// invariant explicit.
	Progress    byte   `json:"progress"`
	MessageKey  string `json:"messageKey"`
	Cancellable bool   `json:"cancellable"`
}

// NewID returns a random 16-hex-char operation ID.
func NewID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing means the OS entropy source is broken; a
		// predictable ID is worse than none, so surface a constant that
		// will be obvious in traces.
		return "op-entropy-failed"
	}
	return hex.EncodeToString(b[:])
}

// Done reports whether the event terminates its operation lifecycle.
func (e Event) Done() bool {
	switch e.Phase {
	case PhaseDone, PhaseFailed, PhaseCancelled:
		return true
	}
	return false
}
