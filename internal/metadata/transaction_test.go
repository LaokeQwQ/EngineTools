package metadata

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type fakeTx struct {
	mu        sync.Mutex
	lastTitle string
	reads     int
	failReadN int
}

func (f *fakeTx) Probe(path string) (Capabilities, error) {
	c, _ := CapabilitiesFor(FormatFLAC)
	return c, nil
}

func (f *fakeTx) Read(path string) (Document, error) {
	f.mu.Lock()
	f.reads++
	fail := f.failReadN > 0 && f.reads >= f.failReadN
	title := f.lastTitle
	f.mu.Unlock()
	if fail {
		return Document{}, errors.New("injected read failure")
	}
	return Document{Format: FormatFLAC, Title: title}, nil
}

func (f *fakeTx) Apply(path string, patch Patch, _ WriteOptions) (WriteResult, error) {
	f.mu.Lock()
	if patch.Title.Mode == Set {
		f.lastTitle = patch.Title.Value
	}
	f.mu.Unlock()
	return WriteResult{OK: true, Changed: changedFields(patch)}, nil
}

func (f *fakeTx) AudioPayloadSHA256(path string) ([32]byte, error) {
	var out [32]byte
	copy(out[:], []byte("audio"))
	return out, nil
}

func setupService(t *testing.T) (*Service, *fakeTx, string) {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "song.flac")
	if err := os.WriteFile(target, []byte("ORIGINAL-AUDIO"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	fake := &fakeTx{}
	svc := NewService(fake, store)
	return svc, fake, target
}

func TestTransactionHappyPath(t *testing.T) {
	svc, _, target := setupService(t)
	res, err := svc.Apply(context.Background(), target,
		Patch{Title: FieldPatch[string]{Mode: Set, Value: "Hello"}})
	if err != nil || !res.OK || res.Changed != 1 {
		t.Fatalf("apply: %+v %v", res, err)
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
		t.Fatal("snapshot should stay pinned")
	}
}

func TestRollbackRestoresOriginalOnPostReplaceFailure(t *testing.T) {
	svc, fake, target := setupService(t)
	original, _ := os.ReadFile(target)
	fake.failReadN = 3 // fail on final verification read

	patch := Patch{Title: FieldPatch[string]{Mode: Set, Value: "Will Roll Back"}}
	_, err := svc.Apply(context.Background(), target, patch)
	if err == nil {
		t.Fatal("expected final verification failure")
	}
	got, _ := os.ReadFile(target)
	if string(got) != string(original) {
		t.Fatalf("rollback did not restore original: %q", got)
	}
}

func TestHydratePicturePatchUsesCurrentPayloadByID(t *testing.T) {
	full := makeTestJPEG(8, 8)
	current := []Picture{{ID: "apic-0", Type: 3, MIME: "image/jpeg", Data: full, Bytes: int64(len(full))}}
	patch := Patch{Pictures: FieldPatch[[]Picture]{Mode: Set, Value: []Picture{{
		ID: "apic-0", Type: 3, MIME: "image/jpeg", Bytes: int64(len(full)),
	}}}}
	hydrated := hydratePicturePatch(patch, current)
	if string(hydrated.Pictures.Value[0].Data) != string(full) {
		t.Fatal("picture payload was not hydrated")
	}
	if len(patch.Pictures.Value[0].Data) != 0 {
		t.Fatal("hydration mutated caller patch")
	}
}

func TestTransactionDetectsEqualSizeTimestampReplacement(t *testing.T) {
	svc, _, target := setupService(t)
	originalInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	svc.HookAfterSnapshot = func() error {
		content, readErr := os.ReadFile(target)
		if readErr != nil {
			return readErr
		}
		content[0] ^= 0x01
		if writeErr := os.WriteFile(target, content, 0o644); writeErr != nil {
			return writeErr
		}
		return os.Chtimes(target, originalInfo.ModTime(), originalInfo.ModTime())
	}
	_, err = svc.Apply(context.Background(), target,
		Patch{Title: FieldPatch[string]{Mode: Set, Value: "external"}})
	if CodeOf(err) != "external_change" {
		t.Fatalf("equal-size/timestamp replacement was not detected: %v", err)
	}
}
