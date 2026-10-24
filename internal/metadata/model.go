package metadata

import (
	"math"
	"regexp"
)

// PatchMode is the tri-state intent for one field.
type PatchMode string

const (
	Unchanged PatchMode = "unchanged"
	Set       PatchMode = "set"
	Clear     PatchMode = "clear"
)

// Format identifies a supported audio container.
type Format string

const (
	FormatMP3  Format = "mp3"
	FormatFLAC Format = "flac"
	FormatWAV  Format = "wav"
	FormatAIFF Format = "aiff"
)

// FieldPatch expresses one of three intents for one field. A Set with an
// empty value and an Unchanged that carries a value are both rejected.
type FieldPatch[T any] struct {
	Mode  PatchMode `json:"mode"`
	Value T         `json:"value,omitempty"`
}

// isEmpty reports the zero value for the instantiated type so Validate can
// work generically without constraints on T.
func isEmpty[T any](v T) bool {
	switch x := any(v).(type) {
	case string:
		return x == ""
	case int:
		return x == 0
	case float64:
		return x == 0
	case NumberPair:
		return x == NumberPair{}
	case []string:
		return len(x) == 0
	case []Picture:
		return len(x) == 0
	case nil:
		return true
	}
	return false
}

// Validate enforces the tri-state invariants.
func (p FieldPatch[T]) Validate() error {
	switch p.Mode {
	case Set:
		if isEmpty(p.Value) {
			return Ef("empty_set", "set patch carries an empty value")
		}
		return nil
	case Clear:
		return nil
	case "", Unchanged:
		if !isEmpty(p.Value) {
			return Ef("unchanged_has_value", "unchanged patch carries a value")
		}
		return nil
	default:
		return Ef("patch_mode_invalid", "unknown patch mode %q", string(p.Mode))
	}
}

// validateUnchangedValue applies the zero-mode compatibility rule used by
// Patch: an omitted mode is Unchanged, and therefore cannot carry a value.
// Field-specific Set validation remains below because rating=0 is valid while
// most generic zero values are not.
func validateUnchangedValue[T any](p FieldPatch[T]) error {
	if p.Mode != "" && p.Mode != Unchanged {
		return nil
	}
	if !isEmpty(p.Value) {
		return Ef("unchanged_has_value", "unchanged patch carries a value")
	}
	return nil
}

// NumberPair is a track or disc position with optional total.
type NumberPair struct {
	Value int `json:"value"`
	Total int `json:"total,omitempty"`
}

var isrcPattern = regexp.MustCompile("^[A-Z]{2}[A-Z0-9]{3}[0-9]{7}$")

// Picture is one embedded image with its stable identity.
type Picture struct {
	ID          string `json:"id"`
	Type        int    `json:"type"` // ID3v2 APIC picture type (0..20)
	MIME        string `json:"mime"`
	Description string `json:"description,omitempty"`
	// Data uses the standard JSON []byte base64 representation. Read-only
	// bridge responses may omit it, while picture replacement patches must
	// carry the complete payload so non-target images are not lost.
	Data          []byte `json:"data,omitempty"`
	Thumbnail     []byte `json:"thumbnail,omitempty"`
	ThumbnailMIME string `json:"thumbnailMime,omitempty"`
	Bytes         int64  `json:"bytes,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
}

// Warning records something the bridge preserved or had to adapt, surfaced
// to the UI without failing the write.
type Warning struct {
	Code       string `json:"code"`
	MessageKey string `json:"messageKey"`
}

// Document is the canonical readable metadata across all formats.
type Document struct {
	Format       Format        `json:"format"`
	TagVersion   string        `json:"tagVersion"`
	Title        string        `json:"title"`
	Artists      []string      `json:"artists"`
	AlbumArtists []string      `json:"albumArtists"`
	Album        string        `json:"album"`
	Track        NumberPair    `json:"track"`
	Disc         NumberPair    `json:"disc"`
	Date         string        `json:"date"`
	Genres       []string      `json:"genres"`
	Composers    []string      `json:"composers"`
	Comment      string        `json:"comment"`
	BPM          float64       `json:"bpm"`
	Key          string        `json:"key"`
	Rating       int           `json:"rating"`
	ISRC         string        `json:"isrc"`
	Lyrics       string        `json:"lyrics"`
	Pictures     []Picture     `json:"pictures"`
	Warnings     []Warning     `json:"warnings"`
	Capabilities *Capabilities `json:"capabilities,omitempty"`
}

// Patch carries the caller intent for every editable field of one file.
type Patch struct {
	Title        FieldPatch[string]     `json:"title"`
	Artists      FieldPatch[[]string]   `json:"artists"`
	Album        FieldPatch[string]     `json:"album"`
	AlbumArtists FieldPatch[[]string]   `json:"albumArtists"`
	Track        FieldPatch[NumberPair] `json:"track"`
	Disc         FieldPatch[NumberPair] `json:"disc"`
	Date         FieldPatch[string]     `json:"date"`
	Genres       FieldPatch[[]string]   `json:"genres"`
	Composers    FieldPatch[[]string]   `json:"composers"`
	Comment      FieldPatch[string]     `json:"comment"`
	BPM          FieldPatch[float64]    `json:"bpm"`
	Key          FieldPatch[string]     `json:"key"`
	Rating       FieldPatch[int]        `json:"rating"`
	ISRC         FieldPatch[string]     `json:"isrc"`
	Lyrics       FieldPatch[string]     `json:"lyrics"`
	// Wails v2 cannot resolve the nested generic FieldPatch[[]Picture] when
	// generating bindings. Keep the wire type intact but bypass its broken
	// constructor conversion; frontend callers send plain JSON objects.
	Pictures FieldPatch[[]Picture] `json:"pictures" ts_type:"any" ts_transform:"__VALUE__"`
}

// validateNumberPair enforces positive values and value<=total when a
// total is present.
func validateNumberPair(field string, p NumberPair) error {
	if p.Value < 1 {
		return Ef(field+"_invalid", "%s value must be >= 1", field)
	}
	if p.Total < 0 {
		return Ef(field+"_invalid", "%s total must be >= 0", field)
	}
	if p.Total > 0 && p.Value > p.Total {
		return Ef(field+"_out_of_range", "%s value %d exceeds total %d", field, p.Value, p.Total)
	}
	return nil
}

// validate applies tri-state invariants to every touched field plus the
// cross-field range/format rules, returning ALL violations together. An
// omitted (zero) Mode behaves exactly like Unchanged so JSON payloads can
// simply skip untouched fields.
func (p Patch) validate(requirePicturePayload bool) []error {
	var errs []error
	check := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}
	checkMode := func(field string, mode PatchMode) {
		if mode != "" && mode != Unchanged && mode != Set && mode != Clear {
			check(Ef("patch_mode_invalid", "%s has unknown patch mode %q", field, mode))
		}
	}
	touchedString := []FieldPatch[string]{p.Title, p.Album, p.Date, p.Comment, p.Key, p.Lyrics}
	for _, item := range []struct {
		name string
		mode PatchMode
	}{
		{"title", p.Title.Mode}, {"album", p.Album.Mode}, {"date", p.Date.Mode},
		{"comment", p.Comment.Mode}, {"key", p.Key.Mode}, {"lyrics", p.Lyrics.Mode},
		{"isrc", p.ISRC.Mode}, {"bpm", p.BPM.Mode}, {"rating", p.Rating.Mode},
		{"artists", p.Artists.Mode}, {"albumArtists", p.AlbumArtists.Mode},
		{"genres", p.Genres.Mode}, {"composers", p.Composers.Mode},
		{"track", p.Track.Mode}, {"disc", p.Disc.Mode}, {"pictures", p.Pictures.Mode},
	} {
		checkMode(item.name, item.mode)
	}
	for _, fe := range touchedString {
		check(validateUnchangedValue(fe))
		if fe.Mode == Unchanged || fe.Mode == "" {
			continue
		}
		check(fe.Validate())
	}
	touchedSlice := []FieldPatch[[]string]{p.Artists, p.AlbumArtists, p.Genres, p.Composers}
	for _, fe := range touchedSlice {
		check(validateUnchangedValue(fe))
		if fe.Mode == Unchanged || fe.Mode == "" {
			continue
		}
		check(fe.Validate())
	}
	pairs := []struct {
		name  string
		patch FieldPatch[NumberPair]
	}{
		{"track", p.Track},
		{"disc", p.Disc},
	}
	for _, pair := range pairs {
		check(validateUnchangedValue(pair.patch))
		if pair.patch.Mode == Unchanged || pair.patch.Mode == "" {
			continue
		}
		// clear has no value to range-check. A set with the zero pair is
		// reported as the field-specific *_invalid error below.
		if pair.patch.Mode == Clear {
			continue
		}
		if err := pair.patch.Validate(); err != nil && CodeOf(err) != "empty_set" {
			check(err)
			continue
		}
		check(validateNumberPair(pair.name, pair.patch.Value))
	}
	check(validateUnchangedValue(p.BPM))
	check(validateUnchangedValue(p.Rating))
	check(validateUnchangedValue(p.ISRC))
	if p.Pictures.Mode != Unchanged && p.Pictures.Mode != "" {
		check(validateUnchangedValue(p.Pictures))
		if err := p.Pictures.Validate(); err != nil {
			// Keep the generic tri-state error, but do not duplicate the
			// structural empty-set error for a set with no pictures.
			check(err)
		} else if p.Pictures.Mode == Set {
			// A single-file write carries the complete payload into the
			// transaction. Batch preview cannot require bytes yet because it
			// only has descriptors; those are hydrated per file by Apply.
			if requirePicturePayload {
				check(ValidatePictureSet(p.Pictures.Value))
			} else {
				check(ValidatePictures(p.Pictures.Value))
			}
		}
	}
	if p.Pictures.Mode == Unchanged || p.Pictures.Mode == "" {
		check(validateUnchangedValue(p.Pictures))
	}
	if p.BPM.Mode == Set {
		v := p.BPM.Value
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || v > 500 {
			check(Ef("bpm_out_of_range", "bpm %v outside (0,500]", v))
		}
	}
	if p.Rating.Mode == Set {
		r := p.Rating.Value
		if r < 0 || r > 100 {
			check(Ef("rating_out_of_range", "rating %d outside [0,100]", r))
		}
	}
	if p.ISRC.Mode == Set && !isrcPattern.MatchString(p.ISRC.Value) {
		check(Ef("isrc_invalid", "isrc %q does not match ISO 3901", p.ISRC.Value))
	}
	return errs
}

// Validate returns the first cross-field violation, if any.
func (p Patch) Validate() error {
	errList := p.validate(true)
	if len(errList) > 0 {
		return errList[0]
	}
	return nil
}

// ValidateForBatch checks the tri-state and descriptor shape without requiring
// picture bytes. BatchService hydrates each file's existing pictures by ID and
// then calls Service.Apply, which performs the full payload validation.
func (p Patch) ValidateForBatch() error {
	errList := p.validate(false)
	if len(errList) > 0 {
		return errList[0]
	}
	return nil
}

// Capabilities declares what one format supports so the UI can disable
// controls before the user types into them.
type Capabilities struct {
	Format       Format          `json:"format"`
	Readable     map[string]bool `json:"readable"`
	Writable     map[string]bool `json:"writable"`
	MaxPictures  int             `json:"maxPictures"`
	PictureMIMEs []string        `json:"pictureMIMEs"`
	TagVersions  []string        `json:"tagVersions"`
}

var validPictureMIMEs = []string{"image/jpeg", "image/png"}

var formatCapabilities = map[Format]Capabilities{
	FormatMP3: {Format: FormatMP3, Readable: allFields(), Writable: allFields(),
		MaxPictures: 16, PictureMIMEs: validPictureMIMEs,
		TagVersions: []string{"id3v22", "id3v23", "id3v24"}},
	FormatFLAC: {Format: FormatFLAC, Readable: allFields(),
		Writable: without("rating"), MaxPictures: 16, PictureMIMEs: validPictureMIMEs,
		TagVersions: []string{"vorbis"}},
	FormatWAV: {Format: FormatWAV,
		Readable:    only("title", "artists", "album", "track", "date", "comment", "genres"),
		Writable:    only("title", "artists", "album", "track", "date", "comment", "genres"),
		MaxPictures: 0, TagVersions: []string{"id3v23"}},
	FormatAIFF: {Format: FormatAIFF,
		Readable:    only("title", "artists", "album", "track", "date", "comment", "genres"),
		Writable:    only("title", "artists", "album", "track", "date", "comment", "genres"),
		MaxPictures: 0, TagVersions: []string{"id3v23"}},
}

func allFields() map[string]bool {
	fields := []string{"title", "artists", "album", "albumArtists", "track", "disc",
		"date", "genres", "composers", "comment", "bpm", "key", "rating", "isrc",
		"lyrics", "pictures"}
	m := make(map[string]bool, len(fields))
	for _, f := range fields {
		m[f] = true
	}
	return m
}

func without(fields ...string) map[string]bool {
	m := allFields()
	for _, f := range fields {
		delete(m, f)
	}
	return m
}

func only(fields ...string) map[string]bool {
	m := make(map[string]bool, len(fields))
	for _, f := range fields {
		m[f] = true
	}
	return m
}

// CapabilitiesFor returns the capability table for a known format.
func CapabilitiesFor(f Format) (Capabilities, bool) {
	c, ok := formatCapabilities[f]
	if !ok {
		return Capabilities{}, false
	}
	return cloneCapabilities(c), true
}

// ValidatePictures checks every embedded image: APIC type range, MIME
// whitelist, and globally unique IDs across the file.
func ValidatePictures(pics []Picture) error {
	seen := make(map[string]bool, len(pics))
	for _, pic := range pics {
		if pic.ID == "" {
			return Ef("picture_id_invalid", "picture id must not be empty")
		}
		if pic.Type < 0 || pic.Type > 20 {
			return Ef("picture_type_invalid", "picture %q type %d outside [0,20]", pic.ID, pic.Type)
		}
		ok := false
		for _, m := range validPictureMIMEs {
			if pic.MIME == m {
				ok = true
				break
			}
		}
		if !ok {
			return Ef("picture_mime_invalid", "picture %q MIME %q not allowed", pic.ID, pic.MIME)
		}
		if seen[pic.ID] {
			return Ef("duplicate_picture_id", "picture id %q appears twice", pic.ID)
		}
		seen[pic.ID] = true
	}
	return nil
}
