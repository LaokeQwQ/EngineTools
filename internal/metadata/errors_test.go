package metadata

import (
	"errors"
	"strings"
	"testing"
)

func TestPublicErrorStripsPrivateCause(t *testing.T) {
	err := E("file_unreadable", errors.New(`open C:\Users\Alice\song.mp3: access denied`))
	public := PublicError(err)
	if CodeOf(public) != "file_unreadable" {
		t.Fatalf("code = %q", CodeOf(public))
	}
	if strings.Contains(public.Error(), "Alice") || strings.Contains(public.Error(), "access denied") {
		t.Fatalf("public error leaked a private cause: %q", public.Error())
	}
	if MessageKeyOf(public) != "errors.file_unreadable" {
		t.Fatalf("message key = %q", MessageKeyOf(public))
	}
}
