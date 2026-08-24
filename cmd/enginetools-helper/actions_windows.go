//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"EngineTools/internal/apperror"
	"EngineTools/internal/manifest"
	"EngineTools/internal/msi"
	"EngineTools/internal/privilege"
	"EngineTools/internal/process"
	"EngineTools/internal/registry"
	"EngineTools/internal/unlock"
)

// dispatch re-discovers everything from stable identifiers before touching
// the system: the install path comes from the registry (never from the
// caller), PIDs come from scanning that install, MSI products from an
// orphan scan, and volumes are checked for a real Engine library. Caller
// input can only select among what discovery found.
func dispatch(action privilege.Action, params json.RawMessage) privilege.Response {
	fail := func(key string, err error) privilege.Response {
		_ = err // details stay in local logs; the response carries keys only
		return privilege.Response{OK: false, Code: "action_failed", MessageKey: key}
	}

	switch action {
	case privilege.ApplyManifest:
		installDir, err := registry.FindEngineDJInstallPath()
		if err != nil {
			return fail("errors.installNotFound", err)
		}
		if _, err := manifest.WriteManifest(installDir); err != nil {
			return fail("errors.manifestWrite", err)
		}
		if err := registry.SetPreferExternalManifest(); err != nil {
			return fail("errors.registryWrite", err)
		}

	case privilege.RemoveManifest:
		installDir, err := registry.FindEngineDJInstallPath()
		if err != nil {
			return fail("errors.installNotFound", err)
		}
		if _, err := manifest.DeleteManifests(installDir); err != nil {
			return fail("errors.manifestRemove", err)
		}
		if err := registry.DeletePreferExternalManifest(); err != nil {
			return fail("errors.registryWrite", err)
		}

	case privilege.StopEngine:
		// The helper discovers confirmed Engine processes itself; callers
		// cannot name PIDs.
		installDir, err := registry.FindEngineDJInstallPath()
		if err != nil {
			return fail("errors.installNotFound", err)
		}
		procs, err := process.FindRunningExesInDir(installDir)
		if err != nil {
			return fail("errors.processScan", err)
		}
		for _, p := range procs {
			if err := process.KillProcess(p.PID); err != nil {
				return fail("errors.processStop", err)
			}
		}

	case privilege.CleanEngineMSI:
		var p privilege.ProductParams
		if err := strictParams(params, &p); err != nil {
			return fail("errors.privilegeParamsInvalid", err)
		}
		orphans, err := msi.ScanOrphans()
		if err != nil {
			return fail("errors.msiScan", err)
		}
		found := false
		for _, orphan := range orphans {
			if strings.EqualFold(orphan.ProductCode, p.ProductID) {
				found = true
				break
			}
		}
		if !found {
			return fail("errors.productNotFound", fmt.Errorf("product %q not among scanned orphans", p.ProductID))
		}
		if err := msi.CleanOrphan(p.ProductID); err != nil {
			return fail("errors.msiClean", err)
		}

	case privilege.ReleaseVolume:
		var p privilege.VolumeParams
		if err := strictParams(params, &p); err != nil {
			return fail("errors.privilegeParamsInvalid", err)
		}
		letter := strings.ToUpper(strings.TrimSpace(p.VolumeID))
		if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z' {
			return fail("errors.privilegeParamsInvalid", fmt.Errorf("volumeId %q is not a drive letter", p.VolumeID))
		}
		root := letter + `:\`
		if !unlock.DriveHasEngineLibrary(root) {
			return fail("errors.noLibraryOnVolume", fmt.Errorf("no Engine library on %s", root))
		}
		blockers, err := unlock.FindBlockingProcesses(letter)
		if err != nil {
			return fail("errors.handleScan", err)
		}
		pids := make([]uint32, 0, len(blockers))
		for _, b := range blockers {
			pids = append(pids, b.PID)
		}
		killed, errs := unlock.KillProcesses(pids)
		if killed == 0 && len(errs) > 0 {
			return fail("errors.volumeRelease", fmt.Errorf("first error: %s", errs[0]))
		}

	default:
		// ValidateRequest already rejects unknown actions; this is defense
		// in depth.
		return fail("errors.privilegeActionDenied", fmt.Errorf("unhandled action %q", action))
	}

	return privilege.Response{OK: true, MessageKey: "actions.done"}
}

func strictParams(raw json.RawMessage, dst any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apperror.E("privilege_params_invalid", "errors.privilegeParamsInvalid", false, err)
	}
	return nil
}
