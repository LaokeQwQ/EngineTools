package unlock

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modNtdll                    = syscall.NewLazyDLL("ntdll.dll")
	procNtQuerySystemInformation = modNtdll.NewProc("NtQuerySystemInformation")
	modKernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procQueryDosDevice          = modKernel32.NewProc("QueryDosDeviceW")
	modPsapi                    = syscall.NewLazyDLL("psapi.dll")
	procEnumProcesses           = modPsapi.NewProc("EnumProcesses")
	procGetMappedFileName       = modKernel32.NewProc("GetMappedFileNameW")
)

func enumProcesses() ([]uint32, error) {
	var pids [4096]uint32
	var cbNeeded uint32
	ret, _, _ := procEnumProcesses.Call(
		uintptr(unsafe.Pointer(&pids[0])),
		uintptr(len(pids)*4),
		uintptr(unsafe.Pointer(&cbNeeded)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("EnumProcesses failed")
	}
	count := cbNeeded / 4
	return pids[:count], nil
}

// getProcessFiles uses NtQueryInformationProcess + NtQueryObject to list
// file paths that a process has open. This is a simplified approach that
// enumerates memory-mapped files via GetMappedFileName (works for DLLs and
// memory-mapped data). For handles approach we fall back to checking the
// process exe path and its loaded modules.
func getProcessFiles(pid uint32) (exeName string, files []string) {
	h, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ,
		false, pid,
	)
	if err != nil {
		return "", nil
	}
	defer windows.CloseHandle(h)

	// Get process image name
	var buf [windows.MAX_PATH]uint16
	n := uint32(len(buf))
	err = windows.QueryFullProcessImageName(h, 0, &buf[0], &n)
	if err != nil || n == 0 {
		return "", nil
	}
	exePath := syscall.UTF16ToString(buf[:n])
	exeName = filepath.Base(exePath)
	files = append(files, exePath)

	// Also get the CWD-ish path by checking loaded modules
	var modules [1024]windows.Handle
	var cbNeeded uint32
	err = windows.EnumProcessModules(h, &modules[0], uint32(len(modules))*uint32(unsafe.Sizeof(modules[0])), &cbNeeded)
	if err != nil {
		return exeName, files
	}

	modCount := cbNeeded / uint32(unsafe.Sizeof(modules[0]))
	if modCount > uint32(len(modules)) {
		modCount = uint32(len(modules))
	}

	for i := uint32(0); i < modCount; i++ {
		var modPath [windows.MAX_PATH]uint16
		r, _, _ := procGetMappedFileName.Call(
			uintptr(h),
			uintptr(modules[i]),
			uintptr(unsafe.Pointer(&modPath[0])),
			uintptr(len(modPath)),
		)
		if r > 0 {
			p := syscall.UTF16ToString(modPath[:r])
			resolved := resolveDevicePath(p)
			if resolved != "" {
				files = append(files, resolved)
			}
		}
	}

	return exeName, files
}

// resolveDevicePath converts \Device\HarddiskVolumeX\... to a drive letter path.
func resolveDevicePath(devicePath string) string {
	for c := 'A'; c <= 'Z'; c++ {
		drive := string(c) + ":"
		drivePtr, _ := syscall.UTF16PtrFromString(drive)
		var target [windows.MAX_PATH]uint16
		r, _, _ := procQueryDosDevice.Call(
			uintptr(unsafe.Pointer(drivePtr)),
			uintptr(unsafe.Pointer(&target[0])),
			uintptr(len(target)),
		)
		if r == 0 {
			continue
		}
		deviceName := syscall.UTF16ToString(target[:])
		if strings.HasPrefix(strings.ToUpper(devicePath), strings.ToUpper(deviceName)) {
			return drive + devicePath[len(deviceName):]
		}
	}
	return ""
}
