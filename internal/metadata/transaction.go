package metadata

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Service performs one copy-on-write metadata transaction per call:
//
//	identity check -> full-file snapshot -> sibling temp copy ->
//	bridge apply on the temp -> re-read + payload verification -> fsync
//	-> atomic replace -> final re-read
//
// Every failure before replace leaves the original untouched; a failure
// after replace rolls the original back from the snapshot.
type Service struct {
	Bridge        Bridge
	Snapshots     *SnapshotStore
	BridgeVersion string

	// Test-only seams: invoked at named points; a non-nil error aborts
	// the transaction exactly there.
	HookAfterSnapshot func() error
	HookBeforeReplace func() error
}

func NewService(b Bridge, store *SnapshotStore) *Service {
	return &Service{Bridge: b, Snapshots: store}
}

// fileIdentity is a portable identity fingerprint: size plus nanosecond
// modification time. Any external write changes it.
type fileIdentity struct {
	Size      int64
	MtimeNano int64
}

func identityOf(path string) (fileIdentity, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return fileIdentity{}, E("file_unreadable", err)
	}
	return fileIdentity{Size: fi.Size(), MtimeNano: fi.ModTime().UnixNano()}, nil
}

// Apply executes one metadata patch as a single recoverable transaction.
func (s *Service) Apply(ctx context.Context, path string, patch Patch) (WriteResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ensureWritableTarget(path); err != nil {
		return WriteResult{}, err
	}
	idBefore, err := identityOf(path)
	if err != nil {
		return WriteResult{}, err
	}

	// Read current document for format/tag version and pre-hash.
	docBefore, err := s.Bridge.Read(path)
	if err != nil {
		return WriteResult{}, err
	}
	payloadBefore, err := s.Bridge.AudioPayloadSHA256(path)
	if err != nil {
		return WriteResult{}, err
	}
	if err := patch.Validate(); err != nil {
		return WriteResult{}, err
	}

	// Full-file recovery snapshot.
	snap, err := s.Snapshots.Save(path, Snapshot{
		FileID:        fmt.Sprintf("%x-%x", idBefore.Size, idBefore.MtimeNano),
		PayloadSHA256: fmt.Sprintf("%x", payloadBefore),
		Format:        docBefore.Format,
		TagVersion:    docBefore.TagVersion,
	})
	if err != nil {
		return WriteResult{}, err
	}
	if s.HookAfterSnapshot != nil {
		if err := s.HookAfterSnapshot(); err != nil {
			return WriteResult{}, E("injected_failure", err)
		}
	}
	snapBin := s.Snapshots.binPathFor(snap)

	// Sibling temp copy of the ORIGINAL; the bridge mutates only this.
	tmp := path + ".metatmp-" + randHex(6)
	if err := copyFileSync(path, tmp); err != nil {
		return WriteResult{}, E("temp_write_failed", err)
	}
	defer os.Remove(tmp)

	if err := ctx.Err(); err != nil {
		return WriteResult{}, E("cancelled", err)
	}
	if _, err := s.Bridge.Apply(tmp, patch, WriteOptions{OperationID: snap.OperationID}); err != nil {
		return WriteResult{}, err
	}

	// Re-read the temp: requested values present and audio untouched.
	docAfter, err := s.Bridge.Read(tmp)
	if err != nil {
		return WriteResult{}, err
	}
	if err := assertPatchApplied(patch, docBefore, docAfter); err != nil {
		return WriteResult{}, err
	}
	payloadAfter, err := s.Bridge.AudioPayloadSHA256(tmp)
	if err != nil {
		return WriteResult{}, err
	}
	if payloadAfter != payloadBefore {
		return WriteResult{}, E("payload_changed", errors.New("audio payload hash moved across a metadata write"))
	}
	if err := syncFile(tmp); err != nil {
		return WriteResult{}, E("io_failed", err)
	}
	_ = idBefore

	// Atomic replace of the live file.
	if s.HookBeforeReplace != nil {
		if err := s.HookBeforeReplace(); err != nil {
			return WriteResult{}, E("injected_failure", err)
		}
	}
	if err := replaceFile(tmp, path); err != nil {
		return WriteResult{}, E("replace_failed", err)
	}

	// Final re-read on the replaced path.
	finalDoc, err := s.Bridge.Read(path)
	if err != nil {
		return WriteResult{}, rollbackFromSnapshot(snapBin, path, idBefore)
	}
	if err := assertPatchApplied(patch, docBefore, finalDoc); err != nil {
		if rbErr := rollbackFromSnapshot(snapBin, path, idBefore); rbErr != nil {
			return WriteResult{}, rbErr
		}
		return WriteResult{}, err
	}
	return WriteResult{OK: true, Changed: changedFields(patch)}, nil
}

// rollbackFromSnapshot restores the exact bytes captured in snapBin.
func rollbackFromSnapshot(snapBin, targetPath string, want fileIdentity) error {
	tmp := targetPath + ".rollback-tmp"
	if err := copyFileSync(snapBin, tmp); err != nil {
		return E("restore_rollback_failed", err)
	}
	if err := syncFile(tmp); err != nil {
		os.Remove(tmp)
		return E("restore_rollback_failed", err)
	}
	if err := replaceFile(tmp, targetPath); err != nil {
		os.Remove(tmp)
		return E("restore_rollback_failed", err)
	}
	got, err := identityOf(targetPath)
	if err != nil || got != want {
		return E("restore_rollback_failed", errors.New("rolled-back identity mismatch"))
	}
	return nil
}

// ensureWritableTarget fails fast when the target cannot be written.
func ensureWritableTarget(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return E("target_not_writable", err)
	}
	return f.Close()
}

// assertPatchApplied verifies every Set/Clear intent is reflected in the
// resulting document. Unchanged fields are ignored.
func assertPatchApplied(patch Patch, before, after Document) error {
	checkString := func(mode PatchMode, want, got, name string) error {
		switch mode {
		case Set:
			if got != want {
				return Ef("verify_failed", "%s not applied: %q", name, got)
			}
		case Clear:
			if got != "" {
				return Ef("verify_failed", "%s not cleared", name)
			}
		}
		return nil
	}
	if err := checkString(patch.Title.Mode, patch.Title.Value, after.Title, "title"); err != nil {
		return err
	}
	if err := checkString(patch.Album.Mode, patch.Album.Value, after.Album, "album"); err != nil {
		return err
	}
	if err := checkString(patch.Key.Mode, patch.Key.Value, after.Key, "key"); err != nil {
		return err
	}
	if patch.Artists.Mode == Set {
		if len(after.Artists) != len(patch.Artists.Value) {
			return Ef("verify_failed", "artists count mismatch")
		}
		for i := range patch.Artists.Value {
			if after.Artists[i] != patch.Artists.Value[i] {
				return Ef("verify_failed", "artist %d mismatch", i)
			}
		}
	}
	if patch.Track.Mode == Set {
		if after.Track.Value != patch.Track.Value.Value {
			return Ef("verify_failed", "track number mismatch")
		}
	}
	if patch.BPM.Mode == Set && after.BPM != patch.BPM.Value {
		return Ef("verify_failed", "bpm mismatch")
	}
	if patch.Rating.Mode == Set && after.Rating != patch.Rating.Value {
		return Ef("verify_failed", "rating mismatch")
	}
	if patch.ISRC.Mode == Set && after.ISRC != patch.ISRC.Value {
		return Ef("verify_failed", "isrc mismatch")
	}
	_ = before
	return nil
}

func changedFields(p Patch) int {
	n := 0
	count := func(m PatchMode) {
		if m == Set || m == Clear {
			n++
		}
	}
	count(p.Title.Mode)
	count(p.Album.Mode)
	count(p.Date.Mode)
	count(p.Comment.Mode)
	count(p.Key.Mode)
	count(p.Lyrics.Mode)
	count(p.ISRC.Mode)
	count(p.BPM.Mode)
	count(p.Rating.Mode)
	count(p.Track.Mode)
	count(p.Disc.Mode)
	for _, f := range []FieldPatch[[]string]{p.Artists, p.AlbumArtists, p.Genres, p.Composers} {
		count(f.Mode)
	}
	return n
}

var _ = filepath.Join
var _ = fmt.Sprintf
var _ = os.Getpid
