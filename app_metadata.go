package main

import (
	"EngineTools/internal/database"
	"EngineTools/internal/id3"
	"EngineTools/internal/unlock"
	"fmt"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"path/filepath"
)

// Metadata surface: ID3 tools, anti-piracy helpers, USB scan.

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

// ID3ReadTag reads ID3 tags from the given file.

func (a *App) ID3ReadTag(filePath string) id3.TagInfo {
	info, err := id3.ReadTag(filePath)
	if err != nil {
		a.log(fmt.Sprintf("ID3 read error: %v", err))
		return id3.TagInfo{}
	}
	return info
}

// ID3WriteTag writes the given metadata to the file.

func (a *App) ID3WriteTag(filePath, title, artist, album, year, genre string) string {
	if err := id3.WriteTag(filePath, title, artist, album, year, genre); err != nil {
		a.log(fmt.Sprintf("ID3 write error: %v", err))
		return err.Error()
	}
	a.log(fmt.Sprintf("ID3 tags saved: %s", filepath.Base(filePath)))
	return "ok"
}

// ID3GetCover returns the cover art as a base64 data URI.

func (a *App) ID3GetCover(filePath string) string {
	data, err := id3.GetCoverBase64(filePath)
	if err != nil {
		a.log(fmt.Sprintf("ID3 cover read error: %v", err))
		return ""
	}
	return data
}

// ID3SetCover opens a file dialog for the user to pick an image, then embeds it.

func (a *App) ID3SetCover(filePath string) string {
	imgPath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Cover Image",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Images", Pattern: "*.jpg;*.jpeg;*.png;*.gif;*.bmp;*.webp"},
		},
	})
	if err != nil || imgPath == "" {
		return ""
	}
	if err := id3.SetCover(filePath, imgPath); err != nil {
		a.log(fmt.Sprintf("ID3 set cover error: %v", err))
		return err.Error()
	}
	a.log(fmt.Sprintf("Cover set: %s", filepath.Base(filePath)))
	return "ok"
}

// ID3ClearCover removes the cover art.

func (a *App) ID3ClearCover(filePath string) string {
	if err := id3.ClearCover(filePath); err != nil {
		a.log(fmt.Sprintf("ID3 clear cover error: %v", err))
		return err.Error()
	}
	a.log(fmt.Sprintf("Cover cleared: %s", filepath.Base(filePath)))
	return "ok"
}

// ID3ClearAll removes all ID3 tags from the file.

func (a *App) ID3ClearAll(filePath string) string {
	if err := id3.ClearAllTags(filePath); err != nil {
		a.log(fmt.Sprintf("ID3 clear all error: %v", err))
		return err.Error()
	}
	a.log(fmt.Sprintf("All tags cleared: %s", filepath.Base(filePath)))
	return "ok"
}

func (a *App) ID3AntiPiracyV1(dir string) string {
	count, err := id3.AntiPiracyV1(dir)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy v1 error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v1: cleared %d files", count))
	return "ok"
}

func (a *App) ID3AntiPiracyV2(dir string) string {
	count, err := id3.AntiPiracyV2(dir)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy v2 error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v2: shuffled %d files", count))
	return "ok"
}

func (a *App) ID3AntiPiracyRestore(dir string) string {
	count, err := id3.AntiPiracyRestore(dir)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy restore error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy restore: restored %d files", count))
	return "ok"
}

// ID3AntiPiracyV1Paths clears tags for an explicit list of file paths.
// backupKey identifies the backup (e.g. playlist title or "single").

func (a *App) ID3AntiPiracyV1Paths(paths []string, backupKey string) string {
	count, err := id3.AntiPiracyV1Files(paths, backupKey)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy v1 paths error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v1: cleared %d files", count))
	return "ok"
}

// ID3AntiPiracyV2Paths shuffles tags among an explicit list of file paths.

func (a *App) ID3AntiPiracyV2Paths(paths []string, backupKey string) string {
	count, err := id3.AntiPiracyV2Files(paths, backupKey)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy v2 paths error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v2: shuffled %d files", count))
	return "ok"
}

// ID3AntiPiracyRestorePaths restores from the backup identified by backupKey.

func (a *App) ID3AntiPiracyRestorePaths(backupKey string) string {
	count, err := id3.AntiPiracyRestoreFiles(backupKey)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy restore paths error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy restore: restored %d files", count))
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
	key := fmt.Sprintf("playlist_%d", playlistID)
	count, err := id3.AntiPiracyV1Files(paths, key)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy v1 playlist error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v1 playlist %d: cleared %d files", playlistID, count))
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
	key := fmt.Sprintf("playlist_%d", playlistID)
	count, err := id3.AntiPiracyV2Files(paths, key)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy v2 playlist error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy v2 playlist %d: shuffled %d files", playlistID, count))
	return "ok"
}

// ID3AntiPiracyRestorePlaylist restores from the playlist backup.

func (a *App) ID3AntiPiracyRestorePlaylist(playlistID int) string {
	key := fmt.Sprintf("playlist_%d", playlistID)
	count, err := id3.AntiPiracyRestoreFiles(key)
	if err != nil {
		a.log(fmt.Sprintf("Anti-piracy restore playlist error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("Anti-piracy restore playlist %d: restored %d files", playlistID, count))
	return "ok"
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

