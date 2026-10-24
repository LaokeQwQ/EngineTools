package main

import (
	"EngineTools/internal/metadata"
	"errors"
	"os"
	"path/filepath"
)

// metadataRuntime owns the process-wide metadata services used by the Wails
// facade. It is deliberately kept out of NewApp so a missing optional native
// DLL does not prevent the rest of the desktop app from starting.
type metadataRuntime struct {
	service *metadata.Service
	batch   *metadata.BatchService
	anti    *metadata.AntiPiracyService
	store   *metadata.SnapshotStore
}

func (a *App) metadataServices() (*metadataRuntime, error) {
	a.metadataMu.Lock()
	defer a.metadataMu.Unlock()
	if a.metadata != nil {
		return a.metadata, nil
	}

	dllPath := os.Getenv("ET_TAGLIB_DLL")
	if dllPath == "" {
		if exe, err := os.Executable(); err == nil {
			dllPath = filepath.Join(filepath.Dir(exe), "enginetools_tag.dll")
		}
	}
	bridge, err := metadata.NewNativeBridge(dllPath)
	if err != nil {
		return nil, err
	}

	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = filepath.Join(os.TempDir(), "EngineTools")
	}
	store := metadata.NewSnapshotStore(filepath.Join(base, "EngineTools", "metadata-recovery"))
	service := metadata.NewService(bridge, store)
	service.BridgeVersion = "taglib-2.0.2"
	batch := metadata.NewBatchService(service)
	runtime := &metadataRuntime{
		service: service,
		batch:   batch,
		store:   store,
	}
	runtime.anti = metadata.NewAntiPiracyService(batch, store, bridge)
	a.metadata = runtime
	return runtime, nil
}

func metadataUnavailable(err error) error {
	if err == nil {
		return metadata.E("metadata_unavailable", errors.New("metadata service is unavailable"))
	}
	if metadata.CodeOf(err) == "internal" {
		return metadata.E("metadata_unavailable", errors.New("metadata service is unavailable"))
	}
	return err
}

// metadataPublicError converts any internal metadata failure into a stable
// public code/message key. Causes (paths, OS errors, DLL names) must never
// reach the WebView through Wails' error serialization.
func metadataPublicError(err error) error {
	if err == nil {
		return nil
	}
	return metadata.PublicError(metadataUnavailable(err))
}
