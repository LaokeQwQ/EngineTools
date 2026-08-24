package metadata

import (
	"math"
	"testing"
)

func TestFieldPatchStates(t *testing.T) {
	if err := (FieldPatch[string]{Mode: Set, Value: ""}).Validate(); CodeOf(err) != "empty_set" {
		t.Fatal(err)
	}
	if err := (FieldPatch[string]{Mode: Clear}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (FieldPatch[string]{Mode: Unchanged, Value: "ignored"}).Validate(); CodeOf(err) != "unchanged_has_value" {
		t.Fatal(err)
	}
	if err := (FieldPatch[int]{Mode: Set, Value: 7}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (FieldPatch[float64]{Mode: Unchanged, Value: 1.5}).Validate(); CodeOf(err) != "unchanged_has_value" {
		t.Fatal(err)
	}
	if err := (FieldPatch[[]string]{Mode: Set}).Validate(); CodeOf(err) != "empty_set" {
		t.Fatal(err)
	}
	if err := (FieldPatch[string]{Mode: PatchMode("weird")}).Validate(); CodeOf(err) != "patch_mode_invalid" {
		t.Fatal(err)
	}
}

func TestTrackDiscRanges(t *testing.T) {
	ok := Patch{Track: FieldPatch[NumberPair]{Mode: Set, Value: NumberPair{Value: 3, Total: 12}}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	zero := Patch{Track: FieldPatch[NumberPair]{Mode: Set, Value: NumberPair{}}}
	if CodeOf(zero.Validate()) != "track_invalid" {
		t.Fatal(zero.Validate())
	}
	over := Patch{Track: FieldPatch[NumberPair]{Mode: Set, Value: NumberPair{Value: 13, Total: 12}}}
	if CodeOf(over.Validate()) != "track_out_of_range" {
		t.Fatal(over.Validate())
	}
	discOver := Patch{Disc: FieldPatch[NumberPair]{Mode: Set, Value: NumberPair{Value: 2, Total: 1}}}
	if CodeOf(discOver.Validate()) != "disc_out_of_range" {
		t.Fatal(discOver.Validate())
	}
}

func TestBPMValidation(t *testing.T) {
	valid := Patch{BPM: FieldPatch[float64]{Mode: Set, Value: 128.5}}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []float64{-1, 0, 501, math.Inf(1), math.NaN()} {
		p := Patch{BPM: FieldPatch[float64]{Mode: Set, Value: bad}}
		if CodeOf(p.Validate()) != "bpm_out_of_range" {
			t.Fatalf("bpm %v accepted", bad)
		}
	}
	cleared := Patch{BPM: FieldPatch[float64]{Mode: Clear}}
	if err := cleared.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRatingRange(t *testing.T) {
	if err := (Patch{Rating: FieldPatch[int]{Mode: Set, Value: 100}}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Patch{Rating: FieldPatch[int]{Mode: Set, Value: 0}}).Validate(); err != nil {
		t.Fatal(err)
	}
	if CodeOf((Patch{Rating: FieldPatch[int]{Mode: Set, Value: 101}}).Validate()) != "rating_out_of_range" {
		t.Fatal("rating 101 accepted")
	}
}

func TestISRCFormat(t *testing.T) {
	good := Patch{ISRC: FieldPatch[string]{Mode: Set, Value: "USAT29900309"}}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"usat29900309", "USAT2990030", "US-T-29900309", "12AT29900309"} {
		p := Patch{ISRC: FieldPatch[string]{Mode: Set, Value: bad}}
		if CodeOf(p.Validate()) != "isrc_invalid" {
			t.Fatalf("isrc %q accepted", bad)
		}
	}
}

func TestPictureValidation(t *testing.T) {
	valid := []Picture{
		{ID: "front", Type: 3, MIME: "image/jpeg"},
		{ID: "back", Type: 4, MIME: "image/png"},
	}
	if err := ValidatePictures(valid); err != nil {
		t.Fatal(err)
	}
	wrongType := []Picture{{ID: "x", Type: 99, MIME: "image/jpeg"}}
	if CodeOf(ValidatePictures(wrongType)) != "picture_type_invalid" {
		t.Fatal(wrongType)
	}
	wrongMIME := []Picture{{ID: "x", Type: 3, MIME: "image/webp"}}
	if CodeOf(ValidatePictures(wrongMIME)) != "picture_mime_invalid" {
		t.Fatal(wrongMIME)
	}
	dup := []Picture{
		{ID: "same", Type: 3, MIME: "image/png"},
		{ID: "same", Type: 4, MIME: "image/jpeg"},
	}
	if CodeOf(ValidatePictures(dup)) != "duplicate_picture_id" {
		t.Fatal(dup)
	}
}

func TestCapabilitiesFor(t *testing.T) {
	mp3, ok := CapabilitiesFor(FormatMP3)
	if !ok || !mp3.Writable["lyrics"] || mp3.MaxPictures != 16 {
		t.Fatalf("mp3 caps: %+v", mp3)
	}
	flac, _ := CapabilitiesFor(FormatFLAC)
	if flac.Writable["rating"] {
		t.Fatal("flac should not expose rating writes")
	}
	wav, _ := CapabilitiesFor(FormatWAV)
	if wav.Writable["bpm"] || wav.MaxPictures != 0 {
		t.Fatalf("wav caps: %+v", wav)
	}
	if _, ok := CapabilitiesFor(Format("ogg")); ok {
		t.Fatal("unknown format returned capabilities")
	}
}

func TestCodeOfFallback(t *testing.T) {
	if got := CodeOf(Ef("boom", "inner")); got != "boom" {
		t.Fatalf("code = %q", got)
	}
	if got := CodeOf(nil); got != "internal" {
		t.Fatalf("nil code = %q", got)
	}
}
