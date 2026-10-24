package metadata

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

	// pathLocks serializes writes to the same file. Different files remain
	// fully concurrent for batch operations.
	pathLocks sync.Map // map[string]*sync.Mutex
}

func NewService(b Bridge, store *SnapshotStore) *Service {
	return &Service{Bridge: b, Snapshots: store}
}

// Capabilities probes one path and normalizes a bridge response against the
// canonical format table. A bridge may omit optional maps; in that case the
// table supplies conservative defaults. No write is attempted by this call.
func (s *Service) Capabilities(path string) (Capabilities, error) {
	if s == nil || s.Bridge == nil {
		return Capabilities{}, E("bridge_unavailable", errors.New("metadata bridge is unavailable"))
	}
	if path == "" {
		return Capabilities{}, E("file_unreadable", errors.New("metadata path is empty"))
	}
	caps, err := s.Bridge.Probe(path)
	if err != nil {
		return Capabilities{}, err
	}
	if caps.Format == "" {
		// A few older bridges only return a format from Read. Keep those
		// bridges usable while still rejecting an unknown format.
		doc, rerr := s.Bridge.Read(path)
		if rerr != nil {
			return Capabilities{}, rerr
		}
		caps.Format = doc.Format
	}
	canonical, ok := CapabilitiesFor(caps.Format)
	if !ok {
		return Capabilities{}, Ef("format_invalid", "unsupported format %q", caps.Format)
	}
	wasEmpty := caps.Readable == nil && caps.Writable == nil && len(caps.PictureMIMEs) == 0 && len(caps.TagVersions) == 0 && caps.MaxPictures == 0
	if caps.Readable == nil {
		caps.Readable = canonical.Readable
	}
	if caps.Writable == nil {
		caps.Writable = canonical.Writable
	}
	if len(caps.PictureMIMEs) == 0 && canonical.MaxPictures > 0 {
		caps.PictureMIMEs = canonical.PictureMIMEs
	}
	if len(caps.TagVersions) == 0 {
		caps.TagVersions = canonical.TagVersions
	}
	// Zero is a valid limit for WAV/AIFF. Only use the canonical value when
	// the bridge omitted every other capability detail as well.
	if wasEmpty && canonical.MaxPictures > 0 {
		caps.MaxPictures = canonical.MaxPictures
	}
	return cloneCapabilities(caps), nil
}

// Probe is a short alias useful to Wails/facade callers.
func (s *Service) Probe(path string) (Capabilities, error) { return s.Capabilities(path) }

// fileIdentity is a portable identity fingerprint. Size and modification time
// are cheap early signals; the content hash closes the race where another
// process replaces a file with equal size and timestamp.
type fileIdentity struct {
	Size       int64
	MtimeNano  int64
	FullSHA256 string
}

func identityOf(path string) (fileIdentity, error) {
	for attempt := 0; attempt < 2; attempt++ {
		before, err := os.Stat(path)
		if err != nil {
			return fileIdentity{}, E("file_unreadable", err)
		}
		if before.IsDir() {
			return fileIdentity{}, E("file_unreadable", errors.New("metadata target is a directory"))
		}
		hash, err := hashFileHex(path)
		if err != nil {
			return fileIdentity{}, E("file_unreadable", err)
		}
		after, err := os.Stat(path)
		if err != nil {
			return fileIdentity{}, E("file_unreadable", err)
		}
		if before.Size() == after.Size() && before.ModTime() == after.ModTime() {
			return fileIdentity{Size: after.Size(), MtimeNano: after.ModTime().UnixNano(), FullSHA256: hash}, nil
		}
	}
	return fileIdentity{}, E("external_change", errors.New("target changed while identifying the file"))
}

func sameIdentity(path string, want fileIdentity) bool {
	got, err := identityOf(path)
	return err == nil && got == want
}

// Snapshot captures one recoverable full-file image without writing metadata.
// Anti-piracy operations use it to verify every included file before the first
// destructive write, so a later failure cannot leave an earlier file changed
// without a recovery point.
func (s *Service) Snapshot(path string) (Snapshot, error) {
	if s == nil || s.Bridge == nil {
		return Snapshot{}, E("bridge_unavailable", errors.New("metadata bridge is unavailable"))
	}
	if path == "" {
		return Snapshot{}, E("file_unreadable", errors.New("metadata path is empty"))
	}
	unlock := s.lockPath(path)
	defer unlock()
	if err := ensureWritableTarget(path); err != nil {
		return Snapshot{}, err
	}
	idBefore, err := identityOf(path)
	if err != nil {
		return Snapshot{}, err
	}
	caps, err := s.Capabilities(path)
	if err != nil {
		return Snapshot{}, err
	}
	doc, err := s.Bridge.Read(path)
	if err != nil {
		return Snapshot{}, err
	}
	if doc.Format != "" && caps.Format != "" && doc.Format != caps.Format {
		return Snapshot{}, Ef("format_invalid", "probe/read format mismatch")
	}
	payload, err := s.Bridge.AudioPayloadSHA256(path)
	if err != nil {
		return Snapshot{}, err
	}
	if s.Snapshots == nil {
		return Snapshot{}, E("snapshot_store_missing", errors.New("no snapshot store"))
	}
	return s.Snapshots.Save(path, Snapshot{
		FileID:        fmtFileID(idBefore),
		PayloadSHA256: fmt.Sprintf("%x", payload),
		Format:        doc.Format,
		TagVersion:    doc.TagVersion,
		OperationID:   randHex(16),
		BridgeVersion: s.BridgeVersion,
	})
}

// Apply executes one metadata patch as a single recoverable transaction.
func (s *Service) Apply(ctx context.Context, path string, patch Patch) (WriteResult, error) {
	if s == nil || s.Bridge == nil {
		return WriteResult{}, E("bridge_unavailable", errors.New("metadata bridge is unavailable"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return WriteResult{}, E("cancelled", err)
	}
	if path == "" {
		return WriteResult{}, E("file_unreadable", errors.New("metadata path is empty"))
	}
	unlock := s.lockPath(path)
	defer unlock()
	if err := ensureWritableTarget(path); err != nil {
		return WriteResult{}, err
	}
	idBefore, err := identityOf(path)
	if err != nil {
		return WriteResult{}, err
	}

	// Probe capabilities before taking a snapshot. This is intentionally before
	// any temp file or manifest mutation so unsupported fields fail cleanly.
	caps, err := s.Capabilities(path)
	if err != nil {
		return WriteResult{}, err
	}
	// Read current document for format/tag version and pre-hash.
	docBefore, err := s.Bridge.Read(path)
	if err != nil {
		return WriteResult{}, err
	}
	if docBefore.Format != "" && caps.Format != "" && docBefore.Format != caps.Format {
		return WriteResult{}, Ef("format_invalid", "probe/read format mismatch")
	}
	// The WebView receives descriptors plus thumbnails. Rehydrate unchanged
	// picture payloads from the authoritative pre-write document before strict
	// patch validation, so a full-list Set cannot discard non-target artwork.
	patch = hydratePicturePatch(patch, docBefore.Pictures)
	if err := patch.Validate(); err != nil {
		return WriteResult{}, err
	}
	// Keep the capability contract available to callers that use the document
	// directly (notably the Wails editor), while retaining the bridge's native
	// metadata fields untouched.
	docBefore.Capabilities = &caps
	if err := validatePatchCapabilities(caps, patch); err != nil {
		return WriteResult{}, err
	}
	if !patchHasChanges(patch) {
		return WriteResult{OK: true}, nil
	}
	payloadBefore, err := s.Bridge.AudioPayloadSHA256(path)
	if err != nil {
		return WriteResult{}, err
	}

	if s.Snapshots == nil {
		return WriteResult{}, E("snapshot_store_missing", errors.New("no snapshot store"))
	}
	// Full-file recovery snapshot.
	operationID := randHex(16)
	snap, err := s.Snapshots.Save(path, Snapshot{
		FileID:        fmtFileID(idBefore),
		PayloadSHA256: fmt.Sprintf("%x", payloadBefore),
		Format:        docBefore.Format,
		TagVersion:    docBefore.TagVersion,
		OperationID:   operationID,
		BridgeVersion: s.BridgeVersion,
	})
	if err != nil {
		return WriteResult{}, err
	}
	if s.HookAfterSnapshot != nil {
		if err := s.HookAfterSnapshot(); err != nil {
			return WriteResult{}, E("injected_failure", err)
		}
	}
	if !sameIdentity(path, idBefore) {
		return WriteResult{}, E("external_change", errors.New("target changed while preparing metadata write"))
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
	bridgeResult, err := s.Bridge.Apply(tmp, patch, WriteOptions{OperationID: snap.OperationID})
	if err != nil {
		return WriteResult{}, err
	}
	if !bridgeResult.OK {
		return WriteResult{}, E("apply_failed", errors.New("bridge reported an unsuccessful apply"))
	}
	if err := ctx.Err(); err != nil {
		return WriteResult{}, E("cancelled", err)
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
	if !sameIdentity(path, idBefore) {
		return WriteResult{}, E("external_change", errors.New("target changed before atomic replace"))
	}

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
		if rbErr := rollbackFromSnapshot(snapBin, path, snap.FullSHA256); rbErr != nil {
			return WriteResult{}, rbErr
		}
		return WriteResult{}, E("verify_failed", err)
	}
	if err := assertPatchApplied(patch, docBefore, finalDoc); err != nil {
		if rbErr := rollbackFromSnapshot(snapBin, path, snap.FullSHA256); rbErr != nil {
			return WriteResult{}, rbErr
		}
		return WriteResult{}, err
	}
	if err := s.Snapshots.MarkApplied(snap.ID, path); err != nil {
		if rbErr := rollbackFromSnapshot(snapBin, path, snap.FullSHA256); rbErr != nil {
			return WriteResult{}, rbErr
		}
		return WriteResult{}, E("snapshot_io_failed", err)
	}
	return WriteResult{OK: true, Changed: changedFields(patch), SnapshotID: snap.ID, OperationID: snap.OperationID}, nil
}

func hydratePicturePatch(p Patch, current []Picture) Patch {
	if effectiveMode(p.Pictures.Mode) != Set || len(p.Pictures.Value) == 0 || len(current) == 0 {
		return p
	}
	byID := make(map[string]Picture, len(current))
	for _, picture := range current {
		if picture.ID != "" {
			byID[picture.ID] = picture
		}
	}
	pictures := append([]Picture(nil), p.Pictures.Value...)
	changed := false
	for i := range pictures {
		if len(pictures[i].Data) != 0 || pictures[i].ID == "" {
			continue
		}
		source, ok := byID[pictures[i].ID]
		if !ok || len(source.Data) == 0 {
			continue
		}
		pictures[i].Data = append([]byte(nil), source.Data...)
		if pictures[i].MIME == "" {
			pictures[i].MIME = source.MIME
		}
		if pictures[i].Description == "" {
			pictures[i].Description = source.Description
		}
		if pictures[i].Bytes == 0 {
			pictures[i].Bytes = source.Bytes
		}
		if pictures[i].Width == 0 {
			pictures[i].Width = source.Width
		}
		if pictures[i].Height == 0 {
			pictures[i].Height = source.Height
		}
		changed = true
	}
	if changed {
		p.Pictures.Value = pictures
	}
	return p
}

// rollbackFromSnapshot restores the exact bytes captured in snapBin.
func rollbackFromSnapshot(snapBin, targetPath, wantHash string) error {
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
	got, err := hashFileHex(targetPath)
	if err != nil || (wantHash != "" && got != wantHash) {
		return E("restore_rollback_failed", errors.New("rolled-back identity mismatch"))
	}
	return nil
}

// Undo restores the newest complete snapshot for path. An optional snapshot
// ID can be supplied to target a specific recovery point, which keeps the
// method useful to both the single-file editor and batch result UI.
func (s *Service) Undo(ctx context.Context, path string, snapshotID ...string) (WriteResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return WriteResult{}, E("cancelled", err)
	}
	if s == nil || s.Snapshots == nil {
		return WriteResult{}, E("snapshot_store_missing", errors.New("no snapshot store"))
	}
	if path == "" {
		return WriteResult{}, E("file_unreadable", errors.New("metadata path is empty"))
	}
	id := ""
	if len(snapshotID) > 0 {
		id = snapshotID[0]
	}
	if id == "" {
		for _, snap := range s.Snapshots.List() {
			if snap.OriginalPath == path && snap.State == "complete" {
				id = snap.ID
				break
			}
		}
		if id == "" {
			return WriteResult{}, E("snapshot_not_found", errors.New("no undo point is available"))
		}
	} else {
		snap, ok, err := s.Snapshots.Get(id)
		if err != nil {
			return WriteResult{}, err
		}
		if !ok {
			return WriteResult{}, Ef("snapshot_not_found", "snapshot %q is not registered", id)
		}
		if snap.OriginalPath != path {
			return WriteResult{}, E("snapshot_path_mismatch", errors.New("snapshot belongs to another file"))
		}
	}
	return s.UndoSnapshot(ctx, id)
}

// UndoSnapshot restores a specific snapshot whose manifest contains the
// original target path. The snapshot store verifies its integrity and, when
// available, the post-apply target hash before replacing the file.
func (s *Service) UndoSnapshot(ctx context.Context, id string) (WriteResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return WriteResult{}, E("cancelled", err)
	}
	if s == nil || s.Snapshots == nil {
		return WriteResult{}, E("snapshot_store_missing", errors.New("no snapshot store"))
	}
	snap, ok, err := s.Snapshots.Get(id)
	if err != nil {
		return WriteResult{}, err
	}
	if !ok {
		return WriteResult{}, Ef("snapshot_not_found", "snapshot %q is not registered", id)
	}
	unlock := s.lockPath(snap.OriginalPath)
	defer unlock()
	if _, err := s.Snapshots.Restore(ctx, id); err != nil {
		return WriteResult{}, err
	}
	return WriteResult{OK: true, Changed: 1, SnapshotID: snap.ID, OperationID: snap.OperationID}, nil
}

// lockPath serializes transactions for one canonical path. The map is
// intentionally process-local: one desktop process owns each file.
func (s *Service) lockPath(path string) func() {
	key := filepath.Clean(path)
	if absolute, err := filepath.Abs(key); err == nil {
		key = absolute
	}
	value, _ := s.pathLocks.LoadOrStore(key, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

// ensureWritableTarget fails fast when the target cannot be written.
func ensureWritableTarget(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return E("target_not_writable", err)
	}
	return f.Close()
}

// assertPatchApplied verifies every Set/Clear intent and also checks that an
// untouched canonical field did not change as a side effect of the bridge.
// Unknown/container-private data is covered by the payload hash and remains
// the bridge's responsibility.
func assertPatchApplied(patch Patch, before, after Document) error {
	if before.Format != "" && after.Format != "" && before.Format != after.Format {
		return Ef("verify_failed", "format changed during metadata apply")
	}
	checkString := func(name string, mode PatchMode, want, old, got string) error {
		switch effectiveMode(mode) {
		case Set:
			if got != want {
				return Ef("verify_failed", "%s not applied", name)
			}
		case Clear:
			if got != "" {
				return Ef("verify_failed", "%s not cleared", name)
			}
		case Unchanged:
			if got != old {
				return Ef("verify_failed", "%s changed unexpectedly", name)
			}
		}
		return nil
	}
	checkStrings := func(name string, mode PatchMode, want, old, got []string) error {
		switch effectiveMode(mode) {
		case Set:
			if !equalStrings(want, got) {
				return Ef("verify_failed", "%s not applied", name)
			}
		case Clear:
			if len(got) != 0 {
				return Ef("verify_failed", "%s not cleared", name)
			}
		case Unchanged:
			if !equalStrings(old, got) {
				return Ef("verify_failed", "%s changed unexpectedly", name)
			}
		}
		return nil
	}
	checkPair := func(name string, mode PatchMode, want, old, got NumberPair) error {
		switch effectiveMode(mode) {
		case Set:
			if got != want {
				return Ef("verify_failed", "%s not applied", name)
			}
		case Clear:
			if got != (NumberPair{}) {
				return Ef("verify_failed", "%s not cleared", name)
			}
		case Unchanged:
			if got != old {
				return Ef("verify_failed", "%s changed unexpectedly", name)
			}
		}
		return nil
	}
	checkFloat := func(name string, mode PatchMode, want, old, got float64) error {
		switch effectiveMode(mode) {
		case Set:
			if !equalFloat(want, got) {
				return Ef("verify_failed", "%s not applied", name)
			}
		case Clear:
			if got != 0 {
				return Ef("verify_failed", "%s not cleared", name)
			}
		case Unchanged:
			if !equalFloat(old, got) {
				return Ef("verify_failed", "%s changed unexpectedly", name)
			}
		}
		return nil
	}
	checkInt := func(name string, mode PatchMode, want, old, got int) error {
		switch effectiveMode(mode) {
		case Set:
			if got != want {
				return Ef("verify_failed", "%s not applied", name)
			}
		case Clear:
			if got != 0 {
				return Ef("verify_failed", "%s not cleared", name)
			}
		case Unchanged:
			if got != old {
				return Ef("verify_failed", "%s changed unexpectedly", name)
			}
		}
		return nil
	}
	checkPictures := func(mode PatchMode, want, old, got []Picture) error {
		switch effectiveMode(mode) {
		case Set:
			if !equalPictureContent(want, got) {
				return Ef("verify_failed", "pictures not applied")
			}
		case Clear:
			if len(got) != 0 {
				return Ef("verify_failed", "pictures not cleared")
			}
		case Unchanged:
			if !equalPictures(old, got) {
				return Ef("verify_failed", "pictures changed unexpectedly")
			}
		}
		return nil
	}

	checks := []error{
		checkString("title", patch.Title.Mode, patch.Title.Value, before.Title, after.Title),
		checkStrings("artists", patch.Artists.Mode, patch.Artists.Value, before.Artists, after.Artists),
		checkString("album", patch.Album.Mode, patch.Album.Value, before.Album, after.Album),
		checkStrings("albumArtists", patch.AlbumArtists.Mode, patch.AlbumArtists.Value, before.AlbumArtists, after.AlbumArtists),
		checkPair("track", patch.Track.Mode, patch.Track.Value, before.Track, after.Track),
		checkPair("disc", patch.Disc.Mode, patch.Disc.Value, before.Disc, after.Disc),
		checkString("date", patch.Date.Mode, patch.Date.Value, before.Date, after.Date),
		checkStrings("genres", patch.Genres.Mode, patch.Genres.Value, before.Genres, after.Genres),
		checkStrings("composers", patch.Composers.Mode, patch.Composers.Value, before.Composers, after.Composers),
		checkString("comment", patch.Comment.Mode, patch.Comment.Value, before.Comment, after.Comment),
		checkFloat("bpm", patch.BPM.Mode, patch.BPM.Value, before.BPM, after.BPM),
		checkString("key", patch.Key.Mode, patch.Key.Value, before.Key, after.Key),
		checkInt("rating", patch.Rating.Mode, patch.Rating.Value, before.Rating, after.Rating),
		checkString("isrc", patch.ISRC.Mode, patch.ISRC.Value, before.ISRC, after.ISRC),
		checkString("lyrics", patch.Lyrics.Mode, patch.Lyrics.Value, before.Lyrics, after.Lyrics),
		checkPictures(patch.Pictures.Mode, patch.Pictures.Value, before.Pictures, after.Pictures),
	}
	for _, err := range checks {
		if err != nil {
			return err
		}
	}
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
	count(p.Pictures.Mode)
	return n
}
