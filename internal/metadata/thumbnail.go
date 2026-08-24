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

	if srcW <= ThumbnailMaxSide && srcH <= ThumbnailMaxSide {
		var buf bytes.Buffer
		if jerr := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); jerr != nil {
			return nil, Ef("thumbnail_encode_failed", "%s", jerr.Error())
		}
		return buf.Bytes(), nil
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
	var buf bytes.Buffer
	if jerr := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}); jerr != nil {
		return nil, Ef("thumbnail_encode_failed", "%s", jerr.Error())
	}
	return buf.Bytes(), nil
}
