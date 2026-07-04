package terms

import (
	"os"
	"path/filepath"
)

// IsAccepted reports whether the user has previously accepted the terms of use.
func IsAccepted() bool {
	_, err := os.Stat(acceptedFile())
	return err == nil
}

// Accept creates the acceptance marker file.
func Accept() error {
	p := acceptedFile()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	return f.Close()
}

func acceptedFile() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "EngineTools", ".terms_accepted")
}
