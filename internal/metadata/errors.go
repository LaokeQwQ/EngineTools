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
	Code       string `json:"code"`
	MessageKey string `json:"messageKey"`
	Cause      error  `json:"-"`
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

// MessageKeyOf returns the stable localization key carried by a metadata
// error. Unknown errors intentionally map to the generic internal key.
func MessageKeyOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		if e.MessageKey != "" {
			return e.MessageKey
		}
		return "errors." + e.Code
	}
	return "errors.internal"
}

// PublicError strips the private cause before an error crosses a process or
// Wails boundary. Internal callers can continue to use errors.Is/errors.As
// on the original value, while clients only receive a stable code/key.
func PublicError(err error) error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return &Error{Code: e.Code, MessageKey: MessageKeyOf(e)}
	}
	return &Error{Code: "internal", MessageKey: "errors.internal"}
}
