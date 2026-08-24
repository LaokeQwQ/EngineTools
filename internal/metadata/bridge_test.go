package metadata

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeBridge satisfies Bridge without touching any native code, letting
// the contract surface be exercised on every platform.
type fakeBridge struct {
	doc  Document
	caps Capabilities
}

func (f fakeBridge) Probe(string) (Capabilities, error) { return f.caps, nil }
func (f fakeBridge) Read(string) (Document, error)      { return f.doc, nil }
func (f fakeBridge) Apply(string, Patch, WriteOptions) (WriteResult, error) {
	return WriteResult{OK: true, Changed: 1}, nil
}
func (f fakeBridge) AudioPayloadSHA256(string) ([32]byte, error) {
	return [32]byte{1}, nil
}

var _ Bridge = fakeBridge{}

func TestFakeBridgeContract(t *testing.T) {
	f := fakeBridge{
		doc:  Document{Format: FormatFLAC, Title: "T"},
		caps: Capabilities{Format: FormatFLAC, Writable: map[string]bool{"title": true}},
	}
	caps, err := f.Probe("x.flac")
	if err != nil || caps.Format != FormatFLAC {
		t.Fatalf("probe: %+v %v", caps, err)
	}
	doc, err := f.Read("x.flac")
	if err != nil || doc.Title != "T" {
		t.Fatalf("read: %+v %v", doc, err)
	}
	res, err := f.Apply("x.flac", Patch{}, WriteOptions{})
	if err != nil || !res.OK || res.Changed != 1 {
		t.Fatalf("apply: %+v %v", res, err)
	}
}

// capabilitiesForDocument merges the detected format with its table.
func TestCapabilitiesForDocument(t *testing.T) {
	caps, ok := capabilitiesForDocument(Document{Format: FormatWAV})
	if !ok || caps.Writable["bpm"] {
		t.Fatalf("wav caps through helper: %+v", caps)
	}
}

// The native smoke path only activates when a built enginetools_tag.dll
// is available (Windows CI sets ET_TAGLIB_DLL); everywhere else it is a
// no-op so the suite stays green off-Windows.
func TestNativeBridgeSmoke(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native smoke test requires windows; portable contract tests already ran")
	}
	dll := os.Getenv("ET_TAGLIB_DLL")
	if dll == "" {
		candidate := filepath.Join("..", "build", "bin", "enginetools_tag.dll")
		if _, err := os.Stat(candidate); err != nil {
			t.Skip("enginetools_tag.dll not built; skipping native smoke test")
		}
		dll = candidate
	}
	t.Setenv("ET_TAGLIB_DLL", dll)
	b, err := newNativeBridgeForTest(dll)
	if err != nil {
		t.Fatalf("load native bridge: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "smoke.mp3")
	writeMinimalMPEG(t, path, 64)

	caps, err := b.Probe(path)
	if err != nil || caps.Format != FormatMP3 {
		t.Fatalf("probe: %+v %v", caps, err)
	}
	doc, err := b.Read(path)
	if err != nil || doc.Title != "" {
		t.Fatalf("initial read: %+v %v", doc, err)
	}

	h1, err := b.AudioPayloadSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := b.Apply(path, Patch{
		Title: FieldPatch[string]{Mode: Set, Value: "Bridge Smoke"},
	}, WriteOptions{})
	if err != nil || !res.OK || res.Changed != 1 {
		t.Fatalf("apply: %+v %v", res, err)
	}
	h2, err := b.AudioPayloadSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatal("audio payload hash changed across a metadata-only write")
	}
	doc, err = b.Read(path)
	if err != nil || doc.Title != "Bridge Smoke" {
		t.Fatalf("post-apply read: %+v %v", doc, err)
	}
}

// writeMinimalMPEG emits `frames` valid MPEG 1 Layer III frames
// (128 kbps / 44.1 kHz), each 417 bytes of filler payload.
func writeMinimalMPEG(t *testing.T, path string, frames int) {
	t.Helper()
	head := []byte{0xFF, 0xFB, 0x90, 0x00}
	var buf []byte
	for i := 0; i < frames; i++ {
		buf = append(buf, head...)
		buf = append(buf, make([]byte, 413)...)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatal(err)
	}
}
