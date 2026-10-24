package metadata

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAntiPiracyV1Clear(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for i := 0; i < 3; i++ {
		p := filepath.Join(dir, string(rune('A'+i))+".flac")
		os.WriteFile(p, []byte("AUDIO"), 0o644)
		paths = append(paths, p)
	}
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	fake := &fakeTx{}
	svc := &Service{Bridge: fake, Snapshots: store}
	bs := NewBatchService(svc)
	ap := NewAntiPiracyService(bs, store, fake)

	res, err := ap.ClearV1(context.Background(), paths)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 3 || res.Failed != 0 {
		t.Fatalf("result: %+v", res)
	}
}

func TestAntiPiracyRefusesSingleFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "one.flac")
	os.WriteFile(p, []byte("A"), 0o644)

	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	fake := &fakeTx{}
	svc := &Service{Bridge: fake, Snapshots: store}
	bs := NewBatchService(svc)
	ap := NewAntiPiracyService(bs, store, fake)

	_, err := ap.ClearV1(context.Background(), []string{p})
	if err == nil || CodeOf(err) != "antipiracy_too_few" {
		t.Fatalf("err = %v", err)
	}
}

func TestAntiPiracyDerangeV2(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for i := 0; i < 3; i++ {
		p := filepath.Join(dir, string(rune('A'+i))+".flac")
		os.WriteFile(p, []byte("AUDIO"), 0o644)
		paths = append(paths, p)
	}
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	fake := &fakeTx{}
	svc := &Service{Bridge: fake, Snapshots: store}
	bs := NewBatchService(svc)
	ap := NewAntiPiracyService(bs, store, fake)

	res, err := ap.DerangeV2(context.Background(), paths)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 3 {
		t.Fatalf("updated = %d", res.Updated)
	}
}

func TestSattoloPermutationHasNoFixedPoints(t *testing.T) {
	for n := 2; n <= 8; n++ {
		perm, err := sattoloPermutation(n)
		if err != nil {
			t.Fatal(err)
		}
		if len(perm) != n {
			t.Fatalf("n=%d len=%d", n, len(perm))
		}
		for i, j := range perm {
			if i == j {
				t.Fatalf("n=%d fixed point at %d: %v", n, i, perm)
			}
		}
	}
}

func TestAntiPiracySelectedFieldPatch(t *testing.T) {
	fields, err := resolveAntiFields([]string{"title", "artists", "lyrics", "title"})
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 3 {
		t.Fatalf("fields = %v", fields)
	}
	source := Document{
		Title:   "Title",
		Artists: []string{"Artist"},
		Lyrics:  "Lyrics",
	}
	patch := setAntiPatch(fields, source)
	if patch.Title.Mode != Set || patch.Title.Value != "Title" {
		t.Fatalf("title patch = %+v", patch.Title)
	}
	if patch.Artists.Mode != Set || len(patch.Artists.Value) != 1 {
		t.Fatalf("artists patch = %+v", patch.Artists)
	}
	if patch.Lyrics.Mode != Set || patch.Lyrics.Value != "Lyrics" {
		t.Fatalf("lyrics patch = %+v", patch.Lyrics)
	}
	if patch.BPM.Mode != "" || patch.Rating.Mode != "" || patch.Pictures.Mode != "" {
		t.Fatal("non-selected fields must remain untouched")
	}
	if _, err := resolveAntiFields([]string{"rating"}); CodeOf(err) != "antipiracy_field_invalid" {
		t.Fatalf("rating field accepted: %v", err)
	}
}

func TestAntiPiracyV1SnapshotsBeforeWriteAndReportsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	goodA := filepath.Join(dir, "a.flac")
	goodB := filepath.Join(dir, "b.flac")
	missing := filepath.Join(dir, "missing.flac")
	for _, path := range []string{goodA, goodB} {
		if err := os.WriteFile(path, []byte("AUDIO"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	svc := NewService(&fakeTx{}, store)
	ap := NewAntiPiracyService(NewBatchService(svc), store, &fakeTx{})

	result, err := ap.ClearV1(context.Background(), []string{goodA, missing, goodB})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 2 || result.Failed != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	// At least the two eligible files must have a recovery snapshot before
	// Batch.Run starts its own writes.
	if len(store.List()) < 2 {
		t.Fatalf("pre-write snapshots missing: %d", len(store.List()))
	}
}

func TestAntiPiracyFieldsFollowContainerCapabilities(t *testing.T) {
	wav, ok := CapabilitiesFor(FormatWAV)
	if !ok {
		t.Fatal("missing WAV capabilities")
	}
	doc := Document{Format: FormatWAV, Capabilities: &wav}
	fields := fieldsForDocument(doc, antiTextFieldNames)
	allowed := map[string]bool{}
	for _, field := range fields {
		allowed[field] = true
	}
	for _, field := range []string{"title", "artists", "album", "date", "genres", "comment"} {
		if !allowed[field] {
			t.Fatalf("WAV should support %q, got %v", field, fields)
		}
	}
	for _, field := range []string{"key", "isrc", "lyrics", "composers"} {
		if allowed[field] {
			t.Fatalf("WAV should not expose %q, got %v", field, fields)
		}
	}
}
