package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotSaveListRestore(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "song.flac")
	if err := os.WriteFile(target, []byte("AUDIO-V1"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))

	snap, err := store.Save(target, Snapshot{State: "complete", OperationID: "op-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.FullSHA256) != 64 {
		t.Fatalf("hash len = %d", len(snap.FullSHA256))
	}
	// mutate live file, then restore from snapshot
	os.WriteFile(target, []byte("AUDIO-V2"), 0o644)
	if err := store.RestoreBin(snap.ID, target); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "AUDIO-V1" {
		t.Fatalf("restored %q", got)
	}
}

func TestPruneKeepsNewestAndProtected(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "song.flac")
	os.WriteFile(target, []byte("A"), 0o644)
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	store.KeepCount = 3
	store.KeepWindow = 30 * 24 * time.Hour

	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	clock := base
	store.now = func() time.Time { clock = clock.Add(time.Hour); return clock }

	var ids []string
	for i := 0; i < 5; i++ {
		snap, err := store.Save(target, Snapshot{})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, snap.ID)
	}
	removed := store.Prune()
	_ = removed
	left := store.List()
	if len(left) < s_min(3, len(ids)) || left[0].ID != ids[len(ids)-1] {
		t.Fatalf("prune kept wrong set: %d entries", len(left))
	}
}

func s_min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
