package apperror

import (
	"errors"
	"strings"
	"testing"
)

func TestPublicFromRedactsCause(t *testing.T) {
	err := E("snapshot_failed", "errors.snapshotFailed", true,
		errors.New(`open C:\Users\Alice\Music\m.db: denied`))
	got := PublicFrom(err)
	if got.Code != "snapshot_failed" || !got.Retryable || strings.Contains(got.MessageKey, `C:\`) {
		t.Fatalf("unexpected public error: %#v", got)
	}
}

func TestPublicFromUnknownCollapsesToInternal(t *testing.T) {
	got := PublicFrom(errors.New("raw os detail"))
	if got.Code != "internal" || got.MessageKey != "errors.internal" || got.Retryable {
		t.Fatalf("unexpected fallback: %#v", got)
	}
}

func TestCodeOfBranching(t *testing.T) {
	if got := CodeOf(E("busy", "errors.busy", false, nil)); got != "busy" {
		t.Fatalf("CodeOf = %q", got)
	}
	if got := CodeOf(errors.New("x")); got != "internal" {
		t.Fatalf("fallback code = %q", got)
	}
}

func TestErrorsAsUnwrapsCause(t *testing.T) {
	inner := errors.New("denied")
	err := E("locked", "errors.locked", true, inner)
	if !errors.Is(err, inner) {
		t.Fatal("errors.Is lost the wrapped cause")
	}
}

func TestWithOperationCorrelates(t *testing.T) {
	err := E("op_fail", "errors.opFail", true, errors.New("boom"))
	err = WithOperation(err, "op-123")
	got := PublicFrom(err)
	if got.OperationID != "op-123" {
		t.Fatalf("operation id = %q", got.OperationID)
	}
	// foreign errors pass through untouched
	foreign := errors.New("raw")
	if got2 := WithOperation(foreign, "op-x"); got2 != foreign {
		t.Fatal("WithOperation mutated a foreign error")
	}
}
