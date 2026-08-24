package main

import (
	"EngineTools/internal/database"
	"EngineTools/internal/experimental"
	"EngineTools/internal/i18n"
	"EngineTools/internal/logs"
	"EngineTools/internal/process"
	"EngineTools/internal/update"
	"fmt"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"path/filepath"
)

// Status, drives, logs, cache, language and update surface.

func (a *App) GetStatus() StatusInfo {
	var procs []ProcessItem
	var running []process.ProcessInfo

	if a.InstallPath != "" {
		var err error
		running, err = process.FindRunningExesInDir(a.InstallPath)
		if err == nil {
			for _, p := range running {
				procs = append(procs, ProcessItem{
					Name: p.Name,
					PID:  p.PID,
				})
			}
		}
	}

	acpValue := ""
	if a.UTF8Enabled {
		acpValue = "65001"
	} else {
		acpValue = "936"
	}

	return StatusInfo{
		InstallPath:        a.InstallPath,
		EngineVersion:      a.EngineVersion,
		WindowsVersion:     a.WindowsVersion,
		UTF8Enabled:        a.UTF8Enabled,
		ACPValue:           acpValue,
		ManifestConfigured: a.ManifestConfigured,
		IsAdmin:            a.IsAdmin,
		StemsDetected:      a.StemsDetected,
		DBDetected:         a.DBDetected,
		DBPath:             a.DBPath,
		ProcessRunning:     len(running) > 0,
		RunningProcesses:   procs,
	}
}

func (a *App) Refresh() StatusInfo {
	go a.detectStatus()
	return a.GetStatus()
}

func (a *App) AnalyzeLogs() *logs.LogStats {
	a.log("正在分析日志文件...")
	stats, err := logs.AnalyzeLogs()
	if err != nil {
		a.log(fmt.Sprintf("日志分析失败: %v", err))
		return nil
	}
	a.log(fmt.Sprintf("分析完成: %d个文件, %d条警告, %d条错误", stats.TotalFiles, stats.WarningCount, stats.ErrorCount))
	return stats
}

// OpenLogsDir opens the Engine DJ logs directory in Explorer.

func (a *App) OpenLogsDir() string {
	logsDir := logs.GetLogsDir()
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		a.log("日志目录不存在")
		return ""
	}
	if err := process.OpenDirectory(logsDir); err != nil {
		a.log(fmt.Sprintf("Failed to open logs dir: %v", err))
		return ""
	}
	return "ok"
}

// CleanCache removes Engine DJ's QML and pipeline cache directories.
// This can fix UI glitches and rendering issues after Engine DJ updates.

func (a *App) CleanCache() string {
	cacheDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "AIR Music Technology", "EnginePrime", "cache")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		a.log("缓存目录不存在，无需清理")
		return "empty"
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		a.log(fmt.Sprintf("读取缓存目录失败: %v", err))
		return ""
	}

	totalRemoved := 0
	for _, entry := range entries {
		entryPath := filepath.Join(cacheDir, entry.Name())
		if entry.IsDir() {
			if err := os.RemoveAll(entryPath); err != nil {
				a.log(fmt.Sprintf("删除 %s 失败: %v", entry.Name(), err))
				continue
			}
		} else {
			if err := os.Remove(entryPath); err != nil {
				a.log(fmt.Sprintf("删除 %s 失败: %v", entry.Name(), err))
				continue
			}
		}
		totalRemoved++
	}

	a.log(fmt.Sprintf("已清理 %d 个缓存项", totalRemoved))
	return "ok"
}

// GetCacheSize returns the size of the Engine DJ cache directory in bytes.

func (a *App) GetCacheSize() int64 {
	cacheDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "AIR Music Technology", "EnginePrime", "cache")
	var totalSize int64
	filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize
}

// ScanMSIOrphans scans the registry for orphaned MSI installer products.

func (a *App) OpenRegionSettings() string {
	if err := process.OpenControlPanel(); err != nil {
		a.log(fmt.Sprintf("Failed to open region settings: %v", err))
		return err.Error()
	}
	a.log("Opened region settings")
	return ""
}

func (a *App) OpenRepository() {
	wailsRuntime.BrowserOpenURL(a.ctx, "https://github.com/LaokeQwQ/EngineTools")
}

// OpenInstallDir opens the Engine DJ install directory in Explorer.

func (a *App) OpenInstallDir() string {
	if a.InstallPath == "" {
		return ""
	}
	if err := process.OpenDirectory(a.InstallPath); err != nil {
		a.log(fmt.Sprintf("Failed to open install directory: %v", err))
		return err.Error()
	}
	return ""
}

// OpenStemsDir opens the STEM processor directory in Explorer.

func (a *App) OpenStemsDir() string {
	if a.StemsPath == "" {
		return ""
	}
	if err := process.OpenDirectory(a.StemsPath); err != nil {
		a.log(fmt.Sprintf("Failed to open STEM directory: %v", err))
		return err.Error()
	}
	return ""
}

// OpenDBDir opens the folder containing the Engine Library database in Explorer,
// with the m.db file selected.

func (a *App) OpenDBDir() string {
	if a.DBPath == "" {
		return ""
	}
	if err := process.OpenDirectory(a.DBPath); err != nil {
		a.log(fmt.Sprintf("Failed to open database directory: %v", err))
		return err.Error()
	}
	return ""
}

// ListDrives returns the present drive letters (e.g. ["C:", "D:"]) for the
// database drive picker.

func (a *App) ListDrives() []DriveInfo {
	allDrives := database.ListDrives()
	var result []DriveInfo

	for _, drive := range allDrives {
		// Check if this drive has an Engine Library
		dbPath, err := database.FindInDrive(drive)
		if err != nil {
			continue // Skip drives without Engine Library
		}

		// Count tracks in this database
		trackCount := database.CountTracks(dbPath)

		result = append(result, DriveInfo{
			Letter:     drive,
			TrackCount: trackCount,
		})
	}

	return result
}

// SelectDrive scans the given drive for an Engine Library and, if found, pins
// it as the active database. Returns the resolved m.db path, or "" if none was
// found on that drive.

func (a *App) SelectDrive(drive string) string {
	msgs := i18n.Get(a.lang)
	path, err := database.FindInDrive(drive)
	if err != nil {
		database.ClearLibraryPath()
		a.log(fmt.Sprintf("%s: %s", msgs.DBLibraryNotFound, drive))
		go a.detectStatus()
		return ""
	}
	a.services.mu.Lock()
	a.services.libraryPath = path
	a.services.mu.Unlock()
	database.SetLibraryPath(path)
	a.log(fmt.Sprintf("%s: %s", msgs.DBLibraryPathLabel, path))
	go a.detectStatus()
	return path
}

func (a *App) SetLanguage(lang string) string {
	a.lang = i18n.Lang(lang)
	go a.detectStatus()
	return ""
}

// ---- ID3 Editor ----

// ID3PickFile opens a native file dialog for the user to select an audio file.

func (a *App) CheckForUpdates(currentVersion string) map[string]interface{} {
	release, hasUpdate, err := update.CheckUpdate(currentVersion)
	if err != nil {
		a.log(fmt.Sprintf("Update check failed: %v", err))
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return map[string]interface{}{
		"hasUpdate": hasUpdate,
		"version":   release.TagName,
		"name":      release.Name,
		"notes":     release.Body,
		"url":       release.HTMLURL,
	}
}

func (a *App) GetMessages() i18n.Messages {
	return i18n.Get(a.lang)
}

func (a *App) GetAvailableLanguages() []map[string]string {
	langs := i18n.AvailableLangs()
	result := make([]map[string]string, len(langs))
	for i, l := range langs {
		result[i] = map[string]string{
			"code":   string(l),
			"native": i18n.LangDisplayName(l),
		}
	}
	return result
}

// shutdown is called by Wails when the application is closing.
// Context cancellation propagates to the watchDrives goroutine via a.ctx.Done().

func (a *App) LogExperimentalEnabled() string {
	if err := experimental.LogEnabled(); err != nil {
		a.log(fmt.Sprintf("experimental log error: %v", err))
		return err.Error()
	}
	a.log("实验性功能已开启（记录已写入审计文件）")
	return "ok"
}

// ---- Cover Art Compression ----

// CompressCovers compresses embedded cover art in all tracks (playlistID=-1)
// or in a specific playlist. Covers larger than maxKB kilobytes are
// re-encoded as JPEG at progressively lower quality until they fit.
// Returns a JSON-serialisable summary of the operation.
