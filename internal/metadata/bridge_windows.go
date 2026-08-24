//go:build windows

package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// nativeBridge drives enginetools_tag.dll through its C ABI. The DLL is
// loaded lazily so a missing build degrades into a typed error instead of
// failing process startup.
type nativeBridge struct {
	dllPath string // empty = default search path
}

func NewNativeBridge(dllPath string) (Bridge, error) {
	if dllPath == "" {
		if _, err := os.Stat("enginetools_tag.dll"); err != nil {
			return nil, E("bridge_dll_missing", fmt.Errorf("enginetools_tag.dll not found beside the application"))
		}
	} else if _, err := os.Stat(dllPath); err != nil {
		return nil, E("bridge_dll_missing", err)
	}
	return nativeBridge{dllPath: dllPath}, nil
}

var (
	taglibDLL     = windows.NewLazySystemDLL("enginetools_tag.dll")
	procProbe     = taglibDLL.NewProc("et_tag_probe")
	procRead      = taglibDLL.NewProc("et_tag_read")
	procApply     = taglibDLL.NewProc("et_tag_apply")
	procPayload   = taglibDLL.NewProc("et_audio_payload_sha256")
	procBufferFre = taglibDLL.NewProc("et_buffer_free")
)

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

func freeBuf(p *etBuffer) {
	if p == nil {
		return
	}
	procBufferFre.Call(uintptr(unsafe.Pointer(p)))
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
	case "io_failed":
		return "io_failed"
	default:
		return "internal"
	}
}

func (nativeBridge) Probe(path string) (Capabilities, error) {
	if err := taglibDLL.Load(); err != nil {
		return Capabilities{}, E("bridge_dll_missing", err)
	}
	pathW, err := utf16Ptr(path)
	if err != nil {
		return Capabilities{}, err
	}
	var jsonBuf, errBuf etBuffer
	defer func() { freeBuf(&jsonBuf); freeBuf(&errBuf) }()
	status, _, _ := procProbe.Call(
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
	defer func() { freeBuf(&jsonBuf); freeBuf(&errBuf) }()
	status, _, _ := procRead.Call(
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
	defer func() { freeBuf(&jsonBuf); freeBuf(&errBuf) }()
	status, _, _ := procApply.Call(
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
	defer freeBuf(&errBuf)
	status, _, _ := procPayload.Call(
		uintptr(unsafe.Pointer(pathW)),
		uintptr(unsafe.Pointer(&out[0])),
		uintptr(unsafe.Pointer(&errBuf)))
	if status != 0 {
		return out, decodeError(errBuf)
	}
	return out, nil
}
