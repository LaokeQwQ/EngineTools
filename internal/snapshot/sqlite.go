// Package snapshot produces consistent SQLite database backups through the
// SQLite Online Backup API instead of raw file copies, validates them, and
// keeps registered local backups recoverable.
package snapshot

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"EngineTools/internal/apperror"
)

// Info describes one completed snapshot.
type Info struct {
	Path      string
	SHA256    string
	Integrity string
	Size      int64
}

// SQLite snapshots a live database through (*SQLiteConn).Backup.
type SQLite struct{}

// Create backs up the SQLite database at src into dst using the online
// backup API, which tolerates an open WAL writer on src. The write lands
// on a sibling temp file that is synced before an atomic rename.
func (SQLite) Create(ctx context.Context, src, dst string) (Info, error) {
	if _, err := os.Stat(src); err != nil {
		return Info{}, apperror.E("source_unavailable", "errors.sourceUnavailable", false,
			fmt.Errorf("stat %s: %w", src, err))
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return Info{}, apperror.E("backup_write_failed", "errors.backupWriteFailed", true, err)
	}

	suffix := randHex(6)
	temp := dst + ".snapshot-" + suffix
	defer os.Remove(temp) // no-op after successful rename

	if err := backupInto(ctx, src, temp); err != nil {
		return Info{}, err
	}
	if err := syncFile(temp); err != nil {
		return Info{}, apperror.E("backup_write_failed", "errors.backupWriteFailed", true, err)
	}
	if err := os.Rename(temp, dst); err != nil {
		return Info{}, apperror.E("backup_write_failed", "errors.backupWriteFailed", true, err)
	}

	if err := Validate(dst); err != nil {
		return Info{}, err
	}
	sum, size, err := hashFile(dst)
	if err != nil {
		return Info{}, apperror.E("backup_write_failed", "errors.backupWriteFailed", true, err)
	}
	integrity, err := integrityCheck(dst)
	if err != nil {
		return Info{}, err
	}
	return Info{Path: dst, SHA256: sum, Integrity: integrity, Size: size}, nil
}

// Validate opens a finished SQLite file and asserts its header magic,
// integrity_check verdict, and an empty foreign_key_check.
func Validate(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return apperror.E("backup_invalid", "errors.backupInvalid", true, err)
	}
	header := make([]byte, 16)
	_, rerr := io.ReadFull(f, header)
	f.Close()
	if rerr != nil {
		return apperror.E("backup_invalid", "errors.backupInvalid", true, rerr)
	}
	if string(header) != "SQLite format 3\x00" {
		return apperror.E("backup_invalid", "errors.backupInvalid", false,
			fmt.Errorf("%s is not a SQLite 3 database", path))
	}

	db, err := sql.Open("sqlite3", "file:"+path+"?mode=ro")
	if err != nil {
		return apperror.E("backup_invalid", "errors.backupInvalid", true, err)
	}
	defer db.Close()

	var verdict string
	if err := db.QueryRowContext(context.Background(), "PRAGMA integrity_check").Scan(&verdict); err != nil {
		return apperror.E("backup_invalid", "errors.backupInvalid", true, err)
	}
	if verdict != "ok" {
		return apperror.E("backup_invalid", "errors.backupInvalid", false,
			fmt.Errorf("integrity_check reported %q", verdict))
	}
	rows, err := db.QueryContext(context.Background(), "PRAGMA foreign_key_check")
	if err != nil {
		return apperror.E("backup_invalid", "errors.backupInvalid", true, err)
	}
	violations := 0
	for rows.Next() {
		violations++
	}
	rows.Close()
	if violations != 0 {
		return apperror.E("backup_invalid", "errors.backupInvalid", false,
			fmt.Errorf("foreign_key_check reported %d violations", violations))
	}
	return nil
}

// backupInto streams src into dstPath through the online backup API using
// dedicated connections so neither side disturbs pooled handles.
func backupInto(ctx context.Context, src, dstPath string) error {
	srcDB, err := sql.Open("sqlite3", "file:"+src+"?mode=ro")
	if err != nil {
		return apperror.E("backup_failed", "errors.backupFailed", true, err)
	}
	defer srcDB.Close()
	dstDB, err := sql.Open("sqlite3", "file:"+dstPath)
	if err != nil {
		return apperror.E("backup_failed", "errors.backupFailed", true, err)
	}
	defer dstDB.Close()

	srcConn, err := srcDB.Conn(ctx)
	if err != nil {
		return apperror.E("backup_failed", "errors.backupFailed", true, err)
	}
	defer srcConn.Close()
	dstConn, err := dstDB.Conn(ctx)
	if err != nil {
		return apperror.E("backup_failed", "errors.backupFailed", true, err)
	}
	defer dstConn.Close()

	if err := runBackupAPI(ctx, srcConn, dstConn); err != nil {
		if ctx.Err() != nil {
			return apperror.E("backup_cancelled", "errors.backupCancelled", false, ctx.Err())
		}
		return apperror.E("backup_failed", "errors.backupFailed", true, err)
	}
	return nil
}

func integrityCheck(path string) (string, error) {
	db, err := sql.Open("sqlite3", "file:"+path+"?mode=ro")
	if err != nil {
		return "", apperror.E("backup_invalid", "errors.backupInvalid", true, err)
	}
	defer db.Close()
	var verdict string
	if err := db.QueryRowContext(context.Background(), "PRAGMA integrity_check").Scan(&verdict); err != nil {
		return "", apperror.E("backup_invalid", "errors.backupInvalid", true, err)
	}
	return verdict, nil
}

func syncFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "entropy"
	}
	return hex.EncodeToString(b)
}
