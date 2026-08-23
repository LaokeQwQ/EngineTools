package privilege

import (
	"fmt"
	"strings"

	"EngineTools/internal/apperror"
)

// InstallParams names a detected Engine DJ installation by its stable
// registry-derived identifier. Absolute paths are deliberately
// inexpressible: the helper resolves the install itself.
type InstallParams struct {
	InstallID string `json:"installId"`
}

// NoParams is the empty parameter set (process.stop-engine discovers and
// stops confirmed Engine processes itself; callers never name PIDs).
type NoParams struct{}

// ProductParams names an MSI product code for cleanup.
type ProductParams struct {
	ProductID string `json:"productId"`
}

// VolumeParams names a removable volume by its stable identifier.
type VolumeParams struct {
	VolumeID string `json:"volumeId"`
}

// validateIdentifier rejects empty identifiers and anything that smells
// like a path or traversal: separators, drive colons, dot segments, or
// whitespace-only padding.
func validateIdentifier(field, value string) error {
	const code = "privilege_params_invalid"
	const key = "errors.privilegeParamsInvalid"
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed != value {
		return apperror.E(code, key, false,
			fmt.Errorf("%s must be a bare identifier", field))
	}
	if strings.ContainsAny(value, `/\\:`) || strings.Contains(value, "..") {
		return apperror.E(code, key, false,
			fmt.Errorf("%s must not contain path syntax", field))
	}
	if len(value) > 128 {
		return apperror.E(code, key, false,
			fmt.Errorf("%s exceeds 128 chars", field))
	}
	return nil
}
