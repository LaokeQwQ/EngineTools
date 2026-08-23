// Package apperror defines the public error surface crossing the Wails
// boundary: stable machine-readable codes, i18n message keys, and a
// retryability hint. Causes stay inside the process; PublicFrom strips them
// so paths, SQL, or network details never reach the UI.
package apperror

import "errors"

// Error is the canonical application error wrapping a private cause.
type Error struct {
	Code       string
	MessageKey string
	Retryable  bool
	Cause      error

	opID string // optional operation correlation, set by WithOperation
}

// Error implements the error interface. The cause text stays here (for
// logs), never inside PublicError (which reaches the UI).
func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Code
	}
	return e.Code + ": " + e.Cause.Error()
}

// Unwrap exposes the cause for errors.Is / errors.As inside the process.
func (e *Error) Unwrap() error { return e.Cause }

// E builds an application error with a stable code, an i18n message key,
// and a retryability hint for the frontend.
func E(code, messageKey string, retryable bool, cause error) error {
	return &Error{Code: code, MessageKey: messageKey, Retryable: retryable, Cause: cause}
}

// WithOperation attaches an operation ID to an application error so the
// UI can correlate progress events with the failure that aborted them.
// Non-application errors pass through unchanged.
func WithOperation(err error, operationID string) error {
	var e *Error
	if errors.As(err, &e) {
		e.opID = operationID
	}
	return err
}

// PublicError is the JSON shape handed across the Wails boundary.
type PublicError struct {
	Code        string `json:"code"`
	MessageKey  string `json:"messageKey"`
	OperationID string `json:"operationId,omitempty"`
	Retryable   bool   `json:"retryable"`
}

// PublicFrom converts any error into its redacted public form. Errors not
// produced by this package collapse to the generic internal error so raw
// causes can never leak.
func PublicFrom(err error) PublicError {
	var e *Error
	if errors.As(err, &e) {
		return PublicError{
			Code:        e.Code,
			MessageKey:  e.MessageKey,
			OperationID: e.opID,
			Retryable:   e.Retryable,
		}
	}
	return PublicError{Code: "internal", MessageKey: "errors.internal"}
}

// CodeOf returns just the stable code, for tests and internal branching.
func CodeOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return "internal"
}
