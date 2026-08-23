package registry

import "fmt"

// Windows product types from OSVERSIONINFOEXW.
const (
	verNTWorkstation      byte = 1
	verNTDomainController byte = 2
	verNTServer           byte = 3
)

// classifyWindowsVersion converts the kernel version and product type into a
// stable Windows family label. Windows 11 intentionally retains the 10.0
// kernel version, so the client build number is the family boundary.
func classifyWindowsVersion(major, minor, build uint32, productType byte) string {
	if major == 0 {
		return "Unknown"
	}

	if major == 10 && minor == 0 {
		switch productType {
		case verNTDomainController, verNTServer:
			return "Windows Server"
		case verNTWorkstation:
			if build >= 22000 {
				return "Windows 11"
			}
			return "Windows 10"
		default:
			return fmt.Sprintf("Windows %d.%d", major, minor)
		}
	}

	return fmt.Sprintf("Windows %d.%d", major, minor)
}

func formatWindowsVersion(family, displayVersion string, build uint32, ubr uint64) string {
	result := family
	if displayVersion != "" {
		result += " " + displayVersion
	}
	if build != 0 {
		if ubr != 0 {
			return fmt.Sprintf("%s (Build %d.%d)", result, build, ubr)
		}
		return fmt.Sprintf("%s (Build %d)", result, build)
	}
	return result
}
