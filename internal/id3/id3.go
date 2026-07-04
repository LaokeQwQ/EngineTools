package id3

import (
	"fmt"

	"github.com/bogem/id3v2/v2"
)

// TagInfo holds the metadata we expose to the frontend.
type TagInfo struct {
	FilePath string `json:"filePath"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Year     string `json:"year"`
	Genre    string `json:"genre"`
	HasCover bool   `json:"hasCover"`
}

// ReadTag opens an MP3 and returns its ID3v2 tag info.
func ReadTag(filePath string) (TagInfo, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return TagInfo{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer tag.Close()

	info := TagInfo{
		FilePath: filePath,
		Title:    tag.Title(),
		Artist:   tag.Artist(),
		Album:    tag.Album(),
		Year:     tag.Year(),
		Genre:    tag.Genre(),
		HasCover: hasCoverArt(tag),
	}
	return info, nil
}

// WriteBPMKey writes BPM and/or musical key back into the file's ID3v2 tags.
// bpm should be a string like "128.00" and key a name like "Am" or "C".
// Pass empty string for either field to leave it unchanged.
func WriteBPMKey(filePath, bpm, key string) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer tag.Close()

	changed := false
	if bpm != "" {
		tag.AddTextFrame(tag.CommonID("BPM"), id3v2.EncodingUTF8, bpm)
		changed = true
	}
	if key != "" {
		// TKEY frame holds the initial key
		tag.AddTextFrame(tag.CommonID("Initial key"), id3v2.EncodingUTF8, key)
		changed = true
	}
	if !changed {
		return nil
	}
	return tag.Save()
}

// WriteTag writes the given metadata fields to the file. Empty strings are
// skipped (not cleared). Use ClearTag to remove fields.
func WriteTag(filePath, title, artist, album, year, genre string) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer tag.Close()

	if title != "" {
		tag.SetTitle(title)
	}
	if artist != "" {
		tag.SetArtist(artist)
	}
	if album != "" {
		tag.SetAlbum(album)
	}
	if year != "" {
		tag.SetYear(year)
	}
	if genre != "" {
		tag.SetGenre(genre)
	}

	return tag.Save()
}
