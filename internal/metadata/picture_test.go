package metadata

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func makeTestJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes()
}

func makeTestPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{64, 128, uint8(x % 256), 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestProbeImage(t *testing.T) {
	jpg := makeTestJPEG(100, 50)
	mime, w, h, err := ProbeImage(jpg)
	if err != nil || mime != "image/jpeg" || w != 100 || h != 50 {
		t.Fatalf("probe jpeg: mime=%q w=%d h=%d err=%v", mime, w, h, err)
	}

	pngData := makeTestPNG(200, 100)
	mime, w, h, err = ProbeImage(pngData)
	if err != nil || mime != "image/png" || w != 200 || h != 100 {
		t.Fatalf("probe png: mime=%q w=%d h=%d err=%v", mime, w, h, err)
	}

	// garbage input
	if _, _, _, perr := ProbeImage([]byte("not an image")); perr == nil {
		t.Fatal("garbage accepted")
	}
}

func TestValidatePictureForWrite(t *testing.T) {
	jpg := makeTestJPEG(64, 64)
	if _, err := ValidatePictureForWrite(jpg); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidatePictureForWrite([]byte("junk")); err == nil {
		t.Fatal("junk accepted")
	}
}

func TestNeedsConversion(t *testing.T) {
	if NeedsConversion("image/jpeg") || NeedsConversion("image/png") {
		t.Fatal("JPEG/PNG should not need conversion")
	}
	if !NeedsConversion("image/gif") {
		t.Fatal("GIF should need conversion")
	}
}

func TestConvertToJPEG(t *testing.T) {
	pngData := makeTestPNG(32, 32)
	result, err := ConvertToJPEG(pngData)
	if err != nil {
		t.Fatal(err)
	}
	mime, _, _, err := ProbeImage(result)
	if err != nil || mime != "image/jpeg" {
		t.Fatalf("converted MIME = %q err=%v", mime, err)
	}
	// JPEG input should pass through unchanged
	jpgData := makeTestJPEG(32, 32)
	passthrough, err := ConvertToJPEG(jpgData)
	if err != nil || string(passthrough) != string(jpgData) {
		t.Fatal("JPEG passthrough failed")
	}
}

func TestGenerateThumbnail(t *testing.T) {
	// large image -> shrunk to max side
	big := makeTestJPEG(2048, 1024)
	thumb, err := GenerateThumbnail(big)
	if err != nil {
		t.Fatal(err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(thumb))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width > ThumbnailMaxSide+1 || cfg.Height > ThumbnailMaxSide+1 {
		t.Fatalf("thumbnail too large: %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.Width != ThumbnailMaxSide {
		t.Fatalf("expected width %d, got %d", ThumbnailMaxSide, cfg.Width)
	}

	// small image -> passthrough re-encode
	small := makeTestJPEG(64, 32)
	smallThumb, err := GenerateThumbnail(small)
	if err != nil {
		t.Fatal(err)
	}
	if len(smallThumb) == 0 {
		t.Fatal("small thumb is empty")
	}
}

func TestValidatePictureSet(t *testing.T) {
	pics := []Picture{
		{ID: "front", Type: 3, MIME: "image/jpeg", Data: makeTestJPEG(16, 16)},
		{ID: "back", Type: 4, MIME: "image/png", Data: makeTestPNG(16, 16)},
	}
	if err := ValidatePictureSet(pics); err != nil {
		t.Fatal(err)
	}
	dups := []Picture{
		{ID: "same", Type: 3, MIME: "image/jpeg", Data: makeTestJPEG(8, 8)},
		{ID: "same", Type: 4, MIME: "image/png", Data: makeTestPNG(8, 8)},
	}
	if err := ValidatePictureSet(dups); err == nil {
		t.Fatal("duplicate IDs accepted")
	}
}
