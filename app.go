package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"

	"EngineTools/internal/database"
	"EngineTools/internal/i18n"
	"EngineTools/internal/manifest"
	"EngineTools/internal/msi"
	"EngineTools/internal/registry"
)

type App struct {
	ctx                context.Context
	lang               i18n.Lang
	Logs               []string
	InstallPath        string
	EngineVersion      string
	WindowsVersion     string
	UTF8Enabled        bool
	ManifestConfigured bool
	IsAdmin            bool
	StemsPath          string
	StemsDetected      bool
	DBPath             string
	DBDetected         bool
	Progress           float64

	privilege privilegeClient

	// services guards mutable selections shared across facade files.
	services struct {
		mu          sync.Mutex
		libraryPath string
	}
}

type StatusInfo struct {
	InstallPath        string        `json:"installPath"`
	EngineVersion      string        `json:"engineVersion"`
	WindowsVersion     string        `json:"windowsVersion"`
	UTF8Enabled        bool          `json:"utf8Enabled"`
	ACPValue           string        `json:"acpValue"`
	ManifestConfigured bool          `json:"manifestConfigured"`
	IsAdmin            bool          `json:"isAdmin"`
	StemsDetected      bool          `json:"stemsDetected"`
	DBDetected         bool          `json:"dbDetected"`
	DBPath             string        `json:"dbPath"`
	ProcessRunning     bool          `json:"processRunning"`
	RunningProcesses   []ProcessItem `json:"runningProcesses"`
}

type ProcessItem struct {
	Name string `json:"name"`
	PID  uint32 `json:"pid"`
}

type DriveInfo struct {
	Letter     string `json:"letter"`
	TrackCount int    `json:"trackCount"`
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
	for {
		select {
		case <-ticker.C:
			current := getLogicalDrives()
			if current != last {
				last = current
				a.detectStatus()
			}
		case <-a.ctx.Done():
			return
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
func (a *App) HandleUTF8AlreadyEnabled() string {
	msgs := i18n.Get(a.lang)

	_, _ = wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.WarningDialog,
		Title:   msgs.UTF8AlreadyEnabled,
		Message: msgs.UTF8AlreadyEnabledTip,
	})

	return ""
}

func (a *App) ID3PickDir() string {
	dir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Music Directory",
	})
	if err != nil || dir == "" {
		return ""
	}
	return dir
}

// ---- USB Unlock ----

// USBUnlockAvailable checks if any removable/external drive with "Engine Library" is present.
// Returns the drive letter (e.g. "E:") if found, otherwise "".
func (a *App) shutdown(ctx context.Context) {}

// LogExperimentalEnabled records an activation timestamp in the
// experimental_allow.xml audit file (hidden + read-only on disk).
