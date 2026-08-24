package metadata

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Snapshot is a full-file recovery point taken before one metadata write.
// It preserves every unknown chunk and padding because it is a complete
// image of the original file, not just the tag region.
type Snapshot struct {
	ID            string    `json:"id"`
	OriginalPath  string    `json:"originalPath"`
	FileID        string    `json:"fileId"`
	FullSHA256    string    `json:"fullSha256"`
	PayloadSHA256 string    `json:"payloadSha256,omitempty"`
	Format        Format    `json:"format,omitempty"`
	TagVersion    string    `json:"tagVersion,omitempty"`
	OperationID   string    `json:"operationId,omitempty"`
	BridgeVersion string    `json:"bridgeVersion,omitempty"`
	State         string    `json:"state"` // complete | partial-failed
	CreatedAt     time.Time `json:"createdAt"`
}

type indexFile struct {
	Version   int        `json:"version"`
	Snapshots []Snapshot `json:"snapshots"`
}

// SnapshotStore persists full-file recovery images plus per-snapshot JSON
// manifests. Retention keeps the newest completed operations inside the
// window and never auto-deletes unresolved failures or the newest undo.
type SnapshotStore struct {
	dir        string
	mu         sync.Mutex
	now        func() time.Time
	KeepCount  int
	KeepWindow time.Duration
}

func NewSnapshotStore(dir string) *SnapshotStore {
	_ = os.MkdirAll(dir, 0o755)
	return &SnapshotStore{
		dir:        dir,
		now:        func() time.Time { return time.Now().UTC() },
		KeepCount:  20,
		KeepWindow: 30 * 24 * time.Hour,
	}
}

func (s *SnapshotStore) dirFor(originalPath string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(originalPath)))
	return filepath.Join(s.dir, hex.EncodeToString(sum[:8]))
}

func (s *SnapshotStore) binPathFor(snap Snapshot) string {
	return filepath.Join(s.dirFor(snap.OriginalPath), snap.ID+".bin")
}

func (s *SnapshotStore) manifestPathFor(snap Snapshot) string {
	return filepath.Join(s.dirFor(snap.OriginalPath), snap.ID+".json")
}

// Save captures a full-file recovery image. A same-volume hard link is
// preferred (cheap, and the original inode keeps the pre-write bytes);
// any filesystem refusal falls back to a verified stream copy.
func (s *SnapshotStore) Save(originalPath string, meta Snapshot) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := randHex(16)
	dir := s.dirFor(originalPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Snapshot{}, E("snapshot_io_failed", err)
	}
	bin := filepath.Join(dir, id+".bin")

	// Always stream-copy. Hard links alias the original inode, so an
	// in-place rewrite of the live file would silently change the
	// recovery image too.
	if cerr := copyFileSync(originalPath, bin); cerr != nil {
		return Snapshot{}, E("snapshot_io_failed", cerr)
	}

	sum, err := hashFileHex(bin)
	if err != nil {
		os.Remove(bin)
		return Snapshot{}, E("snapshot_io_failed", err)
	}

	snap := Snapshot{
		ID:            id,
		OriginalPath:  originalPath,
		FileID:        meta.FileID,
		FullSHA256:    sum,
		PayloadSHA256: meta.PayloadSHA256,
		Format:        meta.Format,
		TagVersion:    meta.TagVersion,
		OperationID:   meta.OperationID,
		BridgeVersion: meta.BridgeVersion,
		State:         "complete",
		CreatedAt:     s.now(),
	}

	if err := writeJSONAtomic(s.manifestPathFor(snap), snap); err != nil {
		os.Remove(bin)
		return Snapshot{}, E("snapshot_io_failed", err)
	}
	return snap, nil
}

// Get returns one stored snapshot by ID.
func (s *SnapshotStore) Get(id string) (Snapshot, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, snap := range s.listLocked() {
		if snap.ID == id {
			return snap, true, nil
		}
	}
	return Snapshot{}, false, nil
}

// List returns every snapshot, newest first.
func (s *SnapshotStore) List() []Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listLocked()
}

func (s *SnapshotStore) listLocked() []Snapshot {
	var out []Snapshot
	_ = filepath.WalkDir(s.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".json") || strings.Contains(name, ".tmp-") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		var snap Snapshot
		if json.Unmarshal(data, &snap) == nil && snap.ID != "" {
			out = append(out, snap)
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// RestoreBin materializes the snapshot binary into destPath through a
// sibling temp file and atomic rename after re-verifying its hash.
func (s *SnapshotStore) RestoreBin(id, destPath string) error {
	s.mu.Lock()
	all := s.listLocked()
	s.mu.Unlock()
	var snap *Snapshot
	for i := range all {
		if all[i].ID == id {
			snap = &all[i]
			break
		}
	}
	if snap == nil {
		return Ef("snapshot_not_found", "snapshot %q is not registered", id)
	}
	bin := s.binPathFor(*snap)
	sum, herr := hashFileHex(bin)
	if herr != nil || sum != snap.FullSHA256 {
		return Ef("snapshot_corrupt", "snapshot %q failed integrity check", id)
	}
	tmp := destPath + ".restore-tmp"
	if err := copyFileSync(bin, tmp); err != nil {
		os.Remove(tmp)
		return E("snapshot_io_failed", err)
	}
	if err := syncFile(tmp); err != nil {
		os.Remove(tmp)
		return E("snapshot_io_failed", err)
	}
	if err := os.Rename(tmp, destPath); err != nil {
		os.Remove(tmp)
		return E("snapshot_io_failed", err)
	}
	return nil
}

// Prune enforces retention. Never auto-deletes partial failures or the
// single newest usable undo point.
func (s *SnapshotStore) Prune() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	all := s.listLocked()
	removed := 0
	kept := 0
	for i, snap := range all {
		age := now.Sub(snap.CreatedAt)
		protected := snap.State != "complete" || i == 0
		if protected || age <= s.KeepWindow {
			kept++
			continue
		}
		if kept >= s.KeepCount {
			os.Remove(s.binPathFor(snap))
			os.Remove(s.manifestPathFor(snap))
			removed++
			continue
		}
		kept++
	}
	return removed
}

func (s *SnapshotStore) writeJSONAtomic(path string, v Snapshot) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp-" + randHex(6)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func copyFileSync(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err = out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func syncFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	err = f.Sync()
	cerr := f.Close()
	if err != nil {
		return err
	}
	return cerr
}

func hashFileHex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "entropy"
	}
	return hex.EncodeToString(b)
}

func writeJSONAtomic(path string, v Snapshot) error {
	data, err := json.MarshalIndent(v, "", "")
	if err != nil {
		return err
	}
	tmp := path + ".tmp-" + randHex(6)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
