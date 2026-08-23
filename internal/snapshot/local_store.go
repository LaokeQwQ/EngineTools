package snapshot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"EngineTools/internal/apperror"
)

// LibraryRef identifies one library and where its live database lives.
type LibraryRef struct {
	ID           string
	DatabasePath string
}

// CreateOptions carries user-visible metadata for a new backup.
type CreateOptions struct {
	Note string
}

// Backup is one registered local backup.
type Backup struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	Note           string    `json:"note,omitempty"`
	DatabaseSHA256 string    `json:"databaseSha256"`
	Size           int64     `json:"size"`
}

type indexFile struct {
	Version int      `json:"version"`
	Backups []Backup `json:"backups"`
}

var backupIDPattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// Store keeps registered backups under one directory with an atomically
// written index. Restore resolves only IDs already present in the index,
// so path traversal can never reach an arbitrary file.
type Store struct {
	sqlite SQLite
	dir    string
	now    func() time.Time
	mu     sync.Mutex // serializes index.json read-modify-write
}

func NewStore(dir string) *Store {
	_ = os.MkdirAll(dir, 0o755)
	return &Store{dir: dir, now: func() time.Time { return time.Now().UTC() }}
}

// Create snapshots ref.DatabasePath into the store and registers it.
func (s *Store) Create(ctx context.Context, ref LibraryRef, opts CreateOptions) (Backup, error) {
	if ref.DatabasePath == "" {
		return Backup{}, apperror.E("library_unavailable", "errors.libraryUnavailable", false,
			fmt.Errorf("empty database path for library %q", ref.ID))
	}
	id := newUUIDv4()
	dest := filepath.Join(s.dir, id+".db")
	info, err := s.sqlite.Create(ctx, ref.DatabasePath, dest)
	if err != nil {
		return Backup{}, err
	}
	backup := Backup{
		ID:             id,
		CreatedAt:      s.now(),
		Note:           opts.Note,
		DatabaseSHA256: info.SHA256,
		Size:           info.Size,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.readIndex()
	index.Backups = append(index.Backups, backup)
	if err := s.writeIndex(index); err != nil {
		os.Remove(dest)
		return Backup{}, apperror.E("index_write_failed", "errors.indexWriteFailed", true, err)
	}
	return backup, nil
}

// Restore replaces ref.DatabasePath with the stored backup identified by
// backupID. The current database is snapshotted as a rollback first; if
// anything fails after the swap it is put back.
func (s *Store) Restore(ctx context.Context, backupID string, ref LibraryRef) error {
	if !backupIDPattern.MatchString(backupID) {
		return apperror.E("backup_not_found", "errors.backupNotFound", false,
			fmt.Errorf("malformed backup id %q", backupID))
	}
	s.mu.Lock()
	index := s.readIndex()
	s.mu.Unlock()
	entry, ok := findBackup(index, backupID)
	if !ok {
		return apperror.E("backup_not_found", "errors.backupNotFound", false,
			fmt.Errorf("backup %q is not registered", backupID))
	}
	storedPath := filepath.Join(s.dir, entry.ID+".db")
	if _, err := os.Stat(storedPath); err != nil {
		return apperror.E("backup_not_found", "errors.backupNotFound", false, err)
	}
	if err := Validate(storedPath); err != nil {
		return err
	}
	if ref.DatabasePath == "" {
		return apperror.E("library_unavailable", "errors.libraryUnavailable", false,
			fmt.Errorf("empty database path for library %q", ref.ID))
	}

	// Rollback snapshot of the CURRENT target, kept next to the target so
	// the final rename stays on one filesystem.
	rollback := ref.DatabasePath + ".rollback-" + randHex(6)
	defer os.Remove(rollback)
	if _, err := os.Stat(ref.DatabasePath); err == nil {
		if _, err := s.sqlite.Create(ctx, ref.DatabasePath, rollback); err != nil {
			return err
		}
	}

	// Stage the stored backup beside the target, then swap atomically.
	staged := ref.DatabasePath + ".stage-" + randHex(6)
	defer os.Remove(staged)
	if _, err := s.sqlite.Create(ctx, storedPath, staged); err != nil {
		return err
	}
	if err := os.Rename(staged, ref.DatabasePath); err != nil {
		return apperror.E("restore_failed", "errors.restoreFailed", true, err)
	}
	if err := Validate(ref.DatabasePath); err != nil {
		if rbErr := os.Rename(rollback, ref.DatabasePath); rbErr != nil {
			return apperror.E("restore_rollback_failed", "errors.restoreRollbackFailed", true,
				fmt.Errorf("restore failed (%v) and rollback failed (%v)", err, rbErr))
		}
		return apperror.E("restore_failed", "errors.restoreFailed", true, err)
	}
	return nil
}

// List returns the registered backups newest first.
func (s *Store) List() []Backup {
	s.mu.Lock()
	index := s.readIndex()
	s.mu.Unlock()
	sorted := append([]Backup(nil), index.Backups...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].CreatedAt.After(sorted[j].CreatedAt) })
	return sorted
}

func findBackup(index indexFile, id string) (Backup, bool) {
	for _, b := range index.Backups {
		if b.ID == id {
			return b, true
		}
	}
	return Backup{}, false
}

func (s *Store) indexPath() string { return filepath.Join(s.dir, "index.json") }

func (s *Store) readIndex() indexFile {
	index := indexFile{Version: 1}
	data, err := os.ReadFile(s.indexPath())
	if err != nil {
		return index
	}
	_ = json.Unmarshal(data, &index)
	if index.Version != 1 {
		index = indexFile{Version: 1}
	}
	return index
}

func (s *Store) writeIndex(index indexFile) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	temp := s.indexPath() + ".tmp-" + randHex(6)
	if err := os.WriteFile(temp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(temp, s.indexPath()); err != nil {
		os.Remove(temp)
		return err
	}
	return nil
}

func newUUIDv4() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
