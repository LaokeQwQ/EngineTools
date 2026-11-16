package uifix

import (
	"EngineTools/internal/storage"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"time"
)

// Bump this revision when scanner/probe rules change independently of the DLL.
const compatibilityRevision = "1"

func compatibilityValidator() string {
	digest := sha256.Sum256(proxyBinary)
	return compatibilityRevision + ":" + hex.EncodeToString(digest[:])
}

func compatibilityKey(hashes map[string]string) string {
	digest := sha256.New()
	for _, name := range proxyFiles {
		fmt.Fprintf(digest, "%s:%s\n", name, hashes[name])
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func cachedCompatibility(hashes map[string]string, records map[string]storage.CompatibilityResult) *buildProfile {
	record, ok := records[compatibilityKey(hashes)]
	if !ok || record.Validator != compatibilityValidator() || len(record.Callers) == 0 || len(record.Callers) > maxCacheCallers {
		return nil
	}
	for _, name := range proxyFiles {
		if len(hashes[name]) != 64 || record.Hashes[name] != hashes[name] {
			return nil
		}
	}
	for i, caller := range record.Callers {
		if caller < 6 || (i > 0 && caller <= record.Callers[i-1]) {
			return nil
		}
	}
	return &buildProfile{Name: "Saved verified Engine DJ/Qt build", Hashes: hashes, Callers: slices.Clone(record.Callers), Automatic: true, Cached: true}
}

func saveVerifiedCompatibility(directory string, profile *buildProfile) error {
	current, err := hashBuildFiles(directory)
	if err != nil {
		return err
	}
	for _, name := range proxyFiles {
		if current[name] != profile.Hashes[name] {
			return fmt.Errorf("%s changed during compatibility validation; retry installation", name)
		}
	}
	record := storage.CompatibilityResult{Hashes: current, Callers: slices.Clone(profile.Callers), Validator: compatibilityValidator(), CheckedAt: time.Now().Format(time.RFC3339)}
	return storage.UpdateSettings(func(s *storage.Settings) {
		if s.EngineDJCompatibility == nil {
			s.EngineDJCompatibility = make(map[string]storage.CompatibilityResult)
		}
		s.EngineDJCompatibility[compatibilityKey(current)] = record
	})
}

// Clearing saved checks never changes an installed proxy or its running cache.
func (m *Manager) ClearCompatibilityRecords(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := storage.UpdateSettings(func(s *storage.Settings) {
		if key == "" {
			s.EngineDJCompatibility = nil
		} else {
			delete(s.EngineDJCompatibility, key)
		}
	}); err != nil {
		return err
	}
	probeResults.Lock()
	defer probeResults.Unlock()
	if key == "" {
		clear(probeResults.passed)
	} else {
		delete(probeResults.passed, compatibilityValidator()+":"+key)
	}
	return nil
}
