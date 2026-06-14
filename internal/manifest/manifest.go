package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const manifestTemplate = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly manifestVersion="1.0" xmlns="urn:schemas-microsoft-com:asm.v1">
  <assemblyIdentity type="win32" name="%s" version="1.0.0.0"/>
  <application>
    <windowsSettings>
      <activeCodePage xmlns="http://schemas.microsoft.com/SMI/2019/WindowsSettings">UTF-8</activeCodePage>
    </windowsSettings>
  </application>
</assembly>`

// ListExes returns the names of all .exe files directly inside installDir.
func ListExes(installDir string) ([]string, error) {
	entries, err := os.ReadDir(installDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read install directory: %w", err)
	}

	var exes []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".exe") {
			exes = append(exes, e.Name())
		}
	}
	return exes, nil
}

// manifestName derives the assemblyIdentity name from an exe file name.
// "Engine DJ.exe" -> "EngineDJ"
func manifestName(exeName string) string {
	base := strings.TrimSuffix(exeName, filepath.Ext(exeName))
	return strings.ReplaceAll(base, " ", "")
}

// WriteManifest writes a UTF-8 activeCodePage manifest next to every .exe
// file in installDir. It returns the number of manifests written.
func WriteManifest(installDir string) (int, error) {
	exes, err := ListExes(installDir)
	if err != nil {
		return 0, err
	}
	if len(exes) == 0 {
		return 0, fmt.Errorf("no .exe files found in %s", installDir)
	}

	written := 0
	for _, exe := range exes {
		content := fmt.Sprintf(manifestTemplate, manifestName(exe))
		manifestPath := filepath.Join(installDir, exe+".manifest")
		if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
			return written, fmt.Errorf("failed to write manifest for %s: %w", exe, err)
		}
		written++
	}

	return written, nil
}

// ManifestExists reports whether every .exe in installDir has a non-empty
// .manifest file alongside it.
func ManifestExists(installDir string) bool {
	exes, err := ListExes(installDir)
	if err != nil || len(exes) == 0 {
		return false
	}

	for _, exe := range exes {
		manifestPath := filepath.Join(installDir, exe+".manifest")
		fi, err := os.Stat(manifestPath)
		if err != nil || fi.Size() == 0 {
			return false
		}
	}
	return true
}
