package uifix

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"EngineTools/internal/process"
	"golang.org/x/sys/windows"
)

//go:embed THIRD_PARTY.txt
var Licenses string

type Stats struct {
	Lookups    uint64 `json:"lookups"`
	Hits       uint64 `json:"hits"`
	Misses     uint64 `json:"misses"`
	Refreshes  uint64 `json:"refreshes"`
	Errors     uint64 `json:"errors"`
	AgeMS      uint64 `json:"ageMs"`
	Interfaces uint64 `json:"interfaces"`
}

type State struct {
	Installed     bool   `json:"installed"`
	InstallStatus string `json:"installStatus"`
	InstallError  string `json:"installError"`
	ProxyVersion  string `json:"proxyVersion"`
	Running       bool   `json:"running"`
	PID           uint32 `json:"pid"`
	Active        bool   `json:"active"`
	Busy          bool   `json:"busy"`
	Phase         string `json:"phase"`
	Error         string `json:"error"`
	Stats         Stats  `json:"stats"`
}

type Manager struct {
	mu          sync.Mutex
	diagnostics map[uint32]diagnosticCursor
}

func New() *Manager { return &Manager{} }

func findEngine(directory string) (uint32, error) {
	if directory == "" {
		return 0, nil
	}
	items, err := process.FindRunningExesInDir(directory)
	if err != nil {
		return 0, err
	}
	var pid uint32
	expected := filepath.Join(directory, "Engine DJ.exe")
	for _, item := range items {
		if strings.EqualFold(item.Name, "Engine DJ.exe") {
			if !strings.EqualFold(filepath.Clean(item.ExePath), filepath.Clean(expected)) {
				return 0, errors.New("Engine DJ is running from a different installation")
			}
			if pid != 0 {
				return 0, errors.New("Multiple Engine DJ processes are running")
			}
			pid = item.PID
		}
	}
	return pid, nil
}

func (m *Manager) Status(directory string) State {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := State{Phase: "idle"}
	state.Installed, state.InstallStatus, state.InstallError = installation(directory)
	if state.Installed {
		if data, err := os.ReadFile(filepath.Join(directory, "version.dll")); err == nil && len(data) > proxyFooterSize {
			digest := sha256.Sum256(data[:len(data)-proxyFooterSize])
			state.ProxyVersion = hex.EncodeToString(digest[:6])
		}
	}
	pid, err := findEngine(directory)
	state.Running, state.PID = pid != 0, pid
	if pid != 0 {
		readStandalone(pid, &state)
	}
	if err != nil {
		state.Error = err.Error()
	}
	return state
}

var kernel = windows.NewLazySystemDLL("kernel32.dll")

func nativeError(code uint64) error {
	switch code {
	case 500:
		return errors.New("Installed cache configuration is invalid; reinstall the cache")
	case 501:
		return errors.New("Engine DJ or Qt has changed; close Engine DJ and reinstall the cache")
	case 15:
		return errors.New("Engine DJ network query call sites do not match the selected compatibility profile")
	case 302:
		return errors.New("Network snapshot expired; cache disabled and original queries restored")
	case 303:
		return errors.New("Cache refresh thread did not stop; restart Engine DJ before enabling the cache again")
	default:
		if code >= 330 && code < 350 {
			return errors.New("Cache hook removal failed; close and restart Engine DJ")
		}
		return fmt.Errorf("Could not enable the native cache (code %d)", code)
	}
}
