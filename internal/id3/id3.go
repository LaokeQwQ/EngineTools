// Package id3 provides transitional MP3 tag operations.
//
// Deprecated: use internal/metadata with the TagLib bridge instead.
// This package is a stub that returns "not implemented" for all operations.
package id3

import "errors"

var errNotImplemented = errors.New("id3: migrated to metadata service")

// TagInfo holds basic MP3 tag data.
type TagInfo struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Year   string `json:"year"`
	Genre  string `json:"genre"`
}

func ReadTag(filePath string) (TagInfo, error) { return TagInfo{}, errNotImplemented }
func WriteTag(filePath, title, artist, album, year, genre string) error {
	return errNotImplemented
}
func ClearAllTags(filePath string) error { return errNotImplemented }
