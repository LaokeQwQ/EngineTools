package snapshot

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// openWritable opens a plain writable connection used to build fixtures.
func openWritable(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteSnapshotIncludesWALCommittedRow(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "m.db")
	dst := filepath.Join(dir, "m.backup.db")

	// WAL mode with a second long-lived connection is the exact scenario
	// raw file copies corrupt.
	db := openWritable(t, src)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("enable WAL: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE track_count (n INTEGER)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec("INSERT INTO track_count (n) VALUES (42)"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// keep a second open connection so the WAL and shared memory stay hot
	second := openWritable(t, src)
	if _, err := second.Exec("SELECT count(*) FROM track_count"); err != nil {
		t.Fatalf("warm second connection: %v", err)
	}

	got, err := (SQLite{}).Create(context.Background(), src, dst)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if got.Integrity != "ok" {
		t.Fatalf("integrity = %q", got.Integrity)
	}
	if got.Size == 0 || len(got.SHA256) != 64 {
		t.Fatalf("bad info: %#v", got)
	}

	// the committed row must be readable from the snapshot
	backup := openWritable(t, dst)
	var n int
	if err := backup.QueryRow("SELECT n FROM track_count").Scan(&n); err != nil {
		t.Fatalf("query snapshot: %v", err)
	}
	if n != 42 {
		t.Fatalf("row value = %d", n)
	}
}

func TestCreateHonorsCancelledContext(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "m.db")
	setupTinyDB(t, src)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (SQLite{}).Create(ctx, src, filepath.Join(dir, "out.db"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestValidateRejectsNonDatabaseFile(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake.db")
	if err := os.WriteFile(fake, []byte("definitely not sqlite"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(fake); err == nil {
		t.Fatal("Validate accepted a text file")
	}
}

func setupTinyDB(t *testing.T, path string) {
	t.Helper()
	db := openWritable(t, path)
	if _, err := db.Exec("CREATE TABLE t (x TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO t VALUES ('v')"); err != nil {
		t.Fatal(err)
	}
}
