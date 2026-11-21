package uifix

import (
	"EngineTools/internal/storage"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const probeArgument = "--uifix-abi-probe"

// The installed proxy also exports the isolated ABI probe. No second DLL asset
// or persistent extraction directory is needed.
func extractProbeDLL() (string, func(), error) {
	file, err := os.CreateTemp(storage.Path(), "enginetools-probe-*.dll")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.Remove(file.Name()) }
	if _, err = file.Write(proxyBinary); err != nil {
		file.Close()
		cleanup()
		return "", nil, err
	}
	if err = file.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return file.Name(), cleanup, nil
}

var probeResults = struct {
	sync.Mutex
	passed map[string]bool
}{passed: make(map[string]bool)}

// RunProbeMode is checked before window creation/elevation in the app entry
// point. Reusing the app executable avoids distributing another executable.
func RunProbeMode(args []string) bool {
	if len(args) == 0 || args[0] != probeArgument {
		return false
	}
	result := struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}{}
	var err error
	if len(args) != 3 {
		err = fmt.Errorf("invalid Qt probe arguments")
	} else {
		err = probeLocal(args[1], args[2])
	}
	result.OK = err == nil
	if err != nil {
		result.Error = err.Error()
	}
	json.NewEncoder(os.Stdout).Encode(result)
	if err != nil {
		os.Exit(1)
	}
	return true
}

func probeLocal(directory, dll string) error {
	if !filepath.IsAbs(directory) || !filepath.IsAbs(dll) {
		return fmt.Errorf("probe paths must be absolute")
	}
	data, err := os.ReadFile(dll)
	if err != nil {
		return err
	}
	if sha256.Sum256(data) != sha256.Sum256(proxyBinary) {
		return fmt.Errorf("cache probe DLL integrity check failed")
	}
	kernel.NewProc("SetErrorMode").Call(0x1 | 0x2)
	module, err := windows.LoadLibraryEx(dll, 0, windows.LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|windows.LOAD_LIBRARY_SEARCH_DEFAULT_DIRS)
	if err != nil {
		return err
	}
	defer windows.FreeLibrary(module)
	address, err := windows.GetProcAddress(module, "ProbeABI")
	if err != nil {
		return err
	}
	path, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return err
	}
	code, _, _ := syscall.SyscallN(address, uintptr(unsafe.Pointer(path)))
	if code != 0 {
		return fmt.Errorf("Qt object layout/capability validation failed (code %d)", code)
	}
	return nil
}

func checkAutomaticABI(directory, dll string, profile *buildProfile) error {
	key := compatibilityValidator() + ":" + compatibilityKey(profile.Hashes)
	probeResults.Lock()
	cached := probeResults.passed[key]
	probeResults.Unlock()
	if cached {
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	ctx, stop := context.WithTimeout(context.Background(), 15*time.Second)
	defer stop()
	command := exec.CommandContext(ctx, executable, probeArgument, directory, dll)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	output, err := command.Output()
	if ctx.Err() != nil {
		return fmt.Errorf("Qt compatibility probe cancelled or timed out")
	}
	if err != nil {
		return fmt.Errorf("Qt compatibility probe failed; Engine DJ was not modified: %v %s", err, string(output))
	}
	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if json.Unmarshal(output, &result) != nil || !result.OK {
		return fmt.Errorf("Qt compatibility probe did not confirm the object layout: %s", result.Error)
	}
	probeResults.Lock()
	probeResults.passed[key] = true
	probeResults.Unlock()
	return nil
}
