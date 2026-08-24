package main

import (
	"EngineTools/internal/database"
	"EngineTools/internal/i18n"
	"EngineTools/internal/id3"
	"EngineTools/internal/process"
	"fmt"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
	"path/filepath"
	"time"
)

// Database lifecycle surface: backups, repair, stats, playlists, sync.

func (a *App) BackupDatabase(note string) string {
	msgs := i18n.Get(a.lang)
	a.log(msgs.DBBackingUp)
	path, err := database.BackupDatabase(note)
	if err != nil {
		a.log(fmt.Sprintf("%s: %v", msgs.DBBackupError, err))
		return ""
	}
	a.log(fmt.Sprintf("%s: %s", msgs.DBBackupComplete, path))
	return "ok"
}

// ListBackups returns the available database backups, newest first.

func (a *App) ListBackups() []database.BackupInfo {
	backups, err := database.ListBackups()
	if err != nil {
		a.log(fmt.Sprintf("ListBackups error: %v", err))
		return []database.BackupInfo{}
	}
	if backups == nil {
		return []database.BackupInfo{}
	}
	return backups
}

// RestoreDatabase restores the Engine Library from the given backup filename.

func (a *App) RestoreDatabase(filename string) string {
	msgs := i18n.Get(a.lang)

	result, err := wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.QuestionDialog,
		Title:   msgs.DBRestoreConfirmTitle,
		Message: msgs.DBRestoreConfirmMessage,
		Buttons: []string{"Yes", "No"},
	})
	if err != nil || result != "Yes" {
		a.log("User cancelled database restore")
		return "cancelled"
	}

	if a.InstallPath != "" {
		running, _ := process.FindRunningExesInDir(a.InstallPath)
		if len(running) > 0 {
			a.log(msgs.KillingProcesses)
			for _, p := range running {
				_ = process.KillProcess(p.PID)
			}
			time.Sleep(2 * time.Second)
		}
	}

	a.log(msgs.DBRestoring)
	if err := database.RestoreDatabase(filename); err != nil {
		a.log(fmt.Sprintf("%s: %v", msgs.FixFailed, err))
		return ""
	}
	a.log(msgs.DBRestoreComplete)
	return "ok"
}

// OptimizeDatabase runs VACUUM and REINDEX on the Engine Library.

func (a *App) OptimizeDatabase() string {
	msgs := i18n.Get(a.lang)

	if a.InstallPath != "" {
		running, _ := process.FindRunningExesInDir(a.InstallPath)
		if len(running) > 0 {
			a.log(msgs.KillingProcesses)
			for _, p := range running {
				_ = process.KillProcess(p.PID)
			}
			time.Sleep(2 * time.Second)
		}
	}

	a.log(msgs.DBOptimizing)
	if err := database.OptimizeDatabase(); err != nil {
		a.log(fmt.Sprintf("%s: %v", msgs.FixFailed, err))
		return ""
	}
	a.log(msgs.DBOptimizeComplete)
	return "ok"
}

func (a *App) ListPlaylists() []database.PlaylistInfo {
	playlists, err := database.ListPlaylists()
	if err != nil {
		a.log(fmt.Sprintf("Playlist error: %v", err))
		return []database.PlaylistInfo{}
	}
	return playlists
}

func (a *App) GetPlaylistTracks(playlistID int) []database.TrackInfo {
	tracks, err := database.GetPlaylistTracks(playlistID)
	if err != nil {
		a.log(fmt.Sprintf("Playlist tracks error: %v", err))
		return []database.TrackInfo{}
	}
	return tracks
}

// RepairDatabase runs integrity checks and repairs on the database.

func (a *App) RepairDatabase() string {
	msgs := i18n.Get(a.lang)

	if a.InstallPath != "" {
		running, _ := process.FindRunningExesInDir(a.InstallPath)
		if len(running) > 0 {
			a.log(msgs.KillingProcesses)
			for _, p := range running {
				_ = process.KillProcess(p.PID)
			}
			time.Sleep(2 * time.Second)
		}
	}

	a.log(msgs.DBRepairStart)
	report, err := database.RepairDatabase()
	if err != nil {
		a.log(fmt.Sprintf("%s: %v", msgs.FixFailed, err))
		return ""
	}
	a.log(msgs.DBRepairDone)
	a.log(report)
	return "ok"
}

// AnalyzeLogs analyzes Engine DJ log files and returns statistics.

func (a *App) CompressCovers(playlistID int, maxKB int) map[string]interface{} {
	const defaultMaxKB = 1024
	if maxKB <= 0 {
		maxKB = defaultMaxKB
	}
	maxBytes := int64(maxKB) * 1024

	// Fetch track paths
	var tracks []database.TrackPath
	var err error
	if playlistID < 0 {
		a.log(fmt.Sprintf("封面压缩：加载全库曲目..."))
		tracks, err = database.GetAllTrackPaths()
	} else {
		a.log(fmt.Sprintf("封面压缩：加载播放列表 %d 的曲目...", playlistID))
		tracks, err = database.GetPlaylistTrackPaths(playlistID)
	}
	if err != nil {
		a.log(fmt.Sprintf("CompressCovers fetch error: %v", err))
		return map[string]interface{}{"error": err.Error()}
	}

	total := len(tracks)
	compressed := 0
	skipped := 0
	failed := 0
	var savedBytes int64
	var errors []string

	a.log(fmt.Sprintf("封面压缩：共 %d 首曲目，目标 ≤%d KB", total, maxKB))
	a.setProgress(0)

	for i, t := range tracks {
		if i%20 == 0 {
			a.setProgress(float64(i) / float64(total))
		}

		res, err := id3.CompressCover(t.Path, maxBytes)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(t.Path), err))
			continue
		}
		if res.Skipped {
			skipped++
		} else {
			compressed++
			savedBytes += res.OriginalSize - res.FinalSize
		}
	}

	a.setProgress(1.0)
	a.log(fmt.Sprintf("封面压缩完成：压缩 %d，跳过 %d，失败 %d（节省 %s）",
		compressed, skipped, failed, formatSavedBytes(savedBytes)))

	errList := errors
	if errList == nil {
		errList = []string{}
	}
	return map[string]interface{}{
		"total":      total,
		"compressed": compressed,
		"skipped":    skipped,
		"failed":     failed,
		"savedBytes": savedBytes,
		"errors":     errList,
	}
}

func formatSavedBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
}

// ---- Library Stats & Play History ----

// GetLibraryStats returns aggregate statistics about the Engine Library.

func (a *App) GetLibraryStats() *database.LibraryStats {
	stats, err := database.GetLibraryStats()
	if err != nil {
		a.log(fmt.Sprintf("GetLibraryStats error: %v", err))
		return nil
	}
	return stats
}

// GetPlayStats returns play-history statistics (most played, recently played, never played).

func (a *App) GetPlayStats() *database.PlayStats {
	stats, err := database.GetPlayStats()
	if err != nil {
		a.log(fmt.Sprintf("GetPlayStats error: %v", err))
		return nil
	}
	return stats
}

// ---- Missing Track Scanner ----

// ScanMissingTracks returns all tracks where the file is no longer available.

func (a *App) ScanMissingTracks() []database.MissingTrack {
	tracks, err := database.FindMissingTracks()
	if err != nil {
		a.log(fmt.Sprintf("ScanMissingTracks error: %v", err))
		return []database.MissingTrack{}
	}
	a.log(fmt.Sprintf("缺失曲目扫描完成：找到 %d 首", len(tracks)))
	return tracks
}

// RemoveMissingTracks deletes the given track IDs from the library (only if unavailable).

func (a *App) RemoveMissingTracks(ids []int) string {
	msgs := i18n.Get(a.lang)

	if a.InstallPath != "" {
		running, _ := process.FindRunningExesInDir(a.InstallPath)
		if len(running) > 0 {
			a.log(msgs.KillingProcesses)
			for _, p := range running {
				_ = process.KillProcess(p.PID)
			}
			time.Sleep(2 * time.Second)
		}
	}

	n, err := database.RemoveMissingTracks(ids)
	if err != nil {
		a.log(fmt.Sprintf("RemoveMissingTracks error: %v", err))
		return ""
	}
	a.log(fmt.Sprintf("已从数据库删除 %d 条缺失曲目记录", n))
	return "ok"
}

// ---- BPM / Key Sync ----

// GetSyncableTracks returns all analyzed tracks with BPM/Key data ready for write-back.

func (a *App) GetSyncableTracks() []database.SyncableTrack {
	tracks, err := database.GetSyncableTracks()
	if err != nil {
		a.log(fmt.Sprintf("GetSyncableTracks error: %v", err))
		return []database.SyncableTrack{}
	}
	return tracks
}

// SyncBPMKeyToTags writes Engine DJ's BPM and musical key analysis back into
// each track file's ID3/RIFF tags. Only available, analyzed tracks are processed.
// Returns a JSON-serialisable summary of successes and failures.

func (a *App) SyncBPMKeyToTags() map[string]interface{} {
	a.log("正在获取分析数据...")
	tracks, err := database.GetSyncableTracks()
	if err != nil {
		a.log(fmt.Sprintf("SyncBPMKeyToTags: fetch error: %v", err))
		return map[string]interface{}{"error": err.Error()}
	}

	total := len(tracks)
	success := 0
	failed := 0
	var errors []string

	for i, t := range tracks {
		// Emit progress every 10 tracks
		if i%10 == 0 {
			a.setProgress(float64(i) / float64(total))
		}

		bpmStr := ""
		if t.BPM > 0 {
			bpmStr = fmt.Sprintf("%.2f", t.BPM)
		}
		keyStr := t.KeyName // e.g. "Am", "C"

		if err := id3.WriteBPMKey(t.Path, bpmStr, keyStr); err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(t.Path), err))
		} else {
			success++
		}
	}

	a.setProgress(1.0)
	a.log(fmt.Sprintf("BPM/Key 写回完成：成功 %d，失败 %d（共 %d）", success, failed, total))

	errList := errors
	if errList == nil {
		errList = []string{}
	}
	return map[string]interface{}{
		"total":   total,
		"success": success,
		"failed":  failed,
		"errors":  errList,
	}
}

func checkIsAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.GetCurrentProcessToken()
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}
