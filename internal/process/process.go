package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

type ProcessInfo struct {
	Name   string `json:"name"`
	PID    uint32 `json:"pid"`
	ExePath string `json:"exePath"`
}

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess  = modkernel32.NewProc("OpenProcess")
	procTerminateProc = modkernel32.NewProc("TerminateProcess")
	procCloseHandle  = modkernel32.NewProc("CloseHandle")
	modpsapi         = syscall.NewLazyDLL("psapi.dll")
	procEnumProcs    = modpsapi.NewProc("EnumProcesses")
	procEnumProcMods = modpsapi.NewProc("EnumProcessModulesEx")
	procGetModBase   = modpsapi.NewProc("GetModuleBaseNameW")
	procGetModFile   = modpsapi.NewProc("GetModuleFileNameExW")
)

func FindRunningExesInDir(dir string) ([]ProcessInfo, error) {
	var running []ProcessInfo

	exes, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var exeNames []string
	for _, f := range exes {
		if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".exe") {
			exeNames = append(exeNames, f.Name())
		}
	}

	pids, err := enumProcesses()
	if err != nil {
		return nil, err
	}

	dirLower := strings.ToLower(dir)

	for _, pid := range pids {
		procInfo, err := getProcessInfo(pid)
		if err != nil {
			continue
		}

		if procInfo.ExePath != "" && strings.HasPrefix(strings.ToLower(procInfo.ExePath), dirLower) {
			for _, exe := range exeNames {
				if strings.EqualFold(filepath.Base(procInfo.ExePath), exe) {
					running = append(running, ProcessInfo{
						Name:    exe,
						PID:     pid,
						ExePath: procInfo.ExePath,
					})
					break
				}
			}
		} else if procInfo.Name != "" {
			for _, exe := range exeNames {
				if strings.EqualFold(procInfo.Name, exe) {
					running = append(running, ProcessInfo{
						Name:    exe,
						PID:     pid,
						ExePath: procInfo.ExePath,
					})
					break
				}
			}
		}
	}

	return running, nil
}

func KillProcess(pid uint32) error {
	handle, _, err := procOpenProcess.Call(
		uintptr(0x0001|0x0008),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return fmt.Errorf("failed to open process %d: %w", pid, err)
	}
	defer procCloseHandle.Call(handle)

	ret, _, err := procTerminateProc.Call(handle, 1)
	if ret == 0 {
		return fmt.Errorf("failed to terminate process %d: %w", pid, err)
	}

	return nil
}

func RefreshSystemSettings() error {
	user32 := syscall.NewLazyDLL("user32.dll")
	sendMessage := user32.NewProc("SendMessageW")
	sendMessageTimeout := user32.NewProc("SendMessageTimeoutW")

	HWND_BROADCAST := uintptr(0xFFFF)
	WM_SETTINGCHANGE := uintptr(0x001A)

	sendMessageTimeout.Call(
		HWND_BROADCAST,
		WM_SETTINGCHANGE,
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Environment"))),
		0x0002,
		5000,
		0,
	)

	sendMessage.Call(
		HWND_BROADCAST,
		WM_SETTINGCHANGE,
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("intl"))),
	)

	return nil
}

type processDetail struct {
	Name    string
	ExePath string
}

func getProcessInfo(pid uint32) (processDetail, error) {
	var detail processDetail

	handle, _, _ := procOpenProcess.Call(
		uintptr(0x0400|0x0010),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return detail, fmt.Errorf("cannot open process")
	}
	defer procCloseHandle.Call(handle)

	var hModule [1024]syscall.Handle
	var cbNeeded uint32

	ret, _, _ := procEnumProcMods.Call(
		handle,
		uintptr(unsafe.Pointer(&hModule[0])),
		unsafe.Sizeof(hModule),
		uintptr(unsafe.Pointer(&cbNeeded)),
		uintptr(0x03),
	)
	if ret == 0 {
		return detail, fmt.Errorf("cannot enum modules")
	}

	var nameBuf [syscall.MAX_PATH]uint16
	ret, _, _ = procGetModBase.Call(
		handle,
		uintptr(hModule[0]),
		uintptr(unsafe.Pointer(&nameBuf[0])),
		syscall.MAX_PATH,
	)
	if ret != 0 {
		detail.Name = syscall.UTF16ToString(nameBuf[:])
	}

	var pathBuf [syscall.MAX_PATH]uint16
	ret, _, _ = procGetModFile.Call(
		handle,
		uintptr(hModule[0]),
		uintptr(unsafe.Pointer(&pathBuf[0])),
		syscall.MAX_PATH,
	)
	if ret != 0 {
		detail.ExePath = syscall.UTF16ToString(pathBuf[:])
	}

	return detail, nil
}

func enumProcesses() ([]uint32, error) {
	var pids [4096]uint32
	var cbNeeded uint32

	ret, _, err := procEnumProcs.Call(
		uintptr(unsafe.Pointer(&pids[0])),
		unsafe.Sizeof(pids),
		uintptr(unsafe.Pointer(&cbNeeded)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("EnumProcesses failed: %w", err)
	}

	count := cbNeeded / 4
	result := make([]uint32, count)
	copy(result, pids[:count])
	return result, nil
}

func OpenControlPanel() error {
	return exec.Command("cmd", "/c", "start", "control", "intl.cpl").Start()
}