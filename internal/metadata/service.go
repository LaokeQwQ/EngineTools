package metadata

import (
	"errors"
)

// Read returns one canonical document together with the normalized capability
// table for its detected container. Keeping this in the service prevents UI
// callers from accidentally bypassing the capability gate used by writes.
func (s *Service) Read(path string) (Document, error) {
	if s == nil || s.Bridge == nil {
		return Document{}, E("bridge_unavailable", errors.New("metadata bridge is unavailable"))
	}
	if path == "" {
		return Document{}, E("file_unreadable", errors.New("metadata path is empty"))
	}
	doc, err := s.Bridge.Read(path)
	if err != nil {
		return Document{}, err
	}
	caps, err := s.Capabilities(path)
	if err != nil {
		return Document{}, err
	}
	if doc.Format != "" && caps.Format != "" && doc.Format != caps.Format {
		return Document{}, Ef("format_invalid", "probe/read format mismatch")
	}
	doc.Capabilities = &caps
	return doc, nil
}

// FieldMapping describes how one canonical field maps into a specific
// container's tag family.
type FieldMapping struct {
	Canonical string // Document/Patch field name
	MP3       string // ID3v2 frame or property key
	FLAC      string // Vorbis comment key
	WAV       string // RIFF INFO chunk ID or ID3v2 key (empty = not supported)
	AIFF      string // AIFF chunk name or ID3v2 key (empty = falls back)
}

// FormatFieldMappings is the authoritative table of canonical-to-native
// field names per container.
var FormatFieldMappings = map[Format][]FieldMapping{
	FormatMP3: {
		{Canonical: "title", MP3: "TIT2"},
		{Canonical: "artist", MP3: "TPE1"},
		{Canonical: "album", MP3: "TALB"},
		{Canonical: "albumArtist", MP3: "TPE2"},
		{Canonical: "track", MP3: "TRCK"},
		{Canonical: "disc", MP3: "TPOS"},
		{Canonical: "date", MP3: "TDRC"},
		{Canonical: "genre", MP3: "TCON"},
		{Canonical: "composer", MP3: "TCOM"},
		{Canonical: "comment", MP3: "COMM"},
		{Canonical: "bpm", MP3: "TBPM"},
		{Canonical: "key", MP3: "TKEY"},
		{Canonical: "isrc", MP3: "TSRC"},
		{Canonical: "lyrics", MP3: "USLT"},
	},
	FormatFLAC: {
		{Canonical: "title", FLAC: "TITLE"},
		{Canonical: "artist", FLAC: "ARTIST"},
		{Canonical: "album", FLAC: "ALBUM"},
		{Canonical: "albumArtist", FLAC: "ALBUMARTIST"},
		{Canonical: "track", FLAC: "TRACKNUMBER"},
		{Canonical: "disc", FLAC: "DISCNUMBER"},
		{Canonical: "date", FLAC: "DATE"},
		{Canonical: "genre", FLAC: "GENRE"},
		{Canonical: "composer", FLAC: "COMPOSER"},
		{Canonical: "comment", FLAC: "COMMENT"},
		{Canonical: "bpm", FLAC: "BPM"},
		{Canonical: "key", FLAC: "INITIALKEY"},
		{Canonical: "isrc", FLAC: "ISRC"},
		{Canonical: "lyrics", FLAC: "LYRICS"},
	},
	FormatWAV: {
		{Canonical: "title", WAV: "INAM"},
		{Canonical: "artist", WAV: "IART"},
		{Canonical: "album", WAV: "IPRD"},
		{Canonical: "track", WAV: "ITRK"},
		{Canonical: "date", WAV: "ICRD"},
		{Canonical: "genre", WAV: "IGNR"},
		{Canonical: "composer", WAV: "IWRI"},
		{Canonical: "comment", WAV: "ICMT"},
	},
	FormatAIFF: {
		{Canonical: "title", AIFF: "NAME"},
		{Canonical: "artist", AIFF: "AUTH"},
		{Canonical: "album", AIFF: "ALBUM"},
		{Canonical: "track", AIFF: "TRACKNUMBER"},
		{Canonical: "date", AIFF: "DATE"},
		{Canonical: "genre", AIFF: "GENRE"},
		{Canonical: "composer", AIFF: "COMPOSER"},
		{Canonical: "comment", AIFF: "ANNO"},
	},
}

// MappingFor returns the native field key for the given canonical field
// in the given format.
func MappingFor(format Format, canonical string) (string, error) {
	mappings, ok := FormatFieldMappings[format]
	if !ok {
		return "", Ef("format_invalid", "unsupported format %q", format)
	}
	for _, m := range mappings {
		if m.Canonical == canonical {
			switch format {
			case FormatMP3:
				return m.MP3, nil
			case FormatFLAC:
				return m.FLAC, nil
			case FormatWAV:
				if m.WAV != "" {
					return m.WAV, nil
				}
				return m.MP3, nil
			case FormatAIFF:
				return m.AIFF, nil
			}
		}
	}
	return "", Ef("field_unknown", "unknown canonical field %q", canonical)
}
