package uifix

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:embed assets/version.bin
var proxyBinary []byte

const proxyMagic = "EngineToolsPrx01"
const proxyFooterSize = 256

type proxyConfig struct {
	Hashes  [3][32]byte
	Callers []uint32
}

var proxyFiles = []string{"Engine DJ.exe", "Qt6Core.dll", "Qt6Network.dll"}

func makeProxy(profile *buildProfile) ([]byte, error) {
	if len(profile.Callers) == 0 || len(profile.Callers) > maxCacheCallers {
		return nil, errors.New("Invalid proxy call-site count")
	}
	footer := make([]byte, proxyFooterSize)
	copy(footer, proxyMagic)
	binary.LittleEndian.PutUint32(footer[16:], 1)
	binary.LittleEndian.PutUint32(footer[20:], uint32(len(profile.Callers)))
	for i, name := range proxyFiles {
		hash, err := hex.DecodeString(profile.Hashes[name])
		if err != nil || len(hash) != 32 {
			return nil, fmt.Errorf("Invalid compatibility hash: %s", name)
		}
		copy(footer[24+i*32:], hash)
	}
	for i, caller := range profile.Callers {
		binary.LittleEndian.PutUint32(footer[120+i*4:], caller)
	}
	digest := sha256.Sum256(proxyBinary)
	copy(footer[184:], digest[:])
	binary.LittleEndian.PutUint64(footer[216:], uint64(len(proxyBinary)))
	checksum := sha256.Sum256(footer[:224])
	copy(footer[224:], checksum[:])
	return append(append([]byte(nil), proxyBinary...), footer...), nil
}

// Ownership is established before either update or removal. Unknown version.dll
// files are never renamed, overwritten or deleted.
func parseProxy(data []byte) (*proxyConfig, error) {
	if len(data) <= proxyFooterSize {
		return nil, errors.New("Not an EngineTools proxy")
	}
	f := data[len(data)-proxyFooterSize:]
	count := binary.LittleEndian.Uint32(f[20:])
	if string(f[:16]) != proxyMagic || binary.LittleEndian.Uint32(f[16:]) != 1 || count == 0 || count > maxCacheCallers ||
		binary.LittleEndian.Uint64(f[216:]) != uint64(len(data)-proxyFooterSize) {
		return nil, errors.New("Unrecognized version.dll; refusing to change it")
	}
	payload, checksum := sha256.Sum256(data[:len(data)-proxyFooterSize]), sha256.Sum256(f[:224])
	if !bytes.Equal(payload[:], f[184:216]) || !bytes.Equal(checksum[:], f[224:]) {
		return nil, errors.New("EngineTools proxy integrity check failed; file left unchanged")
	}
	c := &proxyConfig{}
	for i := range c.Hashes {
		copy(c.Hashes[i][:], f[24+i*32:24+(i+1)*32])
	}
	for i := uint32(0); i < count; i++ {
		c.Callers = append(c.Callers, binary.LittleEndian.Uint32(f[120+i*4:]))
	}
	return c, nil
}

func installation(directory string) (bool, string, string) {
	if directory == "" {
		return false, "missing", ""
	}
	data, err := os.ReadFile(filepath.Join(directory, "version.dll"))
	if os.IsNotExist(err) {
		return false, "missing", ""
	}
	if err != nil {
		return false, "conflict", err.Error()
	}
	if _, err = parseProxy(data); err != nil {
		return false, "conflict", err.Error()
	}
	if !bytes.Equal(data[:len(data)-proxyFooterSize], proxyBinary) {
		return true, "outdated", ""
	}
	return true, "installed", ""
}

func (m *Manager) Install(directory string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !filepath.IsAbs(directory) {
		return errors.New("Engine DJ installation path must be absolute")
	}
	if pid, err := findEngine(directory); err != nil {
		return err
	} else if pid != 0 {
		return errors.New("Close Engine DJ before installing the cache")
	}
	target := filepath.Join(directory, "version.dll")
	previous, err := os.ReadFile(target)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if exists {
		if _, err = parseProxy(previous); err != nil {
			return err
		}
	}
	profile, err := matchBuild(directory)
	if err != nil {
		return err
	}
	if profile.Automatic && !profile.Cached {
		dll, cleanup, err := extractProbeDLL()
		if err != nil {
			return err
		}
		defer cleanup()
		if err = checkAutomaticABI(directory, dll, profile); err != nil {
			return err
		}
		if err = saveVerifiedCompatibility(directory, profile); err != nil {
			return fmt.Errorf("Save compatibility result: %w", err)
		}
	}
	data, err := makeProxy(profile)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".enginetools-proxy-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if pid, err := findEngine(directory); err != nil {
		return err
	} else if pid != 0 {
		return errors.New("Engine DJ started during installation; close it and retry")
	}
	flags := uint32(windows.MOVEFILE_WRITE_THROUGH)
	if exists {
		current, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, previous) {
			return errors.New("version.dll changed during installation; file left unchanged")
		}
		flags |= windows.MOVEFILE_REPLACE_EXISTING
	}
	from, _ := windows.UTF16PtrFromString(file.Name())
	to, _ := windows.UTF16PtrFromString(target)
	if err = windows.MoveFileEx(from, to, flags); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Uninstall(directory string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !filepath.IsAbs(directory) {
		return errors.New("Engine DJ installation path must be absolute")
	}
	if pid, err := findEngine(directory); err != nil {
		return err
	} else if pid != 0 {
		return errors.New("Close Engine DJ before uninstalling the cache")
	}
	target := filepath.Join(directory, "version.dll")
	data, err := os.ReadFile(target)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = parseProxy(data); err != nil {
		return err
	}
	if err = os.Remove(target); err != nil {
		return err
	}
	return nil
}

func readStandalone(pid uint32, state *State) bool {
	name, _ := windows.UTF16PtrFromString(fmt.Sprintf("Local\\EngineToolsCache-%d", pid))
	raw, _, _ := kernel.NewProc("OpenFileMappingW").Call(windows.FILE_MAP_READ, 0, uintptr(unsafe.Pointer(name)))
	if raw == 0 {
		return false
	}
	handle := windows.Handle(raw)
	defer windows.CloseHandle(handle)
	address, err := windows.MapViewOfFile(handle, windows.FILE_MAP_READ, 0, 0, cacheSharedWords*8)
	if err != nil {
		return false
	}
	defer windows.UnmapViewOfFile(address)
	words := (*[cacheSharedWords]uint64)(unsafe.Pointer(address))
	load := func(i int) uint64 { return atomic.LoadUint64(&words[i]) }
	if load(0) != 0x45435431 || (load(1) != cacheProtocol && load(1) != 2) || load(2) != uint64(pid) {
		return false
	}
	phase := load(4)
	state.Active, state.Busy = phase == 2, phase == 1 || phase == 5
	state.Phase = map[uint64]string{1: "starting", 2: "active", 3: "idle", 4: "error", 5: "stopping"}[phase]
	state.Error = ""
	if code := load(5); code != 0 {
		state.Error = nativeError(code).Error()
		if code == 501 {
			state.InstallStatus = "outdated"
		}
	}
	state.Stats = Stats{Lookups: load(7), Hits: load(8), Misses: load(9), Refreshes: load(10), Errors: load(11), Interfaces: load(12), AgeMS: load(6)}
	return true
}
