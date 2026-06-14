package id3

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bogem/id3v2/v2"
)

// SetCover reads an image file and embeds it as the front cover.
func SetCover(filePath, imagePath string) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer tag.Close()

	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Errorf("failed to read image: %w", err)
	}

	mime := detectMime(imagePath)

	// Remove existing covers first
	tag.DeleteFrames(tag.CommonID("Attached picture"))

	pic := id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    mime,
		PictureType: id3v2.PTFrontCover,
		Description: "Cover",
		Picture:     imgData,
	}
	tag.AddAttachedPicture(pic)

	return tag.Save()
}

// GetCoverBase64 returns the front cover as a base64-encoded data URI, or ""
// if no cover exists.
func GetCoverBase64(filePath string) (string, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer tag.Close()

	pics := tag.GetFrames(tag.CommonID("Attached picture"))
	for _, f := range pics {
		pic, ok := f.(id3v2.PictureFrame)
		if !ok {
			continue
		}
		if pic.PictureType == id3v2.PTFrontCover || len(pics) == 1 {
			encoded := base64.StdEncoding.EncodeToString(pic.Picture)
			mime := pic.MimeType
			if mime == "" {
				mime = "image/jpeg"
			}
			return fmt.Sprintf("data:%s;base64,%s", mime, encoded), nil
		}
	}
	return "", nil
}

// ClearCover removes all attached picture frames.
func ClearCover(filePath string) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer tag.Close()

	tag.DeleteFrames(tag.CommonID("Attached picture"))
	return tag.Save()
}

// ClearAllTags removes all ID3v2 frames (title, artist, album, year, genre,
// cover, etc.) from the file — effectively resetting it to a blank tag.
func ClearAllTags(filePath string) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer tag.Close()

	tag.DeleteAllFrames()
	return tag.Save()
}

// ListAudioFiles returns MP3/FLAC/WAV/AIFF files in the given directory (non-recursive).
func ListAudioFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	exts := map[string]bool{
		".mp3": true, ".flac": true, ".wav": true, ".aiff": true, ".aif": true,
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if exts[ext] {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

func hasCoverArt(tag *id3v2.Tag) bool {
	pics := tag.GetFrames(tag.CommonID("Attached picture"))
	return len(pics) > 0
}

func detectMime(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
