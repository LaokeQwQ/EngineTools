package storage

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var settingsMu sync.Mutex

// Path is anchored to the executable, never the caller's working directory.
func Path(parts ...string) string {
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Join(append([]string{filepath.Dir(exe)}, parts...)...)
}

type CompatibilityResult struct {
	Hashes    map[string]string `json:"hashes"`
	Callers   []uint32          `json:"callers"`
	Validator string            `json:"validator"`
	CheckedAt string            `json:"checkedAt"`
}

type Settings struct {
	EngineDJCompatibility   map[string]CompatibilityResult `json:"engineDJCompatibility,omitempty"`
	Language                string                         `json:"language"`
	Theme                   string                         `json:"theme"`
	SquareCorners           bool                           `json:"squareCorners"`
	AlwaysOnTop             bool                           `json:"alwaysOnTop"`
	ExperimentalEnabled     bool                           `json:"experimentalEnabled"`
	ExperimentalActivations []string                       `json:"experimentalActivations,omitempty"`
	TermsAccepted           bool                           `json:"termsAccepted"`
}

func readSettings() (Settings, error) {
	s := Settings{Theme: "orange"}
	data, err := os.ReadFile(Path("setting.json"))
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err = json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("read setting.json: %w", err)
	}
	return s, nil
}

func ReadSettings() (Settings, error) {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	return readSettings()
}

func UpdateSettings(update func(*Settings)) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	s, err := readSettings()
	if err != nil {
		return err
	}
	update(&s)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(Path(), ".setting-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), Path("setting.json"))
}

// CopyMissing preserves originals and never overwrites an existing destination.
func CopyMissing(source, destination string) error {
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("migration refuses symbolic link: %s", path)
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			return sameFile(path, target)
		}
		if err != nil {
			return err
		}
		_, err = io.Copy(output, input)
		closeErr := output.Close()
		if err != nil {
			os.Remove(target)
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		return sameFile(path, target)
	})
}

func sameFile(source, destination string) error {
	hash := func(path string) ([32]byte, error) {
		f, err := os.Open(path)
		if err != nil {
			return [32]byte{}, err
		}
		defer f.Close()
		h := sha256.New()
		_, err = io.Copy(h, f)
		var result [32]byte
		copy(result[:], h.Sum(nil))
		return result, err
	}
	a, err := hash(source)
	if err != nil {
		return err
	}
	b, err := hash(destination)
	if err != nil {
		return err
	}
	if a != b {
		return fmt.Errorf("migration conflict, original retained: %s", source)
	}
	return nil
}

// Migrate verifies every copied file before removing this specific legacy target.
func Migrate(source, destination string) error {
	if !filepath.IsAbs(source) || !filepath.IsAbs(destination) || strings.EqualFold(filepath.Clean(source), filepath.Clean(destination)) {
		return fmt.Errorf("invalid migration paths: %s -> %s", source, destination)
	}
	if _, err := os.Lstat(source); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	if err := CopyMissing(source, destination); err != nil {
		return err
	}
	return os.RemoveAll(source)
}

// Remove only the old executable-specific WebView data, not the new local profile.
func removeLegacyWebView(appData, executableName string) error {
	if !filepath.IsAbs(appData) || executableName == "" || filepath.Base(executableName) != executableName || executableName == "." || executableName == ".." {
		return fmt.Errorf("invalid legacy WebView path")
	}
	oldRoot := filepath.Join(appData, executableName)
	if err := os.RemoveAll(filepath.Join(oldRoot, "EBWebView")); err != nil {
		return err
	}
	// Remove the parent only if it is empty.
	os.Remove(oldRoot)
	return nil
}

func Prepare() error {
	for _, part := range []string{"Backup", filepath.Join("Backup", "Database"), filepath.Join("Backup", "ID3"), "WebView", "Log"} {
		if err := os.MkdirAll(Path(part), 0755); err != nil {
			return err
		}
	}
	local := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(local) {
		return fmt.Errorf("LOCALAPPDATA must be an absolute directory")
	}
	legacy := filepath.Join(local, "EngineTools")
	for source, destination := range map[string]string{"Backups": "Database", "ap_backups": "ID3"} {
		if err := Migrate(filepath.Join(legacy, source), Path("Backup", destination)); err != nil {
			return err
		}
	}
	var audit struct {
		Activations []struct {
			Timestamp string `xml:"timestamp,attr"`
		} `xml:"activation"`
	}
	auditPath := filepath.Join(legacy, "experimental_allow.xml")
	if data, err := os.ReadFile(auditPath); err == nil {
		if err = xml.Unmarshal(data, &audit); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := UpdateSettings(func(s *Settings) {
		if _, err := os.Stat(filepath.Join(legacy, ".terms_accepted")); err == nil {
			s.TermsAccepted = true
		}
		for _, activation := range audit.Activations {
			found := false
			for _, existing := range s.ExperimentalActivations {
				if existing == activation.Timestamp {
					found = true
					break
				}
			}
			if !found {
				s.ExperimentalActivations = append(s.ExperimentalActivations, activation.Timestamp)
			}
		}
	}); err != nil {
		return err
	}
	for _, name := range []string{".terms_accepted", "experimental_allow.xml"} {
		path := filepath.Join(legacy, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if err := os.Chmod(path, 0600); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := removeLegacyWebView(os.Getenv("APPDATA"), filepath.Base(exe)); err != nil {
		return err
	}
	// The retired temporary injection mode has no data worth migrating.
	if err := os.RemoveAll(filepath.Join(legacy, "engine-cache")); err != nil {
		return err
	}
	os.Remove(legacy)
	return nil
}
