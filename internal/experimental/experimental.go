package experimental

import (
	"EngineTools/internal/storage"
	"time"
)

// LogEnabled records activation in the shared settings file.
func LogEnabled() error {
	return storage.UpdateSettings(func(s *storage.Settings) {
		s.ExperimentalEnabled = true
		s.ExperimentalActivations = append(s.ExperimentalActivations, time.Now().Format(time.RFC3339))
	})
}
