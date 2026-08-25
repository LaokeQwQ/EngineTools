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
