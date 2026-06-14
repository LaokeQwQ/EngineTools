package msi

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

var ErrMSIZapNotFound = errors.New("msizap.exe not found")

func FindMsizap() (string, error) {
	system32 := filepath.Join(os.Getenv("SystemRoot"), "System32")
	candidates := []string{
		filepath.Join(system32, "msizap.exe"),
		filepath.Join(system32, "msizapw.exe"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	if exe, err := exec.LookPath("msizap.exe"); err == nil {
		return exe, nil
	}

	return "", ErrMSIZapNotFound
}

type OrphanedMSI struct {
	ProductCode string `json:"productCode"`
	DisplayName string `json:"displayName"`
}

func ScanOrphans() ([]OrphanedMSI, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Installer\UserData\S-1-5-18\Products`,
		registry.READ|registry.WOW64_64KEY,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open installer products: %w", err)
	}
	defer k.Close()

	subkeys, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read product subkeys: %w", err)
	}

	var orphans []OrphanedMSI
	for _, subkey := range subkeys {
		sk, err := registry.OpenKey(k, subkey, registry.READ)
		if err != nil {
			continue
		}
		displayName, _, _ := sk.GetStringValue("DisplayName")
		sk.Close()

		if displayName == "" {
			orphans = append(orphans, OrphanedMSI{ProductCode: subkey})
		} else {
			orphans = append(orphans, OrphanedMSI{
				ProductCode: subkey,
				DisplayName: displayName,
			})
		}
	}

	return orphans, nil
}

func CleanOrphan(productCode string) error {
	msizapPath, err := FindMsizap()
	if err != nil {
		return fmt.Errorf("msizap T {ProductCode} - cannot clean without msizap.exe: %w", ErrMSIZapNotFound)
	}

	productCode = strings.TrimSpace(productCode)
	cmd := exec.Command(msizapPath, "T", productCode)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("msizap cleanup failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}