package metadata

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
)

// GenerateThumbnail creates a bounded JPEG thumbnail from image bytes.
// The longest side is capped at ThumbnailMaxSide; aspect ratio is preserved.
func GenerateThumbnail(data []byte) ([]byte, error) {
	if _, _, _, err := ProbeImage(data); err != nil {
		return nil, err
	}
	limited := io.LimitReader(bytes.NewReader(data), MaxImageBytes+1)
	img, format, derr := image.Decode(limited)
	if derr != nil {
		return nil, Ef("thumbnail_decode_failed", "decode %s: %v", format, derr)
	}
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil, E("thumbnail_decode_failed", nil)
	}

	if srcW <= ThumbnailMaxSide && srcH <= ThumbnailMaxSide {
		return encodeBoundedJPEG(img, 85)
	}

	var newW, newH int
	if srcW >= srcH {
		newW = ThumbnailMaxSide
		newH = srcH * ThumbnailMaxSide / srcW
	} else {
		newH = ThumbnailMaxSide
		newW = srcW * ThumbnailMaxSide / srcH
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		sy := bounds.Min.Y + (y * bounds.Dy() / newH)
		for x := 0; x < newW; x++ {
			sx := bounds.Min.X + (x * bounds.Dx() / newW)
			dst.Set(x, y, img.At(sx, sy))
		}
	}
	return encodeBoundedJPEG(dst, 80)
}

func encodeBoundedJPEG(img image.Image, startQuality int) ([]byte, error) {
	for quality := startQuality; quality >= 30; quality -= 10 {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, Ef("thumbnail_encode_failed", "%s", err.Error())
		}
		if buf.Len() <= ThumbnailMaxBytes {
			return buf.Bytes(), nil
		}
	}
	return nil, Ef("thumbnail_too_large", "thumbnail exceeds %d KiB limit", ThumbnailMaxBytes>>10)
}
