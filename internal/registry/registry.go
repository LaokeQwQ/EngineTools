package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

func FindEngineDJInstallPath() (string, error) {
	path, ver, err := findInUninstall("Engine DJ")
	if err == nil && path != "" {
		engineDJVersion = ver
		return path, nil
	}

	path, ver, err = findInSoftware("Engine DJ")
	if err == nil && path != "" {
		engineDJVersion = ver
		return path, nil
	}

	commonPaths := []string{
		`C:\Program Files\Engine DJ`,
		`C:\Program Files (x86)\Engine DJ`,
		`D:\Program Files\Engine DJ`,
		`D:\Program Files (x86)\Engine DJ`,
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(filepath.Join(p, "Engine DJ.exe")); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("Engine DJ install path not found")
}

var engineDJVersion string

func GetEngineDJVersion() string {
	return engineDJVersion
}

func FindEngineDJVersionFromPath(installPath string) string {
	exePath := filepath.Join(installPath, "Engine DJ.exe")
	if _, err := os.Stat(exePath); err != nil {
		return ""
	}
	ver, err := getFileVersion(exePath)
	if err != nil {
		return ""
	}
	return ver
}

var (
	versionDLL            = syscall.NewLazyDLL("version.dll")
	procGetFileVersionInfoSizeW = versionDLL.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfoW      = versionDLL.NewProc("GetFileVersionInfoW")
	procVerQueryValueW          = versionDLL.NewProc("VerQueryValueW")
)

func getFileVersion(path string) (string, error) {
	if err := versionDLL.Load(); err != nil {
		return "", fmt.Errorf("version.dll not available: %w", err)
	}

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	size, _, _ := procGetFileVersionInfoSizeW.Call(uintptr(unsafe.Pointer(pathPtr)), 0)
	if size == 0 {
		return "", fmt.Errorf("no version info")
	}

	data := make([]byte, size)
	ret, _, _ := procGetFileVersionInfoW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		size,
		uintptr(unsafe.Pointer(&data[0])),
	)
	if ret == 0 {
		return "", fmt.Errorf("GetFileVersionInfo failed")
	}

	var bufPtr uintptr
	var bufLen uint32

	subBlock, _ := syscall.UTF16PtrFromString(`\VarFileInfo\Translation`)
	ret, _, _ = procVerQueryValueW.Call(
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(unsafe.Pointer(subBlock)),
		uintptr(unsafe.Pointer(&bufPtr)),
		uintptr(unsafe.Pointer(&bufLen)),
	)
	if ret == 0 {
		return "", fmt.Errorf("VerQueryValue Translation failed")
	}

	if bufLen < 4 {
		return "", fmt.Errorf("translation data too short")
	}

	translation := *(*uint32)(unsafe.Pointer(bufPtr))
	langID := uint16(translation & 0xFFFF)
	codePage := uint16((translation >> 16) & 0xFFFF)

	queryPath := fmt.Sprintf(`\StringFileInfo\%04x%04x\ProductVersion`, langID, codePage)
	subBlock2, _ := syscall.UTF16PtrFromString(queryPath)

	ret, _, _ = procVerQueryValueW.Call(
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(unsafe.Pointer(subBlock2)),
		uintptr(unsafe.Pointer(&bufPtr)),
		uintptr(unsafe.Pointer(&bufLen)),
	)
	if ret == 0 {
		queryPath2 := fmt.Sprintf(`\StringFileInfo\%04x%04x\FileVersion`, langID, codePage)
		subBlock3, _ := syscall.UTF16PtrFromString(queryPath2)
		ret, _, _ = procVerQueryValueW.Call(
			uintptr(unsafe.Pointer(&data[0])),
			uintptr(unsafe.Pointer(subBlock3)),
			uintptr(unsafe.Pointer(&bufPtr)),
			uintptr(unsafe.Pointer(&bufLen)),
		)
		if ret == 0 {
			return "", fmt.Errorf("VerQueryValue ProductVersion/FileVersion failed")
		}
	}

	versionStr := syscall.UTF16ToString((*[1024]uint16)(unsafe.Pointer(bufPtr))[:bufLen])
	return strings.TrimSpace(versionStr), nil
}

func findInUninstall(name string) (string, string, error) {
	paths := []registry.Key{
		registry.LOCAL_MACHINE,
		registry.CURRENT_USER,
	}

	for _, root := range paths {
		for _, wow64 := range []bool{false, true} {
			var access uint32 = registry.READ
			if wow64 {
				access |= registry.WOW64_64KEY
			}

			k, err := registry.OpenKey(
				root,
				`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
				access,
			)
			if err != nil {
				continue
			}

			subkeys, err := k.ReadSubKeyNames(-1)
			k.Close()
			if err != nil {
				continue
			}

			for _, subkey := range subkeys {
				sk, err := registry.OpenKey(
					root,
					`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\`+subkey,
					access,
				)
				if err != nil {
					continue
				}

				displayName, _, err := sk.GetStringValue("DisplayName")
				if err != nil {
					sk.Close()
					continue
				}

				if strings.Contains(strings.ToLower(displayName), strings.ToLower(name)) {
					installLocation, _, err := sk.GetStringValue("InstallLocation")
					if err != nil {
						installLocation, _, err = sk.GetStringValue("DisplayIcon")
						if err == nil && installLocation != "" {
							installLocation = filepath.Dir(installLocation)
						}
					}

					displayVersion, _, _ := sk.GetStringValue("DisplayVersion")

					sk.Close()

					if installLocation != "" {
						if _, err := os.Stat(filepath.Join(installLocation, "Engine DJ.exe")); err == nil {
							return installLocation, displayVersion, nil
						}
					}
				}
				sk.Close()
			}
		}
	}

	return "", "", fmt.Errorf("not found in uninstall keys")
}

func findInSoftware(name string) (string, string, error) {
	for _, wow64 := range []bool{false, true} {
		var access uint32 = registry.READ
		if wow64 {
			access |= registry.WOW64_64KEY
		}

		k, err := registry.OpenKey(
			registry.LOCAL_MACHINE,
			`SOFTWARE`,
			access,
		)
		if err != nil {
			continue
		}

		subkeys, err := k.ReadSubKeyNames(-1)
		k.Close()
		if err != nil {
			continue
		}

		for _, subkey := range subkeys {
			if strings.Contains(strings.ToLower(subkey), strings.ToLower(name)) {
				fullPath := `SOFTWARE\` + subkey
				sk, err := registry.OpenKey(
					registry.LOCAL_MACHINE,
					fullPath,
					access,
				)
				if err != nil {
					continue
				}

				installDir, _, err := sk.GetStringValue("InstallDir")
				if err != nil {
					installDir, _, err = sk.GetStringValue("Path")
					if err != nil {
						installDir, _, err = sk.GetStringValue("InstallPath")
					}
				}
				sk.Close()

				if installDir != "" {
					if _, err := os.Stat(filepath.Join(installDir, "Engine DJ.exe")); err == nil {
						return installDir, "", nil
					}
				}
			}
		}
	}

	return "", "", fmt.Errorf("not found in software keys")
}

func IsUTF8Enabled() (bool, string, error) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Nls\CodePage`,
		registry.READ,
	)
	if err != nil {
		return false, "", fmt.Errorf("failed to read CodePage registry: %w", err)
	}
	defer k.Close()

	acp, _, err := k.GetStringValue("ACP")
	if err != nil {
		return false, "", fmt.Errorf("failed to read ACP value: %w", err)
	}

	return acp == "65001", acp, nil
}

func SetPreferExternalManifest() error {
	k, _, err := registry.CreateKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide`,
		registry.ALL_ACCESS,
	)
	if err != nil {
		return fmt.Errorf("failed to open SideBySide registry key: %w", err)
	}
	defer k.Close()

	err = k.SetDWordValue("PreferExternalManifest", 1)
	if err != nil {
		return fmt.Errorf("failed to set PreferExternalManifest: %w", err)
	}

	return nil
}

func DeletePreferExternalManifest() error {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide`,
		registry.ALL_ACCESS,
	)
	if err != nil {
		return nil
	}
	defer k.Close()

	err = k.SetDWordValue("PreferExternalManifest", 0)
	if err != nil {
		return fmt.Errorf("failed to reset PreferExternalManifest: %w", err)
	}

	return nil
}

func GetPreferExternalManifest() (bool, error) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide`,
		registry.READ,
	)
	if err != nil {
		return false, nil
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue("PreferExternalManifest")
	if err != nil {
		return false, nil
	}

	return val == 1, nil
}

func GetWindowsVersion() string {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		registry.READ,
	)
	if err != nil {
		return "Unknown"
	}
	defer k.Close()

	productName, _, err := k.GetStringValue("ProductName")
	if err != nil {
		return "Unknown"
	}

	displayVersion, _, _ := k.GetStringValue("DisplayVersion")
	currentBuild, _, _ := k.GetStringValue("CurrentBuildNumber")

	result := productName
	if displayVersion != "" {
		result += " " + displayVersion
	}
	if currentBuild != "" {
		result += " (Build " + currentBuild + ")"
	}

	return result
}