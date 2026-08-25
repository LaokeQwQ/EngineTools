package metadata

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBatchAllUpdated(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for i := 0; i < 4; i++ {
		p := filepath.Join(dir, string(rune('A'+i))+".flac")
		if err := os.WriteFile(p, []byte("AUDIO"), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	fake := &fakeTx{}
	svc := &Service{Bridge: fake, Snapshots: store}
	bs := NewBatchService(svc)

	res := bs.Run(context.Background(), paths,
		Patch{Title: FieldPatch[string]{Mode: Set, Value: "T"}})
	if res.Blocked != 0 || res.Failed != 0 || res.Updated != 4 {
		t.Fatalf("result: %+v", res)
	}
}

func TestBatchPartialFailure(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.flac")
	bad := filepath.Join(dir, "bad.flac")
	os.WriteFile(good, []byte("OK"), 0o644)
	os.WriteFile(bad, []byte("LOCKED"), 0o444)

	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	svc := &Service{Bridge: &fakeTx{}, Snapshots: store}
	bs := NewBatchService(svc)

	patch := Patch{Album: FieldPatch[string]{Mode: Set, Value: "A"}}
	res := bs.Run(context.Background(), []string{good, bad}, patch)
	if res.Failed == 0 && res.Blocked == 0 {
		t.Fatal("expected at least one failure or block")
	}
}

func TestBatchCancel(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for i := 0; i < 20; i++ {
		p := filepath.Join(dir, string(rune('A'+i))+".flac")
		os.WriteFile(p, []byte("AUDIO"), 0o644)
		paths = append(paths, p)
	}
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	fake := &fakeTx{}
	svc := &Service{Bridge: fake, Snapshots: store}
	bs := NewBatchService(svc)
	bs.Workers = 1

	res := bs.Run(context.Background(), paths,
		Patch{Title: FieldPatch[string]{Mode: Set, Value: "T"}})
	if !res.Cancelled && res.Updated+res.Skipped+res.Failed+res.Blocked != res.Total {
		t.Fatalf("inconsistent result: %+v", res)
	}
}
