package metadata

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"image/png"
)

// PictureCompressionResult describes a best-effort front-cover compression.
// A skipped result is still successful: the file was inspected and left
// untouched for the stated reason.
type PictureCompressionResult struct {
	Changed      bool   `json:"changed"`
	Skipped      bool   `json:"skipped"`
	Reason       string `json:"reason,omitempty"`
	OriginalSize int64  `json:"originalSize,omitempty"`
	FinalSize    int64  `json:"finalSize,omitempty"`
}

// CompressFrontPicture compresses the first front-cover picture when the
// bridge supplies its bounded bytes. Descriptor-only bridges are reported as
// skipped instead of being treated as a successful no-op. The complete
// picture set is patched so non-front pictures remain intact.
func (s *Service) CompressFrontPicture(ctx context.Context, path string, maxBytes int64) (PictureCompressionResult, error) {
	if s == nil || s.Bridge == nil {
		return PictureCompressionResult{}, E("bridge_unavailable", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return PictureCompressionResult{}, E("cancelled", err)
	}
	if maxBytes <= 0 {
		return PictureCompressionResult{}, Ef("compression_limit_invalid", "maxBytes must be positive")
	}

	doc, err := s.Read(path)
	if err != nil {
		return PictureCompressionResult{}, err
	}
	front := -1
	for i, pic := range doc.Pictures {
		if pic.Type == 3 { // ID3v2 FrontCover / FLAC front cover
			front = i
			break
		}
	}
	if front < 0 {
		return PictureCompressionResult{Skipped: true, Reason: "no_front_cover"}, nil
	}

	pic := doc.Pictures[front]
	originalSize := int64(len(pic.Data))
	if originalSize == 0 {
		originalSize = pic.Bytes
	}
	if originalSize <= 0 {
		return PictureCompressionResult{Skipped: true, Reason: "picture_size_unavailable"}, nil
	}
	if originalSize <= maxBytes {
		return PictureCompressionResult{Skipped: true, Reason: "already_within_limit", OriginalSize: originalSize, FinalSize: originalSize}, nil
	}
	if len(pic.Data) == 0 {
		return PictureCompressionResult{Skipped: true, Reason: "picture_data_unavailable", OriginalSize: originalSize, FinalSize: originalSize}, nil
	}

	compressed, mime, err := compressPictureBytes(pic.Data, maxBytes)
	if err != nil {
		return PictureCompressionResult{}, err
	}
	if int64(len(compressed)) >= originalSize {
		return PictureCompressionResult{Skipped: true, Reason: "compression_no_gain", OriginalSize: originalSize, FinalSize: originalSize}, nil
	}

	// A Set pictures patch carries the whole list. Refuse to replace a list
	// whose non-target bytes are unavailable, because doing so could delete
	// back-cover or artist images.
	next := make([]Picture, len(doc.Pictures))
	copy(next, doc.Pictures)
	for i := range next {
		if i != front && len(next[i].Data) == 0 {
			return PictureCompressionResult{Skipped: true, Reason: "picture_set_requires_full_payload", OriginalSize: originalSize, FinalSize: originalSize}, nil
		}
	}
	next[front].Data = compressed
	next[front].MIME = mime
	next[front].Bytes = int64(len(compressed))
	if err := ctx.Err(); err != nil {
		return PictureCompressionResult{}, E("cancelled", err)
	}
	if _, err := s.Apply(ctx, path, Patch{Pictures: FieldPatch[[]Picture]{Mode: Set, Value: next}}); err != nil {
		return PictureCompressionResult{}, err
	}
	return PictureCompressionResult{Changed: true, OriginalSize: originalSize, FinalSize: int64(len(compressed))}, nil
}

func compressPictureBytes(data []byte, maxBytes int64) ([]byte, string, error) {
	mime, _, _, err := ProbeImage(data)
	if err != nil {
		return nil, "", err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", Ef("image_convert_failed", "decode %s: %v", mime, err)
	}

	// JPEG is the only format with a predictable quality/size tradeoff. PNG is
	// tried once for lossless preservation, then falls back to JPEG if needed.
	if mime == "image/png" {
		var pngBuf bytes.Buffer
		if err := png.Encode(&pngBuf, img); err == nil && int64(pngBuf.Len()) <= maxBytes {
			return pngBuf.Bytes(), "image/png", nil
		}
	}
	for quality := 90; quality >= 30; quality -= 10 {
		var out bytes.Buffer
		if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, "", Ef("image_convert_failed", "JPEG encode: %v", err)
		}
		if int64(out.Len()) <= maxBytes {
			return out.Bytes(), "image/jpeg", nil
		}
	}
	return nil, "", Ef("compression_no_gain", "could not encode the picture below %d bytes", maxBytes)
}
