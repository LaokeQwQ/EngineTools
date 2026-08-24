package metadata

import (
	"bytes"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
)

// Image safety budgets -- any input exceeding these is rejected before
// decode begins.
const (
	MaxImageBytes    = 20 << 20  // 20 MiB raw file size
	MaxDecodedPixels = 50 << 20  // 50 megapixels
	MaxDecodedBytes  = 256 << 20 // 256 MiB decoded RGBA estimate
	ThumbnailMaxSide = 1024
)

var allowedMIMEs = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
}
var convertibleMIMEs = map[string]bool{
	"image/gif":  true,
	"image/bmp":  true,
	"image/webp": true,
}

// ProbeImage reads only the magic bytes and header to determine MIME type,
// pixel dimensions, and whether the image fits within all safety budgets.
// It never fully decodes the image.
func ProbeImage(data []byte) (mime string, w, h int, err error) {
	if len(data) > MaxImageBytes {
		return "", 0, 0, Ef("image_too_large", "%d bytes exceeds %d MiB limit", len(data), MaxImageBytes>>20)
	}

	// Detect format from magic bytes.
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8:
		mime = "image/jpeg"
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		mime = "image/png"
	case len(data) >= 6 && (bytes.Equal(data[:6], []byte("GIF87a")) || bytes.Equal(data[:6], []byte("GIF89a"))):
		mime = "image/gif"
	case len(data) >= 2 && data[0] == 'B' && data[1] == 'M':
		mime = "image/bmp"
	default:
		return "", 0, 0, Ef("image_format_unknown", "unrecognised image magic bytes")
	}

	// Decode header only to get dimensions.
	cfg, _, derr := image.DecodeConfig(bytes.NewReader(data))
	if derr != nil {
		return "", 0, 0, Ef("image_decode_failed", "header parse: %v", derr)
	}
	w, h = cfg.Width, cfg.Height
	pixels := uint64(w) * uint64(h)
	if pixels > MaxDecodedPixels {
		return "", 0, 0, Ef("image_too_many_pixels", "%dx%d = %d MP exceeds %d MP limit", w, h, pixels/1000000, MaxDecodedPixels/1000000)
	}
	decodedEstimate := pixels * 4 // RGBA
	if decodedEstimate > MaxDecodedBytes {
		return "", 0, 0, Ef("image_decoded_too_large", "decoded estimate %d MiB exceeds %d MiB limit", decodedEstimate>>20, MaxDecodedBytes>>20)
	}
	return mime, w, h, nil
}

// ValidatePictureForWrite validates a picture before allowing it into a
// write transaction: MIME must be JPEG or PNG (or convertible), dimensions
// must be within budget, and the raw bytes must not exceed the size cap.
func ValidatePictureForWrite(data []byte) (mime string, err error) {
	mime, _, _, err = ProbeImage(data)
	if err != nil {
		return "", err
	}
	if !allowedMIMEs[mime] && !convertibleMIMEs[mime] {
		return "", Ef("picture_mime_invalid", "MIME %q is neither storable nor convertible", mime)
	}
	return mime, nil
}

// NeedsConversion reports whether the image must be transcoded to JPEG or
// PNG before TagLib can embed it.
func NeedsConversion(mime string) bool {
	return !allowedMIMEs[mime]
}

// ConvertToJPEG re-encodes a non-JPEG/PNG image as JPEG quality 85.
// Returns the converted bytes or an error if the source is already
// storable or decoding fails.
func ConvertToJPEG(data []byte) ([]byte, error) {
	mime, _, _, err := ProbeImage(data)
	if err != nil {
		return nil, err
	}
	if mime == "image/jpeg" {
		return data, nil // already fine
	}
	img, _, derr := image.Decode(bytes.NewReader(data))
	if derr != nil {
		return nil, Ef("image_convert_failed", "decode %s: %v", mime, derr)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, Ef("image_convert_failed", "JPEG encode: %v", err)
	}
	return buf.Bytes(), nil
}

// ValidatePictureSet enforces unique IDs and checks each entry.
func ValidatePictureSet(pics []Picture) error {
	seen := make(map[string]bool)
	for _, p := range pics {
		if seen[p.ID] {
			return Ef("duplicate_picture_id", "picture id %q appears twice", p.ID)
		}
		seen[p.ID] = true
		if p.Type < 0 || p.Type > 21 {
			return Ef("picture_type_invalid", "picture %q type %d outside [0,21]", p.ID, p.Type)
		}
		mime, _, _, perr := ProbeImage(p.Data)
		if perr != nil {
			return perr
		}
		if !allowedMIMEs[mime] && !convertibleMIMEs[mime] {
			return Ef("picture_mime_invalid", "picture %q has unsupported MIME %q", p.ID, mime)
		}
	}
	return nil
}
