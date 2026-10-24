package metadata

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// goldenCases enumerates every generated fixture and its committed
// expected Document.
func goldenCases() []struct {
	name     string
	file     string
	expected string
} {
	return []struct {
		name     string
		file     string
		expected string
	}{
		{"mp3 v2.3", "mp3-v23.mp3", "mp3-v23.json"},
		{"mp3 v2.4", "mp3-v24.mp3", "mp3-v24.json"},
		{"flac", "fixture.flac", "flac.json"},
		{"wav", "fixture.wav", "wav.json"},
		{"aiff", "fixture.aiff", "aiff.json"},
	}
}

func ffmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func generateFixtures(t *testing.T, outDir string) {
	t.Helper()
	ps1, absErr := filepath.Abs(filepath.Join("tools", "generate_metadata_fixtures.ps1"))
	if absErr != nil {
		t.Fatal(absErr)
	}
	cmd := exec.Command("powershell", "-NoProfile", "-File", ps1, "-OutDir", outDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture generator failed: %v\n%s", err, out)
	}
}

// normalize drops volatile fields (warnings) so comparisons focus on the
// canonical metadata surface.
func normalize(t *testing.T, doc Document) []byte {
	t.Helper()
	doc.Warnings = nil
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func loadExpected(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", "expected", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing expected file %s: %v", name, err)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("bad expected json %s: %v", name, err)
	}
	doc.Warnings = nil
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestGoldenRead runs the full generator + bridge round trip. It requires
// Windows (for the DLL) plus ffmpeg on PATH; everywhere else it is a
// no-op skip.
func TestGoldenRead(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("golden fixtures execute on Windows where the bridge DLL lives")
	}
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not on PATH")
	}
	dll := os.Getenv("ET_TAGLIB_DLL")
	if dll == "" {
		candidate := filepath.Join("..", "..", "build", "bin", "enginetools_tag.dll")
		if _, err := os.Stat(candidate); err != nil {
			t.Skip("enginetools_tag.dll not built")
		}
		dll = candidate
	}
	bridge, err := newNativeBridgeForTest(dll)
	if err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	generateFixtures(t, outDir)

	for _, tc := range goldenCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			doc, err := bridge.Read(filepath.Join(outDir, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			got := normalize(t, doc)
			want := loadExpected(t, tc.expected)
			if !bytes.Equal(got, want) {
				t.Fatalf("mismatch:\n got %s\nwant %s", got, want)
			}
		})
	}
}

// TestID3v1VariantMatchesExpected builds an ID3v1-only file byte by byte
// (128-byte TAG trailer) and checks DetectTagVersion reports id3v1.
func TestID3v1VariantMatchesExpected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v1.mp3")
	writeMinimalMPEG(t, path, 8)

	tag := make([]byte, 128)
	copy(tag[0:], "TAG")
	copy(tag[3:], pad30("Fixture Track"))
	copy(tag[33:], pad30("Fixture Artist"))
	copy(tag[63:], pad30("Golden Album"))
	copy(tag[93:], "2026")
	tag[127] = 12 // genre placeholder
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(tag); err != nil {
		t.Fatal(err)
	}
	f.Close()

	if got := DetectTagVersion(path); got != "id3v1" {
		t.Fatalf("DetectTagVersion = %q", got)
	}
}

func pad30(s string) []byte {
	b := make([]byte, 30)
	copy(b, s)
	return b
}
