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

	"EngineTools/internal/i18n"
	"EngineTools/internal/library"
	"EngineTools/internal/manifest"
	"EngineTools/internal/process"
	"EngineTools/internal/registry"
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

	manifestOK, _ := registry.GetPreferExternalManifest()
	if a.InstallPath != "" {
		manifestOK = manifestOK && manifest.ManifestExists(a.InstallPath)
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

	exePath := filepath.Join(a.InstallPath, "Engine DJ.exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		a.log(msgs.ExeNotFound)
		return msgs.ExeNotFound
	}

	a.log(msgs.WritingManifest)
	if err := manifest.WriteManifest(a.InstallPath); err != nil {
		a.log(fmt.Sprintf("%s: %v", msgs.ManifestWriteError, err))
		return msgs.ManifestWriteError
	}
	a.log(msgs.ManifestWritten)
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

func (a *App) SetLanguage(lang string) string {
	a.lang = i18n.Lang(lang)
	go a.detectStatus()
	return ""
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

// LibraryInfo is the JSON-serialisable version of library.DBInfo.
type LibraryInfo struct {
	Path        string `json:"path"`
	Drive       string `json:"drive"`
	UUID        string `json:"uuid"`
	TotalTracks int    `json:"totalTracks"`
	MissingRGB  int    `json:"missingRGB"`
}

// ScanLibraries scans all drives for Engine Library databases and returns status.
func (a *App) ScanLibraries() []LibraryInfo {
	dbs := library.ScanAll()
	result := make([]LibraryInfo, len(dbs))
	for i, d := range dbs {
		result[i] = LibraryInfo{
			Path:        d.Path,
			Drive:       d.Drive,
			UUID:        d.UUID,
			TotalTracks: d.TotalTracks,
			MissingRGB:  d.MissingRGB,
		}
	}
	return result
}

// RestoreOverviewFiles restores missing .rgb overview files for the given DB path.
func (a *App) RestoreOverviewFiles(dbPath string) string {
	msgs := i18n.Get(a.lang)
	a.setProgress(0)
	a.log(fmt.Sprintf("%s: %s", msgs.RestoringOverview, dbPath))

	res, err := library.Restore(dbPath, func(p float64) {
		a.setProgress(p)
	})
	if err != nil {
		a.log(fmt.Sprintf("%s: %v", msgs.RestoreError, err))
		return "error: " + err.Error()
	}

	a.log(fmt.Sprintf("%s: %s %d, %s %d",
		msgs.RestoreComplete,
		msgs.RestoreWritten, res.Written,
		msgs.RestoreSkipped, res.Skipped,
	))
	a.setProgress(1.0)

	wailsRuntime.EventsEmit(a.ctx, "statusUpdate", a.GetStatus())
	return "ok"
}

// RestoreAllLibraries scans all drives and restores missing .rgb files for every Engine Library found.
func (a *App) RestoreAllLibraries() string {
	msgs := i18n.Get(a.lang)
	a.setProgress(0)
	a.log(msgs.RestoringOverview + "...")

	dbs := library.ScanAll()
	if len(dbs) == 0 {
		a.log(msgs.InstallPathNotFound)
		return "none"
	}

	totalWritten := 0
	totalSkipped := 0
	totalErrors := 0

	for idx, d := range dbs {
		base := float64(idx) / float64(len(dbs))
		a.log(fmt.Sprintf("%s: %s (%d missing)", msgs.RestoringOverview, d.Drive, d.MissingRGB))
		res, err := library.Restore(d.Path, func(p float64) {
			a.setProgress(base + p/float64(len(dbs)))
		})
		if err != nil {
			a.log(fmt.Sprintf("%s %s: %v", msgs.RestoreError, d.Drive, err))
			totalErrors++
			continue
		}
		totalWritten += res.Written
		totalSkipped += res.Skipped
		totalErrors += res.Errors
	}

	a.log(fmt.Sprintf("%s: %s %d, %s %d",
		msgs.RestoreComplete,
		msgs.RestoreWritten, totalWritten,
		msgs.RestoreSkipped, totalSkipped,
	))
	a.setProgress(1.0)

	wailsRuntime.EventsEmit(a.ctx, "libraryRestored", nil)

	if totalErrors > 0 {
		return fmt.Sprintf("partial: %d errors", totalErrors)
	}
	return "ok"
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