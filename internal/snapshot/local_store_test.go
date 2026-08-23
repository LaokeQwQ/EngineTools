package snapshot

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"EngineTools/internal/apperror"
	_ "github.com/mattn/go-sqlite3"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	return NewStore(dir), dir
}

func newRef(t *testing.T, dir string) LibraryRef {
	t.Helper()
	dbPath := filepath.Join(dir, "m.db")
	db := openWritable(t, dbPath)
	if _, err := db.Exec("CREATE TABLE t (x TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO t VALUES ('v1')"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	return LibraryRef{ID: "default", DatabasePath: dbPath}
}

func TestCreateRegistersMetadataAndListSortsNewestFirst(t *testing.T) {
	store, _ := newTestStore(t)
	ref := newRef(t, t.TempDir())

	clock := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { clock = clock.Add(time.Minute); return clock }

	first, err := store.Create(context.Background(), ref, CreateOptions{Note: "first"})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := store.Create(context.Background(), ref, CreateOptions{Note: "second"})
	if err != nil {
		t.Fatalf("second create: %v", err)
	}

	backups := store.List()
	if len(backups) != 2 {
		t.Fatalf("listed %d backups", len(backups))
	}
	if backups[0].ID != second.ID || backups[1].ID != first.ID {
		t.Fatalf("sort order wrong: %s before %s", backups[0].ID, backups[1].ID)
	}
	if backups[1].Note != "first" || backups[0].Note != "second" {
		t.Fatalf("notes lost: %+v", backups)
	}
	if len(backups[0].DatabaseSHA256) != 64 || backups[0].Size == 0 {
		t.Fatalf("metadata incomplete: %+v", backups[0])
	}
}

func TestRestoreRejectsUnknownIDAndTraversal(t *testing.T) {
	store, storeDir := newTestStore(t)
	work := t.TempDir()
	ref := newRef(t, work)

	// sentinel outside the backup store must never be touched
	outside := filepath.Join(work, "outside.db")
	if err := os.WriteFile(outside, []byte("sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"../outside.db", "..\\\\outside.db", "not-a-uuid", ""} {
		err := store.Restore(context.Background(), id, ref)
		if apperror.CodeOf(err) != "backup_not_found" {
			t.Fatalf("Restore(%q) code = %v, want backup_not_found", id, err)
		}
	}
	if got, _ := os.ReadFile(outside); string(got) != "sentinel" {
		t.Fatalf("outside file was modified: %q", got)
	}
	_ = storeDir
}

func TestRestoreRecoversTargetFromBackup(t *testing.T) {
	store, _ := newTestStore(t)
	ref := newRef(t, t.TempDir())

	backup, err := store.Create(context.Background(), ref, CreateOptions{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// corrupt the live database afterwards
	db, err := sql.Open("sqlite3", "file:"+ref.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DROP TABLE t"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	if err := store.Restore(context.Background(), backup.ID, ref); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if err := Validate(ref.DatabasePath); err != nil {
		t.Fatalf("restored target invalid: %v", err)
	}
	db, err = sql.Open("sqlite3", "file:"+ref.DatabasePath+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var x string
	if err := db.QueryRow("SELECT x FROM t LIMIT 1").Scan(&x); err != nil || x != "v1" {
		t.Fatalf("restored row = (%q, %v)", x, err)
	}
}

func TestRestoreFailsClosedOnCorruptStoredBackup(t *testing.T) {
	store, storeDir := newTestStore(t)
	ref := newRef(t, t.TempDir())

	backup, err := store.Create(context.Background(), ref, CreateOptions{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before, _ := os.ReadFile(ref.DatabasePath)

	stored := filepath.Join(storeDir, backup.ID+".db")
	if err := os.WriteFile(stored, []byte("garbage overwriting the backup"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := store.Restore(context.Background(), backup.ID, ref); apperror.CodeOf(err) != "backup_invalid" {
		t.Fatalf("code = %v, want backup_invalid", err)
	}
	after, _ := os.ReadFile(ref.DatabasePath)
	if string(before) != string(after) {
		t.Fatal("target was mutated even though the stored backup was corrupt")
	}
}
