package metadata

import (
	"bytes"
	"math"
	"reflect"
)

func cloneCapabilities(c Capabilities) Capabilities {
	out := c
	out.Readable = cloneBoolMap(c.Readable)
	out.Writable = cloneBoolMap(c.Writable)
	out.PictureMIMEs = append([]string(nil), c.PictureMIMEs...)
	out.TagVersions = append([]string(nil), c.TagVersions...)
	return out
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	if in == nil {
		return nil
	}
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// patchIntent is the non-reflective view of a Patch used by capability and
// verification code. Keeping this table in one place prevents a newly added
// field from silently bypassing either check.
type patchIntent struct {
	name       string
	capability string
	mode       PatchMode
	value      any
}

func intentsForPatch(p Patch) []patchIntent {
	return []patchIntent{
		{name: "title", capability: "title", mode: p.Title.Mode, value: p.Title.Value},
		{name: "artists", capability: "artist", mode: p.Artists.Mode, value: p.Artists.Value},
		{name: "album", capability: "album", mode: p.Album.Mode, value: p.Album.Value},
		{name: "albumArtists", capability: "albumArtist", mode: p.AlbumArtists.Mode, value: p.AlbumArtists.Value},
		{name: "track", capability: "track", mode: p.Track.Mode, value: p.Track.Value},
		{name: "disc", capability: "disc", mode: p.Disc.Mode, value: p.Disc.Value},
		{name: "date", capability: "date", mode: p.Date.Mode, value: p.Date.Value},
		{name: "genres", capability: "genre", mode: p.Genres.Mode, value: p.Genres.Value},
		{name: "composers", capability: "composer", mode: p.Composers.Mode, value: p.Composers.Value},
		{name: "comment", capability: "comment", mode: p.Comment.Mode, value: p.Comment.Value},
		{name: "bpm", capability: "bpm", mode: p.BPM.Mode, value: p.BPM.Value},
		{name: "key", capability: "key", mode: p.Key.Mode, value: p.Key.Value},
		{name: "rating", capability: "rating", mode: p.Rating.Mode, value: p.Rating.Value},
		{name: "isrc", capability: "isrc", mode: p.ISRC.Mode, value: p.ISRC.Value},
		{name: "lyrics", capability: "lyrics", mode: p.Lyrics.Mode, value: p.Lyrics.Value},
		{name: "pictures", capability: "pictures", mode: p.Pictures.Mode, value: p.Pictures.Value},
	}
}

func effectiveMode(mode PatchMode) PatchMode {
	if mode == "" {
		return Unchanged
	}
	return mode
}

func patchHasChanges(p Patch) bool {
	for _, intent := range intentsForPatch(p) {
		if mode := effectiveMode(intent.mode); mode == Set || mode == Clear {
			return true
		}
	}
	return false
}

// validatePatchCapabilities rejects writes that the detected adapter cannot
// represent. It is deliberately called after Patch.Validate and before a
// recovery snapshot is created, so unsupported edits never mutate disk.
func validatePatchCapabilities(caps Capabilities, p Patch) error {
	return validatePatchCapabilitiesMode(caps, p, true)
}

// validatePatchCapabilitiesMode is shared by single-file writes (payload
// required) and batch preflight (descriptors only, payload hydrated later).
func validatePatchCapabilitiesMode(caps Capabilities, p Patch, requirePicturePayload bool) error {
	if caps.Format == "" {
		return E("format_invalid", nil)
	}
	if _, ok := CapabilitiesFor(caps.Format); !ok {
		return Ef("format_invalid", "unsupported format %q", caps.Format)
	}
	for _, intent := range intentsForPatch(p) {
		mode := effectiveMode(intent.mode)
		if mode != Set && mode != Clear {
			continue
		}
		if !capabilityWritable(caps.Writable, intent.capability) {
			return Ef("unsupported_field", "%s is not writable for %s", intent.name, caps.Format)
		}
		if intent.name == "pictures" && mode == Set {
			pics, ok := intent.value.([]Picture)
			if !ok {
				return Ef("patch_invalid", "pictures value has an invalid type")
			}
			if caps.MaxPictures >= 0 && len(pics) > caps.MaxPictures {
				return Ef("picture_limit_exceeded", "%d pictures exceeds the %d-picture limit", len(pics), caps.MaxPictures)
			}
			if err := validatePictureCapabilities(caps, pics, requirePicturePayload); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateBatchCapabilities performs the same capability gate as a write but
// accepts descriptor-only picture lists. Service.Apply performs the final
// payload check after hydrating each target file.
func validateBatchCapabilities(caps Capabilities, p Patch) error {
	return validatePatchCapabilitiesMode(caps, p, false)
}

func capabilityWritable(writable map[string]bool, name string) bool {
	if writable == nil {
		return false
	}
	if writable[name] {
		return true
	}
	// Accept plural spellings from older test/facade bridges while retaining
	// the canonical singular capability names in the format table.
	aliases := map[string][]string{
		"artist":      {"artists"},
		"albumArtist": {"albumArtists"},
		"genre":       {"genres"},
		"composer":    {"composers"},
	}
	for _, alias := range aliases[name] {
		if writable[alias] {
			return true
		}
	}
	return false
}

func validatePictureCapabilities(caps Capabilities, pics []Picture, requirePayload bool) error {
	allowed := make(map[string]bool, len(caps.PictureMIMEs))
	for _, mime := range caps.PictureMIMEs {
		allowed[mime] = true
	}
	for _, pic := range pics {
		if pic.ID == "" {
			return Ef("picture_id_invalid", "picture id must not be empty")
		}
		if requirePayload && len(pic.Data) == 0 {
			return Ef("picture_data_missing", "picture %q does not carry image data", pic.ID)
		}
		if !allowed[pic.MIME] {
			return Ef("picture_mime_invalid", "picture %q MIME %q is not supported for %s", pic.ID, pic.MIME, caps.Format)
		}
		if pic.Type < 0 || pic.Type > 20 {
			return Ef("picture_type_invalid", "picture %q type %d outside [0,20]", pic.ID, pic.Type)
		}
	}
	return nil
}

// equalFloat is intentionally strict enough to catch a changed value while
// allowing decimal text round-trips through native tag libraries.
func equalFloat(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	return a == b || math.Abs(a-b) <= 1e-9*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}

func equalStrings(a, b []string) bool {
	return reflect.DeepEqual(a, b)
}

func equalPictures(a, b []Picture) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		// Bridge read DTOs intentionally omit Data. Compare stable identity and
		// descriptors; when both sides carry bytes, compare those as well.
		if a[i].ID != b[i].ID || a[i].Type != b[i].Type || a[i].MIME != b[i].MIME ||
			a[i].Description != b[i].Description ||
			a[i].Bytes != b[i].Bytes || a[i].Width != b[i].Width || a[i].Height != b[i].Height {
			return false
		}
		if len(a[i].Data) > 0 && len(b[i].Data) > 0 && !bytes.Equal(a[i].Data, b[i].Data) {
			return false
		}
	}
	return true
}

// equalPictureContent verifies a Set operation without treating the client
// side ID as persisted media data. TagLib derives read IDs from the native
// picture order, while newly added UI pictures use a generated UUID; the
// actual safety properties are the descriptor and payload bytes.
func equalPictureContent(a, b []Picture) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type || a[i].MIME != b[i].MIME || a[i].Description != b[i].Description {
			return false
		}
		if a[i].Bytes > 0 && b[i].Bytes > 0 && a[i].Bytes != b[i].Bytes {
			return false
		}
		if a[i].Width > 0 && b[i].Width > 0 && a[i].Width != b[i].Width {
			return false
		}
		if a[i].Height > 0 && b[i].Height > 0 && a[i].Height != b[i].Height {
			return false
		}
		if len(a[i].Data) > 0 && len(b[i].Data) > 0 && !bytes.Equal(a[i].Data, b[i].Data) {
			return false
		}
	}
	return true
}
