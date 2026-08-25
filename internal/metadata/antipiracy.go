package metadata

import (
	"context"
	"crypto/sha256"
	"math/rand"
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
	Items       []BatchItem `json:"items"`
}

func NewAntiPiracyService(batch *BatchService, store *SnapshotStore, bridge Bridge) *AntiPiracyService {
	return &AntiPiracyService{Batch: batch, Store: store, Bridge: bridge}
}

// ClearV1 removes title and album across multiple files.
func (s *AntiPiracyService) ClearV1(ctx context.Context, paths []string) (*AntiPiracyResult, error) {
	if len(paths) < 2 {
		return nil, Ef("antipiracy_too_few", "V1 clear requires at least 2 files")
	}
	patch := Patch{
		Title: FieldPatch[string]{Mode: Clear},
		Album: FieldPatch[string]{Mode: Clear},
	}
	res := s.Batch.Run(ctx, paths, patch)
	return &AntiPiracyResult{OperationID: res.OperationID, Total: res.Total,
		Updated: res.Updated, Failed: res.Failed, Items: res.Items}, nil
}

// DerangeV2 applies a seeded permutation over selected text fields.
func (s *AntiPiracyService) DerangeV2(ctx context.Context, paths []string) (*AntiPiracyResult, error) {
	if len(paths) < 2 {
		return nil, Ef("antipiracy_too_few", "V2 derange requires at least 2 files")
	}
	seed := sha256.Sum256([]byte(randHex(32)))
	rng := rand.New(rand.NewSource(int64(seed[0])<<56 | int64(seed[1])<<48))
	perm := rng.Perm(len(paths))

	result := &AntiPiracyResult{OperationID: randHex(16), Total: len(paths)}
	for i := range paths {
		j := perm[i]
		doc, rerr := s.Bridge.Read(paths[j])
		if rerr != nil {
			result.Failed++
			continue
		}
		_, aerr := s.Bridge.Apply(paths[i], Patch{
			Title: FieldPatch[string]{Mode: Set, Value: doc.Title},
		}, WriteOptions{})
		if aerr != nil {
			result.Failed++
			continue
		}
		result.Updated++
	}
	return result, nil
}

// RestoreAll restores every path from its most recent complete snapshot.
func (s *AntiPiracyService) RestoreAll(ctx context.Context, paths []string) (*AntiPiracyResult, error) {
	result := &AntiPiracyResult{OperationID: randHex(16), Total: len(paths)}
	for _, p := range paths {
		all := s.Store.List()
		var best *Snapshot
		for i := range all {
			if all[i].OriginalPath == p && all[i].State == "complete" {
				if best == nil || all[i].CreatedAt.After(best.CreatedAt) {
					best = &all[i]
				}
			}
		}
		if best == nil {
			result.Failed++
			continue
		}
		if err := s.Store.RestoreBin(best.ID, p); err != nil {
			result.Failed++
			continue
		}
		result.Updated++
	}
	return result, nil
}
