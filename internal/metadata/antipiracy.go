package metadata

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"math/big"
	"strings"
)

// AntiPiracyService provides safe V1 (clear) and V2 (derange) operations
// over multiple audio files using full-file snapshots.
type AntiPiracyService struct {
	Batch  *BatchService
	Store  *SnapshotStore
	Bridge Bridge
}

type AntiPiracyResult struct {
	OperationID string      `json:"operationId"`
	Total       int         `json:"total"`
	Updated     int         `json:"updated"`
	Failed      int         `json:"failed"`
	Blocked     int         `json:"blocked"`
	Items       []BatchItem `json:"items"`
}

func NewAntiPiracyService(batch *BatchService, store *SnapshotStore, bridge Bridge) *AntiPiracyService {
	return &AntiPiracyService{Batch: batch, Store: store, Bridge: bridge}
}

// antiTextFieldNames is the canonical set of non-numeric fields V1 and V2
// operate on. A caller may pass a narrower subset; it must not silently move
// pictures, rating, BPM, or unknown container data.
var antiTextFieldNames = []string{
	"title", "artists", "album", "albumArtists", "date", "genres",
	"composers", "comment", "key", "isrc", "lyrics",
}

func antiFieldCapability(name string) (string, bool) {
	switch name {
	case "title", "album", "date", "comment", "key", "isrc", "lyrics":
		return name, true
	case "artists":
		return "artist", true
	case "albumArtists":
		return "albumArtist", true
	case "genres":
		return "genre", true
	case "composers":
		return "composer", true
	default:
		return "", false
	}
}

func resolveAntiFields(fields []string) ([]string, error) {
	if len(fields) == 0 {
		return append([]string(nil), antiTextFieldNames...), nil
	}
	seen := make(map[string]bool, len(fields))
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if _, ok := antiFieldCapability(field); !ok {
			return nil, Ef("antipiracy_field_invalid", "unsupported anti-piracy field %q", field)
		}
		if !seen[field] {
			seen[field] = true
			out = append(out, field)
		}
	}
	if len(out) == 0 {
		return nil, Ef("antipiracy_field_invalid", "no anti-piracy fields selected")
	}
	return out, nil
}

func clearAntiPatch(fields []string) Patch {
	var patch Patch
	for _, field := range fields {
		switch field {
		case "title":
			patch.Title = FieldPatch[string]{Mode: Clear}
		case "artists":
			patch.Artists = FieldPatch[[]string]{Mode: Clear}
		case "album":
			patch.Album = FieldPatch[string]{Mode: Clear}
		case "albumArtists":
			patch.AlbumArtists = FieldPatch[[]string]{Mode: Clear}
		case "date":
			patch.Date = FieldPatch[string]{Mode: Clear}
		case "genres":
			patch.Genres = FieldPatch[[]string]{Mode: Clear}
		case "composers":
			patch.Composers = FieldPatch[[]string]{Mode: Clear}
		case "comment":
			patch.Comment = FieldPatch[string]{Mode: Clear}
		case "key":
			patch.Key = FieldPatch[string]{Mode: Clear}
		case "isrc":
			patch.ISRC = FieldPatch[string]{Mode: Clear}
		case "lyrics":
			patch.Lyrics = FieldPatch[string]{Mode: Clear}
		}
	}
	return patch
}

func setAntiPatch(fields []string, source Document) Patch {
	var patch Patch
	for _, field := range fields {
		switch field {
		case "title":
			patch.Title = stringPatch(source.Title)
		case "artists":
			patch.Artists = listPatch(source.Artists)
		case "album":
			patch.Album = stringPatch(source.Album)
		case "albumArtists":
			patch.AlbumArtists = listPatch(source.AlbumArtists)
		case "date":
			patch.Date = stringPatch(source.Date)
		case "genres":
			patch.Genres = listPatch(source.Genres)
		case "composers":
			patch.Composers = listPatch(source.Composers)
		case "comment":
			patch.Comment = stringPatch(source.Comment)
		case "key":
			patch.Key = stringPatch(source.Key)
		case "isrc":
			patch.ISRC = stringPatch(source.ISRC)
		case "lyrics":
			patch.Lyrics = stringPatch(source.Lyrics)
		}
	}
	return patch
}

func stringPatch(value string) FieldPatch[string] {
	if value == "" {
		return FieldPatch[string]{Mode: Clear}
	}
	return FieldPatch[string]{Mode: Set, Value: value}
}

func listPatch(value []string) FieldPatch[[]string] {
	if len(value) == 0 {
		return FieldPatch[[]string]{Mode: Clear}
	}
	return FieldPatch[[]string]{Mode: Set, Value: append([]string(nil), value...)}
}

func fieldsForDocument(doc Document, selected []string) []string {
	if doc.Capabilities == nil {
		return nil
	}
	out := make([]string, 0, len(selected))
	for _, field := range selected {
		capability, ok := antiFieldCapability(field)
		if ok && capabilityWritable(doc.Capabilities.Writable, capability) {
			out = append(out, field)
		}
	}
	return out
}

func (s *AntiPiracyService) ensureReady() error {
	if s == nil || s.Batch == nil || s.Batch.Service == nil || s.Batch.Service.Bridge == nil {
		return E("bridge_unavailable", errors.New("metadata bridge is unavailable"))
	}
	return nil
}

// ClearV1 removes selected text fields across multiple files. Every included
// file is read and snapshotted before the first write; fields unsupported by
// one container are filtered per file so mixed MP3/FLAC/WAV/AIFF batches work.
func (s *AntiPiracyService) ClearV1(ctx context.Context, paths []string, fields ...string) (*AntiPiracyResult, error) {
	if len(paths) < 2 {
		return nil, Ef("antipiracy_too_few", "V1 clear requires at least 2 files")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	selected, err := resolveAntiFields(fields)
	if err != nil {
		return nil, err
	}
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	result := &AntiPiracyResult{OperationID: randHex(16), Total: len(paths)}
	type clearFile struct {
		path   string
		fields []string
	}
	eligible := make([]clearFile, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return result, E("cancelled", err)
		}
		if path == "" {
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: "file_unreadable", MessageKey: "errors.file_unreadable"})
			continue
		}
		if _, exists := seen[path]; exists {
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: "antipiracy_duplicate_path", MessageKey: "errors.antipiracy_duplicate_path"})
			continue
		}
		seen[path] = struct{}{}
		doc, err := s.Batch.Service.Read(path)
		if err != nil {
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: CodeOf(err), MessageKey: MessageKeyOf(err), Detail: err.Error()})
			continue
		}
		applicable := fieldsForDocument(doc, selected)
		if len(applicable) == 0 {
			err := Ef("unsupported_field", "none of the selected fields are writable for %s", doc.Format)
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: CodeOf(err), MessageKey: MessageKeyOf(err), Detail: err.Error()})
			continue
		}
		eligible = append(eligible, clearFile{path: path, fields: applicable})
	}

	// Snapshot every eligible file before the first destructive write.
	ready := make([]clearFile, 0, len(eligible))
	for _, file := range eligible {
		if err := ctx.Err(); err != nil {
			return result, E("cancelled", err)
		}
		if _, err := s.Batch.Service.Snapshot(file.path); err != nil {
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: file.path, Status: "failed", Code: CodeOf(err), MessageKey: MessageKeyOf(err), Detail: err.Error()})
			continue
		}
		ready = append(ready, file)
	}
	for _, file := range ready {
		if err := ctx.Err(); err != nil {
			return result, E("cancelled", err)
		}
		writeResult, aerr := s.Batch.Service.Apply(ctx, file.path, clearAntiPatch(file.fields))
		item := BatchItem{Path: file.path}
		if aerr != nil {
			item.Status = "failed"
			item.Code = CodeOf(aerr)
			item.MessageKey = MessageKeyOf(aerr)
			item.Detail = aerr.Error()
			result.Failed++
		} else {
			item.Status = "updated"
			item.SnapshotID = writeResult.SnapshotID
			result.Updated++
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

// DerangeV2 applies a cryptographically seeded Sattolo permutation over the
// selected canonical text fields. All eligible files are read and snapshotted
// before the first write; pictures, numeric fields, unknown frames and raw
// container blocks are never moved.
func (s *AntiPiracyService) DerangeV2(ctx context.Context, paths []string, fields ...string) (*AntiPiracyResult, error) {
	if len(paths) < 2 {
		return nil, Ef("antipiracy_too_few", "V2 derange requires at least 2 files")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	selected, err := resolveAntiFields(fields)
	if err != nil {
		return nil, err
	}
	result := &AntiPiracyResult{OperationID: randHex(16), Total: len(paths)}
	if err := s.ensureReady(); err != nil {
		return result, err
	}

	// Read every source before the first write. A later write can therefore
	// never become the source for another file in the same operation.
	type eligibleFile struct {
		path   string
		doc    Document
		fields []string
	}
	eligible := make([]eligibleFile, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return result, E("cancelled", err)
		}
		if path == "" {
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: "file_unreadable", MessageKey: "errors.file_unreadable"})
			continue
		}
		if _, exists := seen[path]; exists {
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: "antipiracy_duplicate_path", MessageKey: "errors.antipiracy_duplicate_path"})
			continue
		}
		seen[path] = struct{}{}
		doc, err := s.Batch.Service.Read(path)
		if err != nil {
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: CodeOf(err), MessageKey: MessageKeyOf(err), Detail: err.Error()})
			continue
		}
		if doc.Capabilities == nil {
			err := Ef("unsupported_field", "capabilities are unavailable for %s", doc.Format)
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: CodeOf(err), MessageKey: MessageKeyOf(err), Detail: err.Error()})
			continue
		}
		applicable := fieldsForDocument(doc, selected)
		if len(applicable) == 0 {
			err := Ef("unsupported_field", "none of the selected fields are writable for %s", doc.Format)
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: CodeOf(err), MessageKey: MessageKeyOf(err), Detail: err.Error()})
			continue
		}
		eligible = append(eligible, eligibleFile{path: path, doc: doc, fields: applicable})
	}

	// Refuse to begin the destructive phase unless every file that remains in
	// the permutation has a verified recovery snapshot.
	ready := make([]eligibleFile, 0, len(eligible))
	for _, file := range eligible {
		if err := ctx.Err(); err != nil {
			return result, E("cancelled", err)
		}
		if _, err := s.Batch.Service.Snapshot(file.path); err != nil {
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: file.path, Status: "failed", Code: CodeOf(err), MessageKey: MessageKeyOf(err), Detail: err.Error()})
			continue
		}
		ready = append(ready, file)
	}
	if len(ready) < 2 {
		return result, Ef("antipiracy_too_few", "V2 requires at least 2 readable, writable files with verified snapshots")
	}
	perm, err := sattoloPermutation(len(ready))
	if err != nil {
		return result, err
	}
	for i, target := range ready {
		if err := ctx.Err(); err != nil {
			return result, E("cancelled", err)
		}
		source := ready[perm[i]].doc
		patch := setAntiPatch(target.fields, source)
		writeResult, aerr := s.Batch.Service.Apply(ctx, target.path, patch)
		item := BatchItem{Path: target.path}
		if aerr != nil {
			item.Status = "failed"
			item.Code = CodeOf(aerr)
			item.MessageKey = MessageKeyOf(aerr)
			item.Detail = aerr.Error()
			result.Failed++
		} else {
			item.Status = "updated"
			item.SnapshotID = writeResult.SnapshotID
			result.Updated++
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

// sattoloPermutation returns a uniformly sampled permutation with no fixed
// points. The cryptographic source is intentional: V2 changes metadata in a
// way that should not be predictable from process-local pseudo-random state.
func sattoloPermutation(n int) ([]int, error) {
	if n < 2 {
		return nil, Ef("antipiracy_too_few", "Sattolo permutation requires at least 2 items")
	}
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(i)))
		if err != nil {
			return nil, E("antipiracy_random_failed", err)
		}
		k := int(j.Int64())
		perm[i], perm[k] = perm[k], perm[i]
	}
	return perm, nil
}

// RestoreAll restores every path from its most recent complete snapshot.
func (s *AntiPiracyService) RestoreAll(ctx context.Context, paths []string) (*AntiPiracyResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := &AntiPiracyResult{OperationID: randHex(16), Total: len(paths)}
	if s == nil || s.Store == nil {
		return result, E("snapshot_store_missing", errors.New("no snapshot store"))
	}
	for _, p := range paths {
		if err := ctx.Err(); err != nil {
			return result, E("cancelled", err)
		}
		best, ok := s.Store.LatestFor(p)
		if !ok {
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: p, Status: "failed", Code: "snapshot_not_found", MessageKey: "errors.snapshot_not_found"})
			continue
		}
		writeResult, err := s.Store.Restore(ctx, best.ID)
		if err != nil {
			result.Failed++
			result.Items = append(result.Items, BatchItem{Path: p, Status: "failed", Code: CodeOf(err), MessageKey: MessageKeyOf(err), Detail: err.Error(), SnapshotID: best.ID})
			continue
		}
		result.Updated++
		result.Items = append(result.Items, BatchItem{Path: p, Status: "updated", SnapshotID: writeResult.SnapshotID})
	}
	return result, nil
}
