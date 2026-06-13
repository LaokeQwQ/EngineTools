package manifest

import (
	"fmt"
	"os"
	"path/filepath"
)

const manifestContent = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly manifestVersion="1.0" xmlns="urn:schemas-microsoft-com:asm.v1">
  <assemblyIdentity type="win32" name="EngineDJ" version="1.0.0.0"/>
  <application>
    <windowsSettings>
      <activeCodePage xmlns="http://schemas.microsoft.com/SMI/2019/WindowsSettings">UTF-8</activeCodePage>
    </windowsSettings>
  </application>
</assembly>`

func WriteManifest(installDir string) error {
	exePath := filepath.Join(installDir, "Engine DJ.exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("Engine DJ.exe not found at %s", exePath)
	}

	manifestPath := filepath.Join(installDir, "Engine DJ.exe.manifest")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	return nil
}

func ManifestExists(installDir string) bool {
	manifestPath := filepath.Join(installDir, "Engine DJ.exe.manifest")
	if _, err := os.Stat(manifestPath); err != nil {
		return false
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return false
	}

	return len(content) > 0
}