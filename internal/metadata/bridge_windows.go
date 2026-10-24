//go:build windows

package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// nativeBridge drives enginetools_tag.dll through its C ABI. The DLL is
// loaded lazily so a missing build degrades into a typed error instead of
// failing process startup.
type nativeBridge struct {
	dll       *windows.LazyDLL
	procProbe *windows.LazyProc
	procRead  *windows.LazyProc
	procApply *windows.LazyProc
	procHash  *windows.LazyProc
	procFree  *windows.LazyProc
}

func NewNativeBridge(dllPath string) (Bridge, error) {
	if dllPath == "" {
		dllPath = "enginetools_tag.dll"
	}
	if _, err := os.Stat(dllPath); err != nil {
		return nil, E("bridge_dll_missing", err)
	}
	dll := windows.NewLazyDLL(dllPath)
	// Load eagerly with the DLL's own directory on the dependency search path.
	// enginetools_tag.dll depends on tag.dll, which build-taglib.ps1 installs
	// beside it; the default LoadLibrary search would otherwise miss that pair
	// when the Go test binary lives in a temp directory.
	if dir := filepath.Dir(dllPath); dir != "" && dir != "." {
		_ = windows.SetDllDirectory(dir)
		defer windows.SetDllDirectory("")
	}
	if err := dll.Load(); err != nil {
		return nil, E("bridge_dll_missing", err)
	}
	return nativeBridge{
		dll:       dll,
		procProbe: dll.NewProc("et_tag_probe"),
		procRead:  dll.NewProc("et_tag_read"),
		procApply: dll.NewProc("et_tag_apply"),
		procHash:  dll.NewProc("et_audio_payload_sha256"),
		procFree:  dll.NewProc("et_buffer_free"),
	}, nil
}

type etBuffer struct {
	Data *byte
	Len  uintptr
}

const maxJSONBytes = 32 << 20

func (b etBuffer) bytes() ([]byte, error) {
	if b.Len == uintptr(maxJSONBytes) || b.Len > maxJSONBytes {
		return nil, E("internal", errors.New("returned JSON exceeds the safety cap"))
	}
	if b.Len == 0 || b.Data == nil {
		return nil, nil
	}
	return unsafe.Slice(b.Data, b.Len), nil
}

func freeBuf(p *etBuffer, proc *windows.LazyProc) {
	if p == nil {
		return
	}
	if proc != nil {
		proc.Call(uintptr(unsafe.Pointer(p)))
	}
}

func utf16Ptr(path string) (*uint16, error) {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, E("file_unreadable", err)
	}
	return ptr, nil
}

func decodeError(buf etBuffer) error {
	data, err := buf.bytes()
	if err != nil {
		return err
	}
	var payload struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	}
	_ = json.Unmarshal(data, &payload)
	stable := mapNativeCode(payload.Code)
	return E(stable, errors.New(payload.Detail))
}

func mapNativeCode(code string) string {
	switch code {
	case "open_failed":
		return "file_unreadable"
	case "format_failed", "unsupported_container":
		return "format_invalid"
	case "patch_invalid":
		return "patch_invalid"
	case "apply_failed":
		return "apply_failed"
	case "payload_strip_failed":
		return "io_failed"
	case "io_failed":
		return "io_failed"
	default:
		return "internal"
	}
}

func (b nativeBridge) Probe(path string) (Capabilities, error) {
	if err := b.dll.Load(); err != nil {
		return Capabilities{}, E("bridge_dll_missing", err)
	}
	pathW, err := utf16Ptr(path)
	if err != nil {
		return Capabilities{}, err
	}
	var jsonBuf, errBuf etBuffer
	defer func() { freeBuf(&jsonBuf, b.procFree); freeBuf(&errBuf, b.procFree) }()
	status, _, _ := b.procProbe.Call(
		uintptr(unsafe.Pointer(pathW)),
		uintptr(unsafe.Pointer(&jsonBuf)),
		uintptr(unsafe.Pointer(&errBuf)))
	if status != 0 {
		return Capabilities{}, decodeError(errBuf)
	}
	data, err := jsonBuf.bytes()
	if err != nil {
		return Capabilities{}, err
	}
	var probe struct {
		Format Format `json:"format"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return Capabilities{}, E("format_invalid", err)
	}
	caps, ok := CapabilitiesFor(probe.Format)
	if !ok {
		return Capabilities{}, E("format_invalid", fmt.Errorf("unknown format %q", probe.Format))
	}
	return caps, nil
}

func (b nativeBridge) Read(path string) (Document, error) {
	pathW, err := utf16Ptr(path)
	if err != nil {
		return Document{}, err
	}
	var jsonBuf, errBuf etBuffer
	defer func() { freeBuf(&jsonBuf, b.procFree); freeBuf(&errBuf, b.procFree) }()
	if err := b.dll.Load(); err != nil {
		return Document{}, E("bridge_dll_missing", err)
	}
	status, _, _ := b.procRead.Call(
		uintptr(unsafe.Pointer(pathW)),
		uintptr(unsafe.Pointer(&jsonBuf)),
		uintptr(unsafe.Pointer(&errBuf)))
	if status != 0 {
		return Document{}, decodeError(errBuf)
	}
	data, err := jsonBuf.bytes()
	if err != nil {
		return Document{}, err
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return Document{}, E("format_invalid", err)
	}
	doc.TagVersion = DetectTagVersion(path)
	return doc, nil
}

func (b nativeBridge) Apply(path string, patch Patch, _ WriteOptions) (WriteResult, error) {
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return WriteResult{}, E("patch_invalid", err)
	}
	pathW, err := utf16Ptr(path)
	if err != nil {
		return WriteResult{}, err
	}
	var jsonBuf, errBuf etBuffer
	defer func() { freeBuf(&jsonBuf, b.procFree); freeBuf(&errBuf, b.procFree) }()
	if err := b.dll.Load(); err != nil {
		return WriteResult{}, E("bridge_dll_missing", err)
	}
	status, _, _ := b.procApply.Call(
		uintptr(unsafe.Pointer(pathW)),
		uintptr(unsafe.Pointer(&patchJSON[0])),
		uintptr(len(patchJSON)),
		uintptr(unsafe.Pointer(&jsonBuf)),
		uintptr(unsafe.Pointer(&errBuf)))
	if status != 0 {
		return WriteResult{}, decodeError(errBuf)
	}
	data, err := jsonBuf.bytes()
	if err != nil {
		return WriteResult{}, err
	}
	var result WriteResult
	if err := json.Unmarshal(data, &result); err != nil {
		return WriteResult{}, E("apply_failed", err)
	}
	return result, nil
}

func (b nativeBridge) AudioPayloadSHA256(path string) ([32]byte, error) {
	var out [32]byte
	pathW, err := utf16Ptr(path)
	if err != nil {
		return out, err
	}
	var errBuf etBuffer
	defer freeBuf(&errBuf, b.procFree)
	if err := b.dll.Load(); err != nil {
		return out, E("bridge_dll_missing", err)
	}
	status, _, _ := b.procHash.Call(
		uintptr(unsafe.Pointer(pathW)),
		uintptr(unsafe.Pointer(&out[0])),
		uintptr(unsafe.Pointer(&errBuf)))
	if status != 0 {
		return out, decodeError(errBuf)
	}
	return out, nil
}
