package main

import (
	"context"
	"strings"

	"EngineTools/internal/apperror"
	"EngineTools/internal/operation"
	"EngineTools/internal/privilege"
)

// SystemActionResult is the structured outcome of a privileged system
// action: an operation ID for correlating progress events, how many items
// changed, and a redacted public error when the action failed.
type SystemActionResult struct {
	OperationID string                `json:"operationId"`
	Changed     int                   `json:"changed"`
	PublicError *apperror.PublicError `json:"publicError,omitempty"`
}

// privilegeClient is the seam the system facade drives; production wiring
// injects privilege.Client, tests inject fakes.
type privilegeClient interface {
	Execute(ctx context.Context, action privilege.Action, params any) (privilege.Response, error)
}

// runSystem performs one allowlisted action through the injected client
// and converts every failure into a sanitized public error.
func (a *App) runSystem(action privilege.Action, params any, changed int) SystemActionResult {
	result := SystemActionResult{OperationID: operation.NewID(), Changed: changed}
	if a.privilege == nil {
		pub := apperror.PublicFrom(nil)
		result.PublicError = &pub
		return result
	}
	resp, err := a.privilege.Execute(context.Background(), action, params)
	if err != nil {
		pub := apperror.PublicFrom(err)
		if pub.Code == "internal" && resp.Code != "" {
			pub = apperror.PublicError{Code: resp.Code, MessageKey: resp.MessageKey, OperationID: result.OperationID}
		}
		result.PublicError = &pub
		result.Changed = 0
		return result
	}
	if !resp.OK {
		pub := apperror.PublicError{
			Code:        orDefault(resp.Code, "action_failed"),
			MessageKey:  orDefault(resp.MessageKey, "errors.actionFailed"),
			OperationID: result.OperationID,
		}
		result.PublicError = &pub
		result.Changed = 0
	}
	return result
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// FixUnicodeIssues applies the per-application UTF-8 manifest through the
// elevated helper.
func (a *App) FixUnicodeIssues() SystemActionResult {
	return a.runSystem(privilege.ApplyManifest,
		privilege.InstallParams{InstallID: "engine-default"}, 1)
}

// RestoreUnicodeFix removes the manifest and registry keys again.
func (a *App) RestoreUnicodeFix() SystemActionResult {
	return a.runSystem(privilege.RemoveManifest,
		privilege.InstallParams{InstallID: "engine-default"}, 1)
}

// CleanMSIOrphans cleans each requested MSI product through the helper.
func (a *App) CleanMSIOrphans(productCodes []string) SystemActionResult {
	changed := 0
	var firstErr *apperror.PublicError
	for _, code := range productCodes {
		res := a.runSystem(privilege.CleanEngineMSI,
			privilege.ProductParams{ProductID: code}, 1)
		if res.PublicError != nil {
			if firstErr == nil {
				firstErr = res.PublicError
			}
			continue
		}
		changed += res.Changed
	}
	result := SystemActionResult{OperationID: operation.NewID(), Changed: changed}
	result.PublicError = firstErr
	return result
}

// USBUnlockKill releases the removable volume holding the Engine library.
// The drive parameter is a bare drive letter; the helper re-verifies that
// an Engine library actually lives on it before killing blockers.
func (a *App) USBUnlockKill(drive string) SystemActionResult {
	letter := strings.TrimSuffix(strings.ToUpper(strings.TrimSpace(drive)), ":")
	return a.runSystem(privilege.ReleaseVolume,
		privilege.VolumeParams{VolumeID: letter}, 1)
}
