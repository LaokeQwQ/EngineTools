package metadata

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
		{Canonical: "title", MP3: "TIT2", FLAC: "TITLE", WAV: "INAM", AIFF: "NAME"},
		{Canonical: "artist", MP3: "TPE1", FLAC: "ARTIST", WAV: "IART", AIFF: "AUTH"},
		{Canonical: "album", MP3: "TALB", FLAC: "ALBUM", WAV: "IPRD", AIFF: "ALBUM"},
		{Canonical: "albumArtist", MP3: "TPE2", FLAC: "ALBUMARTIST", WAV: "", AIFF: ""},
		{Canonical: "track", MP3: "TRCK", FLAC: "TRACKNUMBER", WAV: "ITRK", AIFF: "TRACKNUMBER"},
		{Canonical: "disc", MP3: "TPOS", FLAC: "DISCNUMBER", WAV: "", AIFF: ""},
		{Canonical: "date", MP3: "TDRC", FLAC: "DATE", WAV: "ICRD", AIFF: "DATE"},
		{Canonical: "genre", MP3: "TCON", FLAC: "GENRE", WAV: "IGNR", AIFF: "GENRE"},
		{Canonical: "composer", MP3: "TCOM", FLAC: "COMPOSER", WAV: "IWRI", AIFF: "COMPOSER"},
		{Canonical: "comment", MP3: "COMM", FLAC: "COMMENT", WAV: "ICMT", AIFF: "ANNO"},
		{Canonical: "bpm", MP3: "TBPM", FLAC: "BPM", WAV: "", AIFF: ""},
		{Canonical: "key", MP3: "TKEY", FLAC: "INITIALKEY", WAV: "", AIFF: ""},
		{Canonical: "isrc", MP3: "TSRC", FLAC: "ISRC", WAV: "", AIFF: ""},
		{Canonical: "lyrics", MP3: "USLT", FLAC: "LYRICS", WAV: "", AIFF: ""},
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
