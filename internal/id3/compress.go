package id3

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"math"

	"github.com/bogem/id3v2/v2"
)

// CompressCoverResult summarises what happened to a single file's cover.
type CompressCoverResult struct {
	Skipped      bool  // no cover, or already small enough
	OriginalSize int64 // bytes before compression
	FinalSize    int64 // bytes after compression (0 if Skipped)
}

// CompressCover reads the front cover from filePath's ID3 tag.
// If it is larger than maxBytes, it re-encodes as JPEG at progressively
// lower quality (and scales down if necessary) until it fits.
// Returns a CompressCoverResult and any error.
func CompressCover(filePath string, maxBytes int64) (CompressCoverResult, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return CompressCoverResult{}, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer tag.Close()

	pics := tag.GetFrames(tag.CommonID("Attached picture"))
	if len(pics) == 0 {
		return CompressCoverResult{Skipped: true}, nil
	}

	// Find the front cover frame (prefer PTFrontCover, fall back to first).
	var frame id3v2.PictureFrame
	found := false
	for _, f := range pics {
		pic, ok := f.(id3v2.PictureFrame)
		if !ok {
			continue
		}
		if pic.PictureType == id3v2.PTFrontCover || !found {
			frame = pic
			found = true
		}
		if pic.PictureType == id3v2.PTFrontCover {
			break
		}
	}
	if !found {
		return CompressCoverResult{Skipped: true}, nil
	}

	origSize := int64(len(frame.Picture))
	if origSize <= maxBytes {
		return CompressCoverResult{Skipped: true, OriginalSize: origSize}, nil
	}

	// Decode the image (supports JPEG and PNG out of the box via registered decoders).
	img, _, err := image.Decode(bytes.NewReader(frame.Picture))
	if err != nil {
		return CompressCoverResult{}, fmt.Errorf("decode cover in %s: %w", filePath, err)
	}

	// Strategy 1: progressive quality reduction (no resize).
	compressed := tryJPEGQualities(img, maxBytes, []int{85, 75, 65, 55, 45, 35})

	// Strategy 2: scale down then retry.
	if compressed == nil {
		for _, maxDim := range []int{1200, 900, 600, 400} {
			scaled := scaleDownImage(img, maxDim)
			compressed = tryJPEGQualities(scaled, maxBytes, []int{80, 65, 50})
			if compressed != nil {
				break
			}
		}
	}

	if compressed == nil {
		return CompressCoverResult{}, fmt.Errorf(
			"cannot compress cover in %s below %d bytes", filePath, maxBytes)
	}

	// Write the compressed JPEG back into the tag.
	tag.DeleteFrames(tag.CommonID("Attached picture"))
	tag.AddAttachedPicture(id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    "image/jpeg",
		PictureType: id3v2.PTFrontCover,
		Description: "Cover",
		Picture:     compressed,
	})

	if err := tag.Save(); err != nil {
		return CompressCoverResult{}, fmt.Errorf("save %s: %w", filePath, err)
	}
	return CompressCoverResult{
		OriginalSize: origSize,
		FinalSize:    int64(len(compressed)),
	}, nil
}

// CoverCompressBatchResult summarises a batch compression operation.
type CoverCompressBatchResult struct {
	Total      int      `json:"total"`
	Compressed int      `json:"compressed"`
	Skipped    int      `json:"skipped"`
	Failed     int      `json:"failed"`
	SavedBytes int64    `json:"savedBytes"`
	Errors     []string `json:"errors"`
}

// tryJPEGQualities encodes img at each quality in descending order, returning
// the first result whose size is ≤ maxBytes, or nil if none qualify.
func tryJPEGQualities(img image.Image, maxBytes int64, qualities []int) []byte {
	for _, q := range qualities {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			continue
		}
		if int64(buf.Len()) <= maxBytes {
			return buf.Bytes()
		}
	}
	return nil
}

// scaleDownImage returns a copy of src scaled so that neither dimension
// exceeds maxDim, using nearest-neighbour interpolation. If src already fits,
// it is returned unchanged.
func scaleDownImage(src image.Image, maxDim int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return src
	}

	ratio := float64(maxDim) / math.Max(float64(w), float64(h))
	newW := int(math.Round(float64(w) * ratio))
	newH := int(math.Round(float64(h) * ratio))
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	// Convert to RGBA so colour operations are uniform.
	src32, ok := src.(*image.RGBA)
	if !ok {
		tmp := image.NewRGBA(b)
		draw.Draw(tmp, b, src, b.Min, draw.Src)
		src32 = tmp
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	scaleX := float64(w) / float64(newW)
	scaleY := float64(h) / float64(newH)

	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			sx := b.Min.X + int(float64(x)*scaleX)
			sy := b.Min.Y + int(float64(y)*scaleY)
			r, g, bv, a := src32.At(sx, sy).RGBA()
			dst.Set(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(bv >> 8),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}
