package metadata

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeTx mutates the temp file physically (so replace/rollback paths run
// for real) and remembers the last applied title so verification can be
// asserted end to end.
type fakeTx struct {
	lastTitle string
	reads     int
	failReadN int // 1-based Read call that fails; 0 = never
}

func (f *fakeTx) Probe(path string) (Capabilities, error) {
	c, _ := CapabilitiesFor(FormatFLAC)
	return c, nil
}

func (f *fakeTx) Read(path string) (Document, error) {
	f.reads++
	if f.failReadN > 0 && f.reads >= f.failReadN {
		return Document{}, errors.New("injected read failure")
	}
	return Document{Format: FormatFLAC, Title: f.lastTitle}, nil
}

func (f *fakeTx) Apply(path string, patch Patch, _ WriteOptions) (WriteResult, error) {
	// Real TagLib rewrites only tag regions; audio payload bytes stay
	// identical. The fake models that by not touching the file at all.
	if patch.Title.Mode == Set {
		f.lastTitle = patch.Title.Value
	}
	return WriteResult{OK: true, Changed: changedFields(patch)}, nil
}

func (f *fakeTx) AudioPayloadSHA256(path string) ([32]byte, error) {
	var out [32]byte
	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	copy(out[:], data)
	return out, nil
}

func setupService(t *testing.T) (*Service, *fakeTx, string) {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "song.flac")
	if err := os.WriteFile(target, []byte("ORIGINAL-AUDIO"), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &fakeTx{}
	svc := NewService(fake, NewSnapshotStore(filepath.Join(dir, "snaps")))
	return svc, fake, target
}

func TestTransactionHappyPath(t *testing.T) {
	svc, _, target := setupService(t)
	res, err := svc.Apply(context.Background(), target,
		Patch{Title: FieldPatch[string]{Mode: Set, Value: "Hello"}})
	if err != nil || !res.OK || res.Changed != 1 {
		t.Fatalf("apply: %+v %v", res, err)
	}
	doc, _ := svc.Bridge.Read(target)
	if doc.Title != "Hello" {
		t.Fatalf("title = %q", doc.Title)
	}
	snaps := svc.Snapshots.List()
	if len(snaps) != 1 || snaps[0].State != "complete" {
		t.Fatalf("snapshots: %+v", snaps)
	}
}

func TestReadOnlyTargetFailsFast(t *testing.T) {
	svc, _, target := setupService(t)
	os.Chmod(target, 0o444)
	defer os.Chmod(target, 0o644)
	_, err := svc.Apply(context.Background(), target,
		Patch{Title: FieldPatch[string]{Mode: Set, Value: "X"}})
	if CodeOf(err) != "target_not_writable" {
		t.Fatalf("err = %v", err)
	}
}

func TestFailureAfterSnapshotKeepsOriginalPinned(t *testing.T) {
	svc, _, target := setupService(t)
	original, _ := os.ReadFile(target)
	svc.HookAfterSnapshot = func() error { return errors.New("boom") }

	_, err := svc.Apply(context.Background(), target,
		Patch{Title: FieldPatch[string]{Mode: Set, Value: "X"}})
	if CodeOf(err) != "injected_failure" {
		t.Fatalf("err = %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != string(original) {
		t.Fatal("original mutated despite injected failure")
	}
	if len(svc.Snapshots.List()) == 0 {
		t.Fatal("snapshot should stay pinned as the undo point")
	}
}
