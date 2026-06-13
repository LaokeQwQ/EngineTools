package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func FindEngineDJInstallPath() (string, error) {
	path, err := findInUninstall("Engine DJ")
	if err == nil && path != "" {
		return path, nil
	}

	path, err = findInSoftware("Engine DJ")
	if err == nil && path != "" {
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

func findInUninstall(name string) (string, error) {
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
					sk.Close()

					if installLocation != "" {
						if _, err := os.Stat(filepath.Join(installLocation, "Engine DJ.exe")); err == nil {
							return installLocation, nil
						}
					}
				}
				sk.Close()
			}
		}
	}

	return "", fmt.Errorf("not found in uninstall keys")
}

func findInSoftware(name string) (string, error) {
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
						return installDir, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("not found in software keys")
}

func IsUTF8Enabled() (bool, error) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Nls\CodePage`,
		registry.READ,
	)
	if err != nil {
		return false, fmt.Errorf("failed to read CodePage registry: %w", err)
	}
	defer k.Close()

	acp, _, err := k.GetStringValue("ACP")
	if err != nil {
		return false, fmt.Errorf("failed to read ACP value: %w", err)
	}

	return acp == "65001", nil
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