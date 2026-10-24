package main

import (
	"EngineTools/internal/database"
	"EngineTools/internal/metadata"
	"EngineTools/internal/unlock"
	"context"
	"encoding/base64"
	"fmt"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Metadata surface: ID3 tools, anti-piracy helpers, USB scan.

// TagInfo is the transitional DTO returned by the legacy ID3ReadTag method.
// New callers should use Document through MetadataRead.
type TagInfo struct {
	FilePath string `json:"filePath"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Year     string `json:"year"`
	Genre    string `json:"genre"`
	HasCover bool   `json:"hasCover"`
}

func (a *App) ID3PickFile() string {
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Audio File",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Audio Files", Pattern: "*.mp3;*.flac;*.wav;*.aiff;*.aif"},
		},
	})
	if err != nil || path == "" {
		return ""
	}
	return path
}

// MetadataPickFile is the format-aware replacement for the legacy ID3 file
// picker. It intentionally keeps the old picker available for older clients.
func (a *App) MetadataPickFile() string {
	return a.ID3PickFile()
}

// MetadataPickDirectory selects a folder for batch metadata editing.
func (a *App) MetadataPickDirectory() string {
	path, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Audio Folder",
	})
	if err != nil {
		return ""
	}
	return path
}

// MetadataScanDirectory returns the supported audio files below dir. The
// frontend deliberately receives paths only; reads remain capability-gated
// through MetadataRead.
func (a *App) MetadataScanDirectory(dir string) ([]string, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, metadataPublicError(metadata.E("file_unreadable", fmt.Errorf("directory path is empty")))
	}
	paths, err := audioPathsInDir(dir)
	if err != nil {
		return nil, metadataPublicError(metadata.E("file_unreadable", err))
	}
	return paths, nil
}

// MetadataRead returns the canonical document for MP3, FLAC, WAV or AIFF.
// Unlike the legacy ID3ReadTag method, read failures are returned to Wails so
// the UI cannot mistake an empty document for a successful read.
func (a *App) MetadataRead(path string) (metadata.Document, error) {
	doc, err := a.readMetadataFull(path)
	if err != nil {
		return metadata.Document{}, metadataPublicError(err)
	}
	return metadata.SanitizeDocumentForUI(doc), nil
}

// readMetadataFull is reserved for in-process operations that must preserve
// original picture bytes (cover replacement, compression and legacy export).
// The public MetadataRead method strips those bytes before crossing Wails.
func (a *App) readMetadataFull(path string) (metadata.Document, error) {
	rt, err := a.metadataServices()
	if err != nil {
		return metadata.Document{}, err
	}
	return rt.service.Read(path)
}

// MetadataApply applies a tri-state patch through the copy-on-write service.
func (a *App) MetadataApply(path string, patch metadata.Patch) (metadata.WriteResult, error) {
	rt, err := a.metadataServices()
	if err != nil {
		return metadata.WriteResult{}, metadataPublicError(err)
	}
	result, err := rt.service.Apply(context.Background(), path, patch)
	if err != nil {
		return metadata.WriteResult{}, metadataPublicError(err)
	}
	return result, nil
}

// MetadataPictures returns picture descriptors without transferring arbitrary
// full-resolution image bytes into the WebView.
func (a *App) MetadataPictures(path string) ([]metadata.Picture, error) {
	doc, err := a.MetadataRead(path)
	if err != nil {
		return nil, err
	}
	return doc.Pictures, nil
}

// MetadataUndo restores the newest complete snapshot for path. Snapshots are
// retained after undo so a failed follow-up operation can still be recovered.
func (a *App) MetadataUndo(path string, snapshotID string) (metadata.WriteResult, error) {
	rt, err := a.metadataServices()
	if err != nil {
		return metadata.WriteResult{}, metadataPublicError(err)
	}
	result, err := rt.service.Undo(context.Background(), path, snapshotID)
	if err != nil {
		return metadata.WriteResult{}, metadataPublicError(err)
	}
	return result, nil
}

// MetadataBatchApply runs the same transactional service for each path and
// preserves per-file status when some files are unsupported or fail.
func (a *App) MetadataBatchApply(paths []string, patch metadata.Patch) (*metadata.BatchResult, error) {
	rt, err := a.metadataServices()
	if err != nil {
		return nil, metadataPublicError(err)
	}
	return rt.batch.Run(context.Background(), paths, patch), nil
}

func (a *App) MetadataBatchStart(paths []string, patch metadata.Patch) (string, error) {
	rt, err := a.metadataServices()
	if err != nil {
		return "", metadataPublicError(err)
	}
	opID := rt.batch.Start(context.Background(), paths, patch)
	if opID == "" {
		return "", metadataPublicError(metadata.E("bridge_unavailable", fmt.Errorf("metadata batch could not start")))
	}
	return opID, nil
}

func (a *App) MetadataCapabilities(path string) (metadata.Capabilities, error) {
	rt, err := a.metadataServices()
	if err != nil {
		return metadata.Capabilities{}, metadataPublicError(err)
	}
	caps, err := rt.service.Capabilities(path)
	if err != nil {
		return metadata.Capabilities{}, metadataPublicError(err)
	}
	return caps, nil
}

func (a *App) MetadataBatchPreview(paths []string, patch metadata.Patch) (*metadata.BatchPreview, error) {
	rt, err := a.metadataServices()
	if err != nil {
		return nil, metadataPublicError(err)
	}
	preview, err := rt.batch.Preview(context.Background(), paths, patch)
	if err != nil {
		return nil, metadataPublicError(err)
	}
	return preview, nil
}

func (a *App) MetadataBatchResult(operationID string) (*metadata.BatchResult, error) {
	a.metadataMu.Lock()
	rt := a.metadata
	a.metadataMu.Unlock()
	if rt == nil {
		return nil, metadataPublicError(metadata.E("operation_not_found", fmt.Errorf("metadata batch has not started")))
	}
	result, ok := rt.batch.Result(operationID)
	if !ok {
		return nil, metadataPublicError(metadata.E("operation_not_found", fmt.Errorf("operation %q is not available", operationID)))
	}
	return result, nil
}

// MetadataCancel cancels a batch operation by its operation ID.
func (a *App) MetadataCancel(operationID string) bool {
	a.metadataMu.Lock()
	rt := a.metadata
	a.metadataMu.Unlock()
	if rt == nil {
		return false
	}
	rt.batch.Cancel(operationID)
	return true
}

// ID3ReadTag reads ID3 tags from the given file.

func (a *App) ID3ReadTag(filePath string) TagInfo {
	doc, err := a.MetadataRead(filePath)
	if err != nil {
		a.log(fmt.Sprintf("ID3 read error: %v", err))
		return TagInfo{}
	}
	artist := ""
	if len(doc.Artists) > 0 {
		artist = doc.Artists[0]
	}
	hasCover := false
	for _, picture := range doc.Pictures {
		if picture.Type == 3 {
			hasCover = true
			break
		}
	}
	return TagInfo{
		FilePath: filePath,
		Title:    doc.Title,
		Artist:   artist,
		Album:    doc.Album,
		Year:     doc.Date,
		Genre:    firstString(doc.Genres),
		HasCover: hasCover,
	}
}

// ID3WriteTag writes the given metadata to the file.

func (a *App) ID3WriteTag(filePath, title, artist, album, year, genre string) string {
	patch := metadata.Patch{
		Title:   legacyStringPatch(title),
		Artists: legacyListPatch(artist),
		Album:   legacyStringPatch(album),
		Date:    legacyStringPatch(year),
		Genres:  legacyListPatch(genre),
	}
	if _, err := a.MetadataApply(filePath, patch); err != nil {
		a.log(fmt.Sprintf("ID3 write error: %v", err))
		return err.Error()
	}
	a.log(fmt.Sprintf("ID3 tags saved: %s", filepath.Base(filePath)))
	return "ok"
}

// ID3GetCover returns the cover art as a base64 data URI.

func (a *App) ID3GetCover(filePath string) string {
	doc, err := a.readMetadataFull(filePath)
	if err != nil {
		a.log(fmt.Sprintf("ID3 cover read error: %v", err))
		return ""
	}
	for _, picture := range doc.Pictures {
		if picture.Type == 3 && len(picture.Data) > 0 {
			return pictureDataURI(picture)
		}
	}
	return ""
}

// ID3SetCover opens a file dialog for the user to pick an image, then embeds it.

func (a *App) ID3SetCover(filePath string) string {
	imgPath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Cover Image",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Images", Pattern: "*.jpg;*.jpeg;*.png;*.gif"},
		},
	})
	if err != nil || imgPath == "" {
		return ""
	}
	data, err := os.ReadFile(imgPath)
	if err != nil {
		a.log(fmt.Sprintf("ID3 set cover read error: %v", err))
		return err.Error()
	}
	mime, width, height, err := metadata.ProbeImage(data)
	if err != nil {
		a.log(fmt.Sprintf("ID3 set cover error: %v", err))
		return err.Error()
	}
	if metadata.NeedsConversion(mime) {
		data, err = metadata.ConvertToJPEG(data)
		if err != nil {
			a.log(fmt.Sprintf("ID3 set cover conversion error: %v", err))
			return err.Error()
		}
		mime, width, height, err = metadata.ProbeImage(data)
		if err != nil {
			a.log(fmt.Sprintf("ID3 set cover conversion error: %v", err))
			return err.Error()
		}
	}
	if _, err := metadata.ValidatePictureForWrite(data); err != nil {
		a.log(fmt.Sprintf("ID3 set cover error: %v", err))
		return err.Error()
	}
	doc, err := a.readMetadataFull(filePath)
	if err != nil {
		a.log(fmt.Sprintf("ID3 set cover read error: %v", err))
		return err.Error()
	}
	replacement := metadata.Picture{
		Type: 3, MIME: mime, Data: data,
		Bytes: int64(len(data)), Width: width, Height: height,
	}
	pictures, err := replaceFrontCover(doc.Pictures, replacement)
	if err != nil {
		a.log(fmt.Sprintf("ID3 set cover error: %v", err))
		return err.Error()
	}
	patch := metadata.Patch{Pictures: metadata.FieldPatch[[]metadata.Picture]{
		Mode: metadata.Set, Value: pictures,
	}}
	if _, err := a.MetadataApply(filePath, patch); err != nil {
		a.log(fmt.Sprintf("ID3 set cover error: %v", err))
		return err.Error()
	}
	a.log(fmt.Sprintf("Cover set: %s", filepath.Base(filePath)))
	return "ok"
}

// ID3ClearCover removes the cover art.

func (a *App) ID3ClearCover(filePath string) string {
	doc, err := a.readMetadataFull(filePath)
	if err != nil {
		a.log(fmt.Sprintf("ID3 clear cover read error: %v", err))
		return err.Error()
	}
	pictures, found, err := clearFrontCover(doc.Pictures)
	if err != nil {
		a.log(fmt.Sprintf("ID3 clear cover error: %v", err))
		return err.Error()
	}
	if !found {
		return "ok"
	}
	patch := metadata.Patch{}
	if len(pictures) == 0 {
		patch.Pictures = metadata.FieldPatch[[]metadata.Picture]{Mode: metadata.Clear}
	} else {
		patch.Pictures = metadata.FieldPatch[[]metadata.Picture]{Mode: metadata.Set, Value: pictures}
	}
	if _, err := a.MetadataApply(filePath, patch); err != nil {
		a.log(fmt.Sprintf("ID3 clear cover error: %v", err))
		return err.Error()
	}
	a.log(fmt.Sprintf("Cover cleared: %s", filepath.Base(filePath)))
	return "ok"
}

// ID3ClearAll removes all ID3 tags from the file.

func (a *App) ID3ClearAll(filePath string) string {
	caps, err := a.MetadataCapabilities(filePath)
	if err != nil {
		a.log(fmt.Sprintf("ID3 clear all capability error: %v", err))
		return err.Error()
	}
	// Clear only fields represented by the detected container. Submitting an
	// unsupported field would make the transactional service reject the whole
	// patch, preventing supported fields from being cleared.
	var patch metadata.Patch
	if appWritableCapability(caps, "title") {
		patch.Title = metadata.FieldPatch[string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "artists", "artist") {
		patch.Artists = metadata.FieldPatch[[]string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "albumArtists", "albumArtist") {
		patch.AlbumArtists = metadata.FieldPatch[[]string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "album") {
		patch.Album = metadata.FieldPatch[string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "track") {
		patch.Track = metadata.FieldPatch[metadata.NumberPair]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "disc") {
		patch.Disc = metadata.FieldPatch[metadata.NumberPair]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "date") {
		patch.Date = metadata.FieldPatch[string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "genres", "genre") {
		patch.Genres = metadata.FieldPatch[[]string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "composers", "composer") {
		patch.Composers = metadata.FieldPatch[[]string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "comment") {
		patch.Comment = metadata.FieldPatch[string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "bpm") {
		patch.BPM = metadata.FieldPatch[float64]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "key") {
		patch.Key = metadata.FieldPatch[string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "rating") {
		patch.Rating = metadata.FieldPatch[int]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "isrc") {
		patch.ISRC = metadata.FieldPatch[string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "lyrics") {
		patch.Lyrics = metadata.FieldPatch[string]{Mode: metadata.Clear}
	}
	if appWritableCapability(caps, "pictures") {
		patch.Pictures = metadata.FieldPatch[[]metadata.Picture]{Mode: metadata.Clear}
	}
	if _, applyErr := a.MetadataApply(filePath, patch); applyErr != nil {
		a.log(fmt.Sprintf("ID3 clear all error: %v", applyErr))
		return applyErr.Error()
	}
	a.log(fmt.Sprintf("All tags cleared: %s", filepath.Base(filePath)))
	return "ok"
}

func appWritableCapability(caps metadata.Capabilities, names ...string) bool {
	if caps.Writable == nil {
		return false
	}
	for _, name := range names {
		if caps.Writable[name] {
			return true
		}
	}
	return false
}

func pictureDataURI(picture metadata.Picture) string {
	if len(picture.Data) == 0 {
		return ""
	}
	mime := picture.MIME
	if mime == "" {
		mime = "application/octet-stream"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(picture.Data)
}

// replaceFrontCover replaces the first front-cover picture and drops any
// duplicate front-cover entries. Every retained non-front picture must carry
// its payload so a full-list patch cannot accidentally delete it.
func replaceFrontCover(pictures []metadata.Picture, replacement metadata.Picture) ([]metadata.Picture, error) {
	if len(replacement.Data) == 0 {
		return nil, metadata.E("picture_data_missing", fmt.Errorf("replacement picture has no payload"))
	}
	next := make([]metadata.Picture, 0, len(pictures)+1)
	replaced := false
	for _, picture := range pictures {
		if picture.Type == 3 {
			if !replaced {
				candidate := replacement
				candidate.ID = picture.ID
				candidate.Description = picture.Description
				next = append(next, candidate)
				replaced = true
			}
			continue
		}
		if len(picture.Data) == 0 {
			return nil, metadata.E("picture_data_missing", fmt.Errorf("picture %q has no payload", picture.ID))
		}
		next = append(next, picture)
	}
	if !replaced {
		candidate := replacement
		candidate.ID = "legacy-front"
		used := make(map[string]bool, len(next)+1)
		for _, picture := range next {
			used[picture.ID] = true
		}
		for used[candidate.ID] {
			candidate.ID += "-1"
		}
		next = append(next, candidate)
	}
	return next, nil
}

func clearFrontCover(pictures []metadata.Picture) ([]metadata.Picture, bool, error) {
	next := make([]metadata.Picture, 0, len(pictures))
	found := false
	for _, picture := range pictures {
		if picture.Type == 3 {
			found = true
			continue
		}
		if len(picture.Data) == 0 {
			return nil, false, metadata.E("picture_data_missing", fmt.Errorf("picture %q has no payload", picture.ID))
		}
		next = append(next, picture)
	}
	return next, found, nil
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func legacyStringPatch(value string) metadata.FieldPatch[string] {
	if value == "" {
		return metadata.FieldPatch[string]{Mode: metadata.Clear}
	}
	return metadata.FieldPatch[string]{Mode: metadata.Set, Value: value}
}

func legacyListPatch(value string) metadata.FieldPatch[[]string] {
	if value == "" {
		return metadata.FieldPatch[[]string]{Mode: metadata.Clear}
	}
	return metadata.FieldPatch[[]string]{Mode: metadata.Set, Value: []string{value}}
}

func (a *App) ID3AntiPiracyV1(dir string) string {
	paths, err := audioPathsInDir(dir)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy v1 error: %v", err))
		return ""
	}
	result, err := a.runAntiV1(paths)
	if err != nil || result.Failed > 0 || result.Blocked > 0 || result.Updated == 0 {
		a.log(fmt.Sprintf("Anti-piracy v1 error: %v", antiResultError(result, err)))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v1: cleared %d files", result.Updated))
	return "ok"
}

func (a *App) ID3AntiPiracyV2(dir string) string {
	paths, err := audioPathsInDir(dir)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy v2 error: %v", err))
		return ""
	}
	result, err := a.runAntiV2(paths)
	if err != nil || result.Failed > 0 || result.Blocked > 0 {
		a.log(fmt.Sprintf("Anti-piracy v2 error: %v", antiResultError(result, err)))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v2: shuffled %d files", result.Updated))
	return "ok"
}

func (a *App) ID3AntiPiracyRestore(dir string) string {
	paths, err := audioPathsInDir(dir)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy restore error: %v", err))
		return ""
	}
	result, err := a.runAntiRestore(paths)
	if err != nil || result.Failed > 0 || result.Blocked > 0 {
		a.log(fmt.Sprintf("Anti-piracy restore error: %v", antiResultError(result, err)))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy restore: restored %d files", result.Updated))
	return "ok"
}

// ID3AntiPiracyV1Paths clears tags for an explicit list of file paths.
// backupKey identifies the backup (e.g. playlist title or "single").

func (a *App) ID3AntiPiracyV1Paths(paths []string, backupKey string) string {
	result, err := a.runAntiV1(paths)
	if err != nil || result.Failed > 0 || result.Blocked > 0 {
		a.log(fmt.Sprintf("Anti-piracy v1 paths error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v1: cleared %d files (backup key %s)", result.Updated, backupKey))
	return "ok"
}

// ID3AntiPiracyV2Paths shuffles tags among an explicit list of file paths.

func (a *App) ID3AntiPiracyV2Paths(paths []string, backupKey string) string {
	result, err := a.runAntiV2(paths)
	if err != nil || result.Failed > 0 || result.Blocked > 0 {
		a.log(fmt.Sprintf("Anti-piracy v2 paths error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v2: shuffled %d files (backup key %s)", result.Updated, backupKey))
	return "ok"
}

// ID3AntiPiracyRestorePaths restores the latest complete recovery snapshot
// for one explicit file path. The legacy backup-key form is not supported by
// the snapshot store, so callers must pass the audio file path itself.

func (a *App) ID3AntiPiracyRestorePaths(filePath string) string {
	if strings.TrimSpace(filePath) == "" {
		return ""
	}
	result, err := a.runAntiRestore([]string{filePath})
	if err != nil || result.Failed > 0 || result.Blocked > 0 {
		a.log(fmt.Sprintf("Anti-piracy restore path error: %v", antiResultError(result, err)))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy restore: restored %d file", result.Updated))
	return "ok"
}

// ID3AntiPiracyV1Playlist clears tags for all tracks in the given playlist.

func (a *App) ID3AntiPiracyV1Playlist(playlistID int) string {
	tracks, err := database.GetPlaylistTrackPaths(playlistID)
	if err != nil || len(tracks) == 0 {
		a.log(fmt.Sprintf("Anti-piracy playlist fetch error: %v", err))
		return ""
	}
	paths := make([]string, len(tracks))
	for i, t := range tracks {
		paths[i] = t.Path
	}
	result, err := a.runAntiV1(paths)
	if err != nil || result.Failed > 0 || result.Blocked > 0 {
		a.log(fmt.Sprintf("Anti-piracy v1 playlist error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v1 playlist %d: cleared %d files", playlistID, result.Updated))
	return "ok"
}

// ID3AntiPiracyV2Playlist shuffles tags among all tracks in the given playlist.

func (a *App) ID3AntiPiracyV2Playlist(playlistID int) string {
	tracks, err := database.GetPlaylistTrackPaths(playlistID)
	if err != nil || len(tracks) == 0 {
		a.log(fmt.Sprintf("Anti-piracy playlist fetch error: %v", err))
		return ""
	}
	paths := make([]string, len(tracks))
	for i, t := range tracks {
		paths[i] = t.Path
	}
	result, err := a.runAntiV2(paths)
	if err != nil || result.Failed > 0 || result.Blocked > 0 {
		a.log(fmt.Sprintf("Anti-piracy v2 playlist error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v2 playlist %d: shuffled %d files", playlistID, result.Updated))
	return "ok"
}

// ID3AntiPiracyRestorePlaylist restores from the playlist backup.

func (a *App) ID3AntiPiracyRestorePlaylist(playlistID int) string {
	tracks, err := database.GetPlaylistTrackPaths(playlistID)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy restore playlist error: %v", err))
		return ""
	}
	paths := make([]string, len(tracks))
	for i, t := range tracks {
		paths[i] = t.Path
	}
	result, err := a.runAntiRestore(paths)
	if err != nil || result.Failed > 0 || result.Blocked > 0 {
		a.log(fmt.Sprintf("Anti-piracy restore playlist error: %v", antiResultError(result, err)))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy restore playlist %d: restored %d files", playlistID, result.Updated))
	return "ok"
}

func audioPathsInDir(dir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".mp3", ".flac", ".wav", ".aiff", ".aif":
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func (a *App) runAntiV1(paths []string) (*metadata.AntiPiracyResult, error) {
	rt, err := a.metadataServices()
	if err != nil {
		return nil, metadataPublicError(err)
	}
	return rt.anti.ClearV1(context.Background(), paths)
}

func (a *App) runAntiV2(paths []string) (*metadata.AntiPiracyResult, error) {
	rt, err := a.metadataServices()
	if err != nil {
		return nil, metadataPublicError(err)
	}
	return rt.anti.DerangeV2(context.Background(), paths)
}

func (a *App) runAntiRestore(paths []string) (*metadata.AntiPiracyResult, error) {
	rt, err := a.metadataServices()
	if err != nil {
		return nil, metadataPublicError(err)
	}
	return rt.anti.RestoreAll(context.Background(), paths)
}

func antiResultError(result *metadata.AntiPiracyResult, err error) error {
	if err != nil {
		return err
	}
	if result == nil {
		return fmt.Errorf("empty anti-piracy result")
	}
	return fmt.Errorf("%d files failed, %d blocked", result.Failed, result.Blocked)
}

func (a *App) USBUnlockAvailable() string {
	return unlock.FindRemovableDriveWithLibrary()
}

// USBUnlockScan scans for processes blocking the given drive (excluding Engine DJ apps).

func (a *App) USBUnlockScan(drive string) []unlock.HandleInfo {
	procs, err := unlock.FindBlockingProcesses(drive)
	if err != nil {
		a.log(fmt.Sprintf("USB unlock scan error: %v", err))
		return []unlock.HandleInfo{}
	}
	if len(procs) == 0 {
		a.log("No blocking processes found")
		return []unlock.HandleInfo{}
	}
	a.log(fmt.Sprintf("Found %d blocking process(es)", len(procs)))
	return procs
}

// USBUnlockKill terminates blocking processes on the given drive.
