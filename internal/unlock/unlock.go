package unlock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// HandleInfo describes a process holding a handle on the target volume.
type HandleInfo struct {
	PID  uint32 `json:"pid"`
	Name string `json:"name"`
}

// whitelistedExes are Engine DJ processes that should NOT be killed.
var whitelistedExes = map[string]bool{
	"engine dj.exe":        true,
	"enginedj.exe":         true,
	"offlineanalyzer.exe":  true,
	"offline analyzer.exe": true,
	"stems-processor.exe":  true,
	"stem-processor.exe":   true,
}

// DriveHasEngineLibrary reports whether the given drive root contains an
// "Engine Library" folder.
func DriveHasEngineLibrary(driveRoot string) bool {
	p := filepath.Join(driveRoot, "Engine Library")
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// FindBlockingProcesses returns processes that have open handles on the given
// drive, excluding whitelisted Engine DJ processes.
func FindBlockingProcesses(driveLetter string) ([]HandleInfo, error) {
	driveLetter = strings.TrimSuffix(strings.ToUpper(driveLetter), "\\")
	if !strings.HasSuffix(driveLetter, ":") {
		driveLetter += ":"
	}
	prefix := driveLetter + "\\"

	pids, err := enumProcesses()
	if err != nil {
		return nil, err
	}

	seen := make(map[uint32]bool)
	var results []HandleInfo

	for _, pid := range pids {
		if pid == 0 || seen[pid] {
			continue
		}

		name, files := getProcessFiles(pid)
		if name == "" {
			continue
		}

		nameLower := strings.ToLower(name)
		if whitelistedExes[nameLower] {
			continue
		}

		for _, f := range files {
			if strings.HasPrefix(strings.ToUpper(f), strings.ToUpper(prefix)) {
				seen[pid] = true
				results = append(results, HandleInfo{PID: pid, Name: name})
				break
			}
		}
	}

	return results, nil
}

// KillProcesses terminates the given PIDs (skipping whitelisted ones again for safety).
func KillProcesses(pids []uint32) (killed int, errors []string) {
	for _, pid := range pids {
		h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_INFORMATION, false, pid)
		if err != nil {
			errors = append(errors, fmt.Sprintf("PID %d: open failed: %v", pid, err))
			continue
		}

		// Double-check it's not whitelisted
		var buf [windows.MAX_PATH]uint16
		n := uint32(len(buf))
		_ = windows.QueryFullProcessImageName(h, 0, &buf[0], &n)
		exeName := strings.ToLower(filepath.Base(syscall.UTF16ToString(buf[:n])))
		if whitelistedExes[exeName] {
			windows.CloseHandle(h)
			continue
		}

		err = windows.TerminateProcess(h, 1)
		windows.CloseHandle(h)
		if err != nil {
			errors = append(errors, fmt.Sprintf("PID %d (%s): terminate failed: %v", pid, exeName, err))
		} else {
			killed++
		}
	}
	return
}

// FindRemovableDriveWithLibrary scans for removable drives (USB/SD) that contain
// an Engine Library folder. Returns the first match, or empty string if none found.
func FindRemovableDriveWithLibrary() string {
	for c := 'A'; c <= 'Z'; c++ {
		drive := string(c) + ":"
		driveRoot := drive + "\\"

		// Check if it's a removable drive (USB, SD card, etc.)
		driveType := windows.GetDriveType(syscall.StringToUTF16Ptr(driveRoot))
		if driveType != windows.DRIVE_REMOVABLE {
			continue
		}

		// Check if it has Engine Library
		if DriveHasEngineLibrary(driveRoot) {
			return drive
		}
	}
	return ""
}
