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
	// Thumbnails are the only picture bytes exposed to the WebView. Keep both
	// per-image and aggregate responses bounded even when a file has many
	// embedded pictures.
	ThumbnailMaxBytes      = 512 << 10
	ThumbnailMaxTotalBytes = 4 << 20
)

var allowedMIMEs = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
}

// GIF is decodable by the standard library and can be explicitly converted
// by callers that opt into that lossy transformation. BMP and WebP are not
// advertised until a decoder is linked, so we never claim to support bytes
// that this build cannot inspect and safely transcode.
var convertibleMIMEs = map[string]bool{
	"image/gif": true,
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
// write transaction: MIME must be JPEG or PNG, dimensions must be within
// budget, and the raw bytes must not exceed the size cap. Callers that accept
// GIF input must transcode it with ConvertToJPEG before this check.
func ValidatePictureForWrite(data []byte) (mime string, err error) {
	mime, _, _, err = ProbeImage(data)
	if err != nil {
		return "", err
	}
	if !allowedMIMEs[mime] {
		return "", Ef("picture_mime_invalid", "MIME %q must be converted before writing", mime)
	}
	return mime, nil
}

// NeedsConversion reports whether the image must be transcoded to JPEG or
// PNG before TagLib can embed it.
func NeedsConversion(mime string) bool {
	return !allowedMIMEs[mime]
}

// ConvertToJPEG re-encodes a decodable non-JPEG image as JPEG quality 85.
// JPEG inputs are returned unchanged. PNG is intentionally converted as well
// when a caller needs a predictable lossy output format.
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

// SanitizeDocumentForUI removes full-resolution picture payloads and replaces
// them with bounded thumbnails. The bridge/service layer still retains the
// original bytes for transactional writes; this function is only for data
// crossing the Wails/WebView boundary.
func SanitizeDocumentForUI(doc Document) Document {
	out := doc
	out.Pictures = append([]Picture(nil), doc.Pictures...)
	remaining := ThumbnailMaxTotalBytes
	for i := range out.Pictures {
		picture := out.Pictures[i]
		full := picture.Data
		picture.Data = nil
		picture.Thumbnail = nil
		picture.ThumbnailMIME = ""
		if remaining > 0 && len(full) > 0 {
			if thumb, err := GenerateThumbnail(full); err == nil && len(thumb) <= remaining {
				picture.Thumbnail = thumb
				picture.ThumbnailMIME = "image/jpeg"
				remaining -= len(thumb)
			}
		}
		out.Pictures[i] = picture
	}
	return out
}

// ValidatePictureSet enforces unique IDs, image safety budgets, and a strict
// MIME-to-bytes match for a write payload. Descriptor-only reads should use
// ValidatePictures instead; a Set patch must include every retained image's
// bytes so a full-list update cannot discard unrelated artwork.
func ValidatePictureSet(pics []Picture) error {
	if err := ValidatePictures(pics); err != nil {
		return err
	}
	for _, p := range pics {
		if len(p.Data) == 0 {
			return Ef("picture_data_missing", "picture %q does not carry image data", p.ID)
		}
		mime, _, _, perr := ProbeImage(p.Data)
		if perr != nil {
			return perr
		}
		if !allowedMIMEs[mime] || p.MIME != mime {
			return Ef("picture_mime_invalid", "picture %q MIME %q does not match %q bytes", p.ID, p.MIME, mime)
		}
	}
	return nil
}
