package metadata

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestBatchDuplicatePathsBlocked(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "song.flac")
	if err := os.WriteFile(p, []byte("AUDIO"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	fake := &fakeTx{}
	svc := &Service{Bridge: fake, Snapshots: store}
	bs := NewBatchService(svc)

	patch := Patch{Title: FieldPatch[string]{Mode: Set, Value: "T"}}

	prev, err := bs.Preview(context.Background(), []string{p, p}, patch)
	if err != nil {
		t.Fatal(err)
	}
	if prev.Total != 2 || prev.Eligible != 1 || prev.Blocked != 1 {
		t.Fatalf("unexpected preview: %+v", prev)
	}
	var dupBlocked bool
	for _, it := range prev.Items {
		if it.Status == "blocked" && it.Code == "duplicate_path" {
			dupBlocked = true
		}
	}
	if !dupBlocked {
		t.Fatalf("expected duplicate_path blocked in preview: %+v", prev.Items)
	}

	res := bs.Run(context.Background(), []string{p, p}, patch)
	if res.Total != 2 || res.Updated != 1 || res.Blocked != 1 {
		t.Fatalf("unexpected run result: %+v", res)
	}
	dupBlocked = false
	for _, it := range res.Items {
		if it.Status == "blocked" && it.Code == "duplicate_path" {
			dupBlocked = true
		}
	}
	if !dupBlocked {
		t.Fatalf("expected duplicate_path blocked in run: %+v", res.Items)
	}
}

func TestBatchPreflightCancel(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "song.flac")
	if err := os.WriteFile(p, []byte("AUDIO"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	fake := &fakeTx{}
	svc := &Service{Bridge: fake, Snapshots: store}
	bs := NewBatchService(svc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	patch := Patch{Title: FieldPatch[string]{Mode: Set, Value: "T"}}
	res := bs.Run(ctx, []string{p}, patch)
	if !res.Cancelled {
		t.Fatal("expected batch result to be cancelled")
	}
	if len(res.Items) != 1 || res.Items[0].Status != "skipped" {
		t.Fatalf("expected item to be skipped, got: %+v", res.Items)
	}
}

func TestBatchStartRegistersResultAndCancelBeforeReturning(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "song.flac")
	if err := os.WriteFile(p, []byte("AUDIO"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewSnapshotStore(filepath.Join(dir, "snaps"))
	bs := NewBatchService(&Service{Bridge: &fakeTx{}, Snapshots: store})

	opID := bs.Start(context.Background(), []string{p},
		Patch{Title: FieldPatch[string]{Mode: Set, Value: "T"}})
	if opID == "" {
		t.Fatal("Start returned an empty operation ID")
	}
	if _, ok := bs.Result(opID); !ok {
		t.Fatal("result was not registered before Start returned")
	}
	// Cancel must be callable immediately, even if the worker has not run yet.
	bs.Cancel(opID)
}

func TestBatchStartPublishesTerminalResultForInvalidPatch(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "song.flac")
	if err := os.WriteFile(p, []byte("AUDIO"), 0o644); err != nil {
		t.Fatal(err)
	}
	bs := NewBatchService(&Service{Bridge: &fakeTx{}, Snapshots: NewSnapshotStore(filepath.Join(dir, "snaps"))})
	opID := bs.Start(context.Background(), []string{p}, Patch{Title: FieldPatch[string]{Mode: Set, Value: ""}})
	if opID == "" {
		t.Fatal("Start returned an empty operation ID")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		result, ok := bs.Result(opID)
		if ok && result.Failed == 1 && len(result.Items) == 1 {
			if result.Items[0].Code != "empty_set" {
				t.Fatalf("unexpected item: %+v", result.Items[0])
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("terminal result was not published after an early validation failure")
}

func TestBatchPreviewAcceptsDescriptorOnlyPictures(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "song.flac")
	if err := os.WriteFile(p, []byte("AUDIO"), 0o644); err != nil {
		t.Fatal(err)
	}
	bs := NewBatchService(&Service{Bridge: &fakeTx{}, Snapshots: NewSnapshotStore(filepath.Join(dir, "snaps"))})
	patch := Patch{Pictures: FieldPatch[[]Picture]{Mode: Set, Value: []Picture{{
		ID: "apic-0", Type: 3, MIME: "image/jpeg",
	}}}}
	preview, err := bs.Preview(context.Background(), []string{p}, patch)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Eligible != 1 || preview.Blocked != 0 {
		t.Fatalf("descriptor-only preview unexpectedly blocked: %+v", preview)
	}
}
