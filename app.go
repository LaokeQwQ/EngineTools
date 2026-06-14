package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"

	"EngineTools/internal/database"
	"EngineTools/internal/i18n"
	"EngineTools/internal/id3"
	"EngineTools/internal/manifest"
	"EngineTools/internal/midi"
	"EngineTools/internal/msi"
	"EngineTools/internal/process"
	"EngineTools/internal/registry"
	"EngineTools/internal/unlock"
)

type App struct {
	ctx                 context.Context
	lang                i18n.Lang
	Logs                []string
	InstallPath         string
	EngineVersion       string
	WindowsVersion      string
	UTF8Enabled         bool
	ManifestConfigured  bool
	IsAdmin             bool
	StemsPath           string
	StemsDetected       bool
	DBPath              string
	DBDetected          bool
	Progress            float64
}

type StatusInfo struct {
	InstallPath        string         `json:"installPath"`
	EngineVersion      string         `json:"engineVersion"`
	WindowsVersion     string         `json:"windowsVersion"`
	UTF8Enabled        bool           `json:"utf8Enabled"`
	ACPValue           string         `json:"acpValue"`
	ManifestConfigured bool           `json:"manifestConfigured"`
	IsAdmin            bool           `json:"isAdmin"`
	StemsDetected      bool           `json:"stemsDetected"`
	DBDetected         bool           `json:"dbDetected"`
	DBPath             string         `json:"dbPath"`
	ProcessRunning     bool           `json:"processRunning"`
	RunningProcesses   []ProcessItem  `json:"runningProcesses"`
}

type ProcessItem struct {
	Name string `json:"name"`
	PID  uint32 `json:"pid"`
}

func NewApp() *App {
	return &App{
		lang:    i18n.DetectLang(),
		Logs:    []string{},
		IsAdmin: checkIsAdmin(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.detectStatus()
	go a.watchDrives()
}

// watchDrives polls the set of mounted logical drives and re-runs detection
// whenever it changes (e.g. a USB stick is inserted or removed), so the
// Engine Library on external drives is picked up automatically.
func (a *App) watchDrives() {
	last := getLogicalDrives()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		current := getLogicalDrives()
		if current != last {
			last = current
			a.detectStatus()
		}
	}
}

// getLogicalDrives returns the bitmask of mounted drives from the Win32
// GetLogicalDrives API. Bit 0 = A:, bit 1 = B:, etc.
func getLogicalDrives() uint32 {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return 0
	}
	return mask
}

// findStemsProcessorDir locates the EnginePrime bin directory that contains
// stems-processor.exe. Returns the directory and true if found.
func findStemsProcessorDir() (string, bool) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", false
	}
	binDir := filepath.Join(localAppData, "AIR Music Technology", "EnginePrime", "bin")
	exePath := filepath.Join(binDir, "stems-processor.exe")
	if _, err := os.Stat(exePath); err != nil {
		return "", false
	}
	return binDir, true
}

func (a *App) log(msg string) {
	timestamp := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("[%s] %s", timestamp, msg)
	a.Logs = append(a.Logs, entry)
	wailsRuntime.EventsEmit(a.ctx, "log", entry)
}

func (a *App) setProgress(value float64) {
	a.Progress = value
	wailsRuntime.EventsEmit(a.ctx, "progress", value)
}

func (a *App) detectStatus() {
	msgs := i18n.Get(a.lang)

	a.log(msgs.StatusChecking)
	a.setProgress(0.05)

	a.WindowsVersion = registry.GetWindowsVersion()
	a.log(fmt.Sprintf("%s: %s", msgs.WindowsVersionLabel, a.WindowsVersion))
	a.setProgress(0.15)

	path, err := registry.FindEngineDJInstallPath()
	if err != nil {
		a.InstallPath = ""
		a.EngineVersion = ""
		a.log(msgs.InstallPathNotFound)
	} else {
		a.InstallPath = path
		a.EngineVersion = registry.GetEngineDJVersion()
		if a.EngineVersion == "" {
			a.EngineVersion = registry.FindEngineDJVersionFromPath(path)
		}
		if a.EngineVersion != "" {
			a.log(fmt.Sprintf("%s: %s (v%s)", msgs.InstallPathLabel, path, a.EngineVersion))
		} else {
			a.log(fmt.Sprintf("%s: %s", msgs.InstallPathLabel, path))
		}
	}
	a.setProgress(0.35)

	utf8Enabled, acpValue, err := registry.IsUTF8Enabled()
	if err != nil {
		a.UTF8Enabled = false
		a.log(fmt.Sprintf("%s: %s", msgs.UTF8StatusLabel, err.Error()))
	} else {
		a.UTF8Enabled = utf8Enabled
		if utf8Enabled {
			a.log(fmt.Sprintf("%s: %s (ACP=%s)", msgs.UTF8StatusLabel, msgs.UTF8Enabled, acpValue))
		} else {
			a.log(fmt.Sprintf("%s: %s (ACP=%s)", msgs.UTF8StatusLabel, msgs.UTF8Disabled, acpValue))
		}
	}
	a.setProgress(0.65)

	a.StemsPath, a.StemsDetected = findStemsProcessorDir()
	if a.StemsDetected {
		a.log(fmt.Sprintf("%s: %s", msgs.StemsStatusLabel, a.StemsPath))
	} else {
		a.log(fmt.Sprintf("%s: %s", msgs.StemsStatusLabel, msgs.StemsNotFound))
	}

	manifestOK, _ := registry.GetPreferExternalManifest()
	if a.InstallPath != "" {
		manifestOK = manifestOK && manifest.ManifestExists(a.InstallPath)
	}
	if a.StemsDetected {
		manifestOK = manifestOK && manifest.ManifestExists(a.StemsPath)
	}
	dbPath, dbErr := database.ResolveLibrary()
	a.DBPath = ""
	a.DBDetected = false
	if dbErr == nil && dbPath != "" {
		a.DBPath = dbPath
		a.DBDetected = true
		a.log(fmt.Sprintf("%s: %s", msgs.DBLibraryPathLabel, dbPath))
	} else {
		a.log(fmt.Sprintf("%s: %s", msgs.DBLibraryPathLabel, msgs.DBLibraryNotFound))
	}

	a.ManifestConfigured = manifestOK

	statusLabel := msgs.ManifestNotExists
	if manifestOK {
		statusLabel = msgs.ManifestExists
	}
	a.log(fmt.Sprintf("%s: %s", msgs.ManifestStatusLabel, statusLabel))
	a.setProgress(1.0)

	wailsRuntime.EventsEmit(a.ctx, "statusUpdate", a.GetStatus())
}

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

func (a *App) FixCJKIssues() string {
	msgs := i18n.Get(a.lang)

	if a.InstallPath == "" {
		a.log(msgs.InstallPathNotFound)
		return msgs.InstallPathNotFound
	}

	a.setProgress(0)
	a.log(msgs.Checking + "...")

	running, err := process.FindRunningExesInDir(a.InstallPath)
	if err != nil {
		a.log(fmt.Sprintf("Error checking processes: %v", err))
	}

	if len(running) > 0 {
		var names []string
		for _, p := range running {
			names = append(names, fmt.Sprintf("%s (PID: %d)", p.Name, p.PID))
		}

		confirmMsg := fmt.Sprintf(msgs.ProcessRunningMessage, strings.Join(names, "\n"))
		result, err := wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
			Type:    wailsRuntime.QuestionDialog,
			Title:   msgs.ProcessRunningTitle,
			Message: confirmMsg,
			Buttons: []string{"Yes", "No"},
		})

		if err != nil || result != "Yes" {
			a.log("User cancelled process termination")
			return "cancelled"
		}

		a.log(msgs.KillingProcesses)
		for _, p := range running {
			if err := process.KillProcess(p.PID); err != nil {
				a.log(fmt.Sprintf("Failed to kill %s: %v", p.Name, err))
			} else {
				a.log(fmt.Sprintf("Killed %s (PID: %d)", p.Name, p.PID))
			}
		}

		time.Sleep(2 * time.Second)
	}
	a.setProgress(0.3)

	exes, err := manifest.ListExes(a.InstallPath)
	if err != nil || len(exes) == 0 {
		a.log(msgs.ExeNotFound)
		return msgs.ExeNotFound
	}

	a.log(msgs.WritingManifest)
	count, err := manifest.WriteManifest(a.InstallPath)
	if err != nil {
		a.log(fmt.Sprintf("%s: %v", msgs.ManifestWriteError, err))
		return msgs.ManifestWriteError
	}
	a.log(fmt.Sprintf("%s (%d)", msgs.ManifestWritten, count))

	if a.StemsDetected && a.StemsPath != "" {
		stemsCount, err := manifest.WriteManifest(a.StemsPath)
		if err != nil {
			a.log(fmt.Sprintf("%s: %v", msgs.ManifestWriteError, err))
		} else {
			a.log(fmt.Sprintf("%s: STEM (%d)", msgs.ManifestWritten, stemsCount))
		}
	}
	a.setProgress(0.6)

	a.log(msgs.WritingRegistry)
	if err := registry.SetPreferExternalManifest(); err != nil {
		a.log(fmt.Sprintf("%s: %v", msgs.RegistryWriteError, err))
		return msgs.RegistryWriteError
	}
	a.log(msgs.RegistryWritten)
	a.setProgress(0.8)

	a.log(msgs.RefreshingSystem)
	process.RefreshSystemSettings()
	a.log(msgs.SystemRefreshed)
	a.setProgress(1.0)

	a.log(msgs.FixCompleteTip)
	a.ManifestConfigured = true

	wailsRuntime.EventsEmit(a.ctx, "statusUpdate", a.GetStatus())

	_, _ = wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.InfoDialog,
		Title:   msgs.FixComplete,
		Message: msgs.FixCompleteTip,
	})

	return "ok"
}

// RestoreCJKFix reverts the CJK fix: removes the .manifest files and resets
// the PreferExternalManifest registry value.
func (a *App) RestoreCJKFix() string {
	msgs := i18n.Get(a.lang)

	result, err := wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.QuestionDialog,
		Title:   msgs.RestoreConfirmTitle,
		Message: msgs.RestoreConfirmMessage,
		Buttons: []string{"Yes", "No"},
	})
	if err != nil || result != "Yes" {
		a.log("User cancelled restore")
		return "cancelled"
	}

	a.setProgress(0)
	a.log(msgs.Checking + "...")

	if a.InstallPath != "" {
		running, _ := process.FindRunningExesInDir(a.InstallPath)
		if len(running) > 0 {
			var names []string
			for _, p := range running {
				names = append(names, fmt.Sprintf("%s (PID: %d)", p.Name, p.PID))
			}
			confirmMsg := fmt.Sprintf(msgs.ProcessRunningMessage, strings.Join(names, "\n"))
			res, err := wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
				Type:    wailsRuntime.QuestionDialog,
				Title:   msgs.ProcessRunningTitle,
				Message: confirmMsg,
				Buttons: []string{"Yes", "No"},
			})
			if err != nil || res != "Yes" {
				a.log("User cancelled process termination")
				return "cancelled"
			}
			a.log(msgs.KillingProcesses)
			for _, p := range running {
				if err := process.KillProcess(p.PID); err != nil {
					a.log(fmt.Sprintf("Failed to kill %s: %v", p.Name, err))
				} else {
					a.log(fmt.Sprintf("Killed %s (PID: %d)", p.Name, p.PID))
				}
			}
			time.Sleep(2 * time.Second)
		}
	}
	a.setProgress(0.3)

	a.log(msgs.DeletingManifests)
	if a.InstallPath != "" {
		count, err := manifest.DeleteManifests(a.InstallPath)
		if err != nil {
			a.log(fmt.Sprintf("%s: %v", msgs.FixFailed, err))
		} else {
			a.log(fmt.Sprintf("%s (%d)", msgs.ManifestsDeleted, count))
		}
	}
	if a.StemsDetected && a.StemsPath != "" {
		stemsCount, err := manifest.DeleteManifests(a.StemsPath)
		if err != nil {
			a.log(fmt.Sprintf("%s: %v", msgs.FixFailed, err))
		} else {
			a.log(fmt.Sprintf("%s: STEM (%d)", msgs.ManifestsDeleted, stemsCount))
		}
	}
	a.setProgress(0.6)

	a.log(msgs.DeletingRegistry)
	if err := registry.DeletePreferExternalManifest(); err != nil {
		a.log(fmt.Sprintf("%s: %v", msgs.RegistryWriteError, err))
		return msgs.FixFailed
	}
	a.log(msgs.RegistryDeleted)
	a.setProgress(0.8)

	a.log(msgs.RefreshingSystem)
	process.RefreshSystemSettings()
	a.log(msgs.SystemRefreshed)
	a.setProgress(1.0)

	a.log(msgs.RestoreCompleteTip)
	a.ManifestConfigured = false

	wailsRuntime.EventsEmit(a.ctx, "statusUpdate", a.GetStatus())

	_, _ = wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.InfoDialog,
		Title:   msgs.RestoreComplete,
		Message: msgs.RestoreCompleteTip,
	})

	return "ok"
}

// BackupDatabase copies the Engine Library m.db into the backup folder, with an
// optional note.
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

// ScanMSIOrphans scans the registry for orphaned MSI installer products.
func (a *App) ScanMSIOrphans() []msi.OrphanedMSI {
	msgs := i18n.Get(a.lang)
	a.log(msgs.MSIScanning)
	orphans, err := msi.ScanOrphans()
	if err != nil {
		a.log(fmt.Sprintf("%s: %v", msgs.MSICleanError, err))
		return []msi.OrphanedMSI{}
	}
	if len(orphans) == 0 {
		a.log(msgs.MSINoOrphans)
		return []msi.OrphanedMSI{}
	}
	a.log(fmt.Sprintf(msgs.MSIFoundOrphans, len(orphans)))
	return orphans
}

// CleanMSIOrphans removes the given MSI products via msizap.
func (a *App) CleanMSIOrphans(productCodes []string) string {
	msgs := i18n.Get(a.lang)

	result, err := wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.QuestionDialog,
		Title:   msgs.MSIConfirmTitle,
		Message: msgs.MSIConfirmMessage,
		Buttons: []string{"Yes", "No"},
	})
	if err != nil || result != "Yes" {
		a.log("User cancelled MSI cleanup")
		return "cancelled"
	}

	a.log(msgs.MSICleaning)
	failed := 0
	for _, code := range productCodes {
		if err := msi.CleanOrphan(code); err != nil {
			a.log(fmt.Sprintf("%s: %s - %v", msgs.MSICleanError, code, err))
			failed++
		}
	}
	if failed > 0 {
		return msgs.MSICleanError
	}
	a.log(msgs.MSICleanComplete)
	return "ok"
}

func (a *App) HandleUTF8AlreadyEnabled() string {
	msgs := i18n.Get(a.lang)

	_, _ = wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.WarningDialog,
		Title:   msgs.UTF8AlreadyEnabled,
		Message: msgs.UTF8AlreadyEnabledTip,
	})

	return ""
}

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
func (a *App) ListDrives() []string {
	return database.ListDrives()
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

// ---- USB Unlock ----

// USBUnlockAvailable checks if any removable/external drive with "Engine Library" is present.
// Returns the drive letter (e.g. "E:") if found, otherwise "".
func (a *App) USBUnlockAvailable() string {
	for _, d := range database.ListDrives() {
		root := d + `\`
		// Skip C: (system drive)
		if strings.ToUpper(d) == "C:" {
			continue
		}
		if unlock.DriveHasEngineLibrary(root) {
			return d
		}
	}
	return ""
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
func (a *App) USBUnlockKill(drive string) string {
	procs, err := unlock.FindBlockingProcesses(drive)
	if err != nil {
		a.log(fmt.Sprintf("USB unlock error: %v", err))
		return err.Error()
	}
	if len(procs) == 0 {
		a.log("No blocking processes to kill")
		return "ok"
	}

	pids := make([]uint32, len(procs))
	for i, p := range procs {
		pids[i] = p.PID
	}

	killed, errs := unlock.KillProcesses(pids)
	a.log(fmt.Sprintf("Killed %d process(es)", killed))
	for _, e := range errs {
		a.log(fmt.Sprintf("  %s", e))
	}
	return "ok"
}

// ---- MIDI 2.0 Toggle ----

// MIDI2Status returns "disabled" if MIDI 2.0 is currently disabled, "enabled" otherwise,
// or "unavailable" if MIDI 2.0 services don't exist on this system.
func (a *App) MIDI2Status() string {
	disabled, err := midi.IsMIDI2Disabled()
	if err != nil {
		return "unavailable"
	}
	if disabled {
		return "disabled"
	}
	return "enabled"
}

// MIDI2Disable sets all MIDI 2.0 service registry entries to disabled (Start=4).
// Does NOT touch the classic MIDI 1.0 service.
func (a *App) MIDI2Disable() string {
	count, err := midi.DisableMIDI2()
	if err != nil {
		a.log(fmt.Sprintf("MIDI 2.0 disable error: %v", err))
		return err.Error()
	}
	a.log(fmt.Sprintf("MIDI 2.0 disabled (%d services)", count))
	return "ok"
}

// MIDI2Enable restores all MIDI 2.0 service registry entries to demand-start (Start=3).
func (a *App) MIDI2Enable() string {
	count, err := midi.EnableMIDI2()
	if err != nil {
		a.log(fmt.Sprintf("MIDI 2.0 enable error: %v", err))
		return err.Error()
	}
	a.log(fmt.Sprintf("MIDI 2.0 re-enabled (%d services)", count))
	return "ok"
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