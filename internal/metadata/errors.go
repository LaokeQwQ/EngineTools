// Package metadata defines the canonical audio-metadata document, the
// tri-state patch model, and the operations that apply them safely through
// the TagLib bridge. Every failure uses a stable code surfaced to the UI;
// causes never leave the process.
package metadata

import (
	"errors"
	"fmt"
)

// Error is the canonical metadata error.
type Error struct {
	Code       string
	MessageKey string
	Cause      error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Code
	}
	return e.Code + ": " + e.Cause.Error()
}

func (e *Error) Unwrap() error { return e.Cause }

// E builds a metadata error. The message key mirrors the code by default
// (errors.<snake_code>) unless overridden.
func E(code string, cause error) error {
	return &Error{Code: code, MessageKey: "errors." + code, Cause: cause}
}

// Ef builds a metadata error with a formatted private cause.
func Ef(code string, format string, args ...any) error {
	return &Error{Code: code, MessageKey: "errors." + code, Cause: fmt.Errorf(format, args...)}
}

// CodeOf returns the stable code for err, falling back to "internal" for
// anything not produced by this package.
func CodeOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return "internal"
}
