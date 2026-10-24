package metadata

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
)

// BatchItem tracks one file's outcome in a batch operation.
type BatchItem struct {
	Path       string `json:"path"`
	Status     string `json:"status"` // updated | skipped | failed | blocked
	Code       string `json:"code,omitempty"`
	MessageKey string `json:"messageKey,omitempty"`
	Detail     string `json:"detail,omitempty"`
	SnapshotID string `json:"snapshotId,omitempty"`
}

// BatchResult aggregates all per-file outcomes for one batch run.
type BatchResult struct {
	OperationID string      `json:"operationId"`
	Total       int         `json:"total"`
	Updated     int         `json:"updated"`
	Skipped     int         `json:"skipped"`
	Failed      int         `json:"failed"`
	Blocked     int         `json:"blocked"`
	Cancelled   bool        `json:"cancelled"`
	Items       []BatchItem `json:"items"`
}

// BatchPreview describes which files can accept a patch without creating
// snapshots or modifying any file.
type BatchPreview struct {
	OperationID string      `json:"operationId"`
	Total       int         `json:"total"`
	Eligible    int         `json:"eligible"`
	Blocked     int         `json:"blocked"`
	Items       []BatchItem `json:"items"`
}

// BatchService runs bounded-worker batch metadata edits through the same
// Service.Apply pipeline used for single-file writes.
type BatchService struct {
	Service *Service
	Workers int

	mu      sync.Mutex
	cancel  map[string]context.CancelFunc
	results map[string]*BatchResult
}

const maxRetainedBatchResults = 64

func NewBatchService(svc *Service) *BatchService {
	return &BatchService{
		Service: svc,
		Workers: 4,
		cancel:  make(map[string]context.CancelFunc),
		results: make(map[string]*BatchResult),
	}
}

func (bs *BatchService) Preview(ctx context.Context, paths []string, patch Patch) (*BatchPreview, error) {
	if bs == nil || bs.Service == nil || bs.Service.Bridge == nil {
		return nil, E("bridge_unavailable", errors.New("metadata bridge is unavailable"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, E("cancelled", err)
	}
	if err := patch.ValidateForBatch(); err != nil {
		return nil, err
	}
	preview := &BatchPreview{OperationID: randHex(16), Total: len(paths)}
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, E("cancelled", err)
		}
		item := BatchItem{Path: path}
		if path == "" {
			item.Status = "blocked"
			item.Code = "file_unreadable"
			item.MessageKey = "errors.file_unreadable"
			preview.Blocked++
			preview.Items = append(preview.Items, item)
			continue
		}
		clean := filepath.Clean(path)
		if _, exists := seen[clean]; exists {
			item.Status = "blocked"
			item.Code = "duplicate_path"
			item.MessageKey = "errors.duplicate_path"
			item.Detail = "duplicate path in batch"
			preview.Blocked++
			preview.Items = append(preview.Items, item)
			continue
		}
		seen[clean] = struct{}{}

		caps, err := bs.Service.Capabilities(path)
		if err == nil {
			err = validateBatchCapabilities(caps, patch)
		}
		if err != nil {
			item.Status = "blocked"
			item.Code = CodeOf(err)
			item.MessageKey = MessageKeyOf(err)
			item.Detail = err.Error()
			preview.Blocked++
		} else {
			item.Status = "eligible"
			preview.Eligible++
		}
		preview.Items = append(preview.Items, item)
	}
	sortItems(preview.Items)
	return preview, nil
}

// Run applies patch to every path concurrently, returning per-file results.
func (bs *BatchService) Run(ctx context.Context, paths []string, patch Patch) *BatchResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if bs == nil {
		result := &BatchResult{OperationID: randHex(16), Total: len(paths), Failed: len(paths)}
		for _, path := range paths {
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: "bridge_unavailable", MessageKey: "errors.bridge_unavailable"})
		}
		return result
	}
	return bs.run(ctx, randHex(16), paths, patch, nil, nil)
}

// Start launches a batch in the background and immediately returns its stable
// operation ID. The result placeholder and cancel function are registered
// before the goroutine starts, so Result/Cancel are valid immediately after
// Start returns. Call Result to inspect progress or completion and Cancel to
// stop pending files; each in-flight Service.Apply remains atomic.
func (bs *BatchService) Start(ctx context.Context, paths []string, patch Patch) string {
	if bs == nil {
		return ""
	}
	if ctx == nil {
		ctx = context.Background()
	}
	opID := randHex(16)
	workerCtx, cancel := context.WithCancel(ctx)
	result := &BatchResult{OperationID: opID, Total: len(paths)}
	bs.mu.Lock()
	bs.cancel[opID] = cancel
	bs.results[opID] = cloneBatchResult(result)
	bs.mu.Unlock()
	go bs.run(workerCtx, opID, paths, patch, result, cancel)
	return opID
}

func (bs *BatchService) run(ctx context.Context, opID string, paths []string, patch Patch, pending *BatchResult, cancel context.CancelFunc) *BatchResult {
	if ctx == nil {
		ctx = context.Background()
	}
	result := pending
	if result == nil {
		result = &BatchResult{
			OperationID: opID,
			Total:       len(paths),
		}
	}
	workerCtx := ctx
	if cancel == nil {
		var child context.Context
		child, cancel = context.WithCancel(ctx)
		workerCtx = child
	}
	// Register the finalizer before any early return so a placeholder created
	// by Start is always replaced by a terminal result.
	defer func() {
		cancel()
		bs.mu.Lock()
		delete(bs.cancel, opID)
		bs.results[opID] = cloneBatchResult(result)
		if len(bs.results) > maxRetainedBatchResults {
			for id := range bs.results {
				if id != opID {
					delete(bs.results, id)
					break
				}
			}
		}
		bs.mu.Unlock()
	}()
	bs.mu.Lock()
	bs.cancel[opID] = cancel
	bs.mu.Unlock()

	if bs.Service == nil || bs.Service.Bridge == nil {
		result.Cancelled = false
		result.Failed = len(paths)
		for _, path := range paths {
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: "bridge_unavailable", MessageKey: "errors.bridge_unavailable"})
		}
		return result
	}
	if err := patch.ValidateForBatch(); err != nil {
		result.Failed = len(paths)
		for _, path := range paths {
			result.Items = append(result.Items, BatchItem{Path: path, Status: "failed", Code: CodeOf(err), MessageKey: MessageKeyOf(err)})
		}
		return result
	}

	// Pre-flight: probe each file and mark blocked ones.
	var eligible []string
	seen := make(map[string]struct{}, len(paths))
	for i, p := range paths {
		select {
		case <-workerCtx.Done():
			for _, remaining := range paths[i:] {
				result.Items = append(result.Items, BatchItem{Path: remaining, Status: "skipped"})
				result.Skipped++
			}
			result.Cancelled = true
			sortItems(result.Items)
			return result
		default:
		}
		if p == "" {
			result.Items = append(result.Items, BatchItem{
				Path: p, Status: "blocked",
				Code: "file_unreadable", MessageKey: "errors.file_unreadable",
			})
			result.Blocked++
			continue
		}
		clean := filepath.Clean(p)
		if _, exists := seen[clean]; exists {
			result.Items = append(result.Items, BatchItem{
				Path: p, Status: "blocked",
				Code: "duplicate_path", MessageKey: "errors.duplicate_path",
				Detail: "duplicate path in batch",
			})
			result.Blocked++
			continue
		}
		seen[clean] = struct{}{}

		caps, err := bs.Service.Capabilities(p)
		if err != nil {
			result.Items = append(result.Items, BatchItem{
				Path: p, Status: "blocked",
				Code: CodeOf(err), MessageKey: MessageKeyOf(err), Detail: err.Error(),
			})
			result.Blocked++
			continue
		}
		if err := validateBatchCapabilities(caps, patch); err != nil {
			result.Items = append(result.Items, BatchItem{Path: p, Status: "blocked", Code: CodeOf(err), MessageKey: MessageKeyOf(err), Detail: err.Error()})
			result.Blocked++
			continue
		}
		eligible = append(eligible, p)
	}
	// Publish blocked/preflight outcomes before workers start so a caller that
	// polls Result sees progress even when the first writes take a long time.
	bs.publishResult(opID, result, nil)

	type job struct{ path string }
	jobs := make(chan job, len(eligible))
	var wg sync.WaitGroup

	workers := bs.Workers
	if workers <= 0 {
		workers = 4
	}
	if workers > len(eligible) {
		workers = len(eligible)
	}

	var mu sync.Mutex
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				select {
				case <-workerCtx.Done():
					mu.Lock()
					result.Items = append(result.Items, BatchItem{Path: j.path, Status: "skipped"})
					result.Skipped++
					mu.Unlock()
					bs.publishResult(opID, result, &mu)
					continue
				default:
				}
				writeResult, aerr := bs.Service.Apply(workerCtx, j.path, patch)
				item := BatchItem{Path: j.path}
				if aerr != nil {
					if CodeOf(aerr) == "cancelled" {
						item.Status = "skipped"
						mu.Lock()
						result.Items = append(result.Items, item)
						result.Skipped++
						mu.Unlock()
						bs.publishResult(opID, result, &mu)
						continue
					}
					item.Status = "failed"
					item.Code = CodeOf(aerr)
					item.MessageKey = MessageKeyOf(aerr)
					item.Detail = aerr.Error()
					mu.Lock()
					result.Items = append(result.Items, item)
					result.Failed++
					mu.Unlock()
				} else {
					item.Status = "updated"
					item.SnapshotID = writeResult.SnapshotID
					mu.Lock()
					result.Items = append(result.Items, item)
					result.Updated++
					mu.Unlock()
				}
				bs.publishResult(opID, result, &mu)
			}
		}()
	}
	for _, p := range eligible {
		jobs <- job{path: p}
	}
	close(jobs)
	wg.Wait()
	result.Cancelled = workerCtx.Err() != nil

	// Sort items by path for stable output.
	sortItems(result.Items)
	return result
}

func (bs *BatchService) Result(operationID string) (*BatchResult, bool) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	result, ok := bs.results[operationID]
	if !ok {
		return nil, false
	}
	return cloneBatchResult(result), true
}

// publishResult snapshots a live batch result into the retained-results map.
// resultMu protects fields mutated by workers; bs.mu protects the map itself.
// Keeping the two locks separate avoids blocking new Result readers while a
// worker is waiting for another worker to finish appending.
func (bs *BatchService) publishResult(operationID string, result *BatchResult, resultMu *sync.Mutex) {
	if bs == nil || result == nil {
		return
	}
	if resultMu != nil {
		resultMu.Lock()
	}
	snapshot := cloneBatchResult(result)
	if resultMu != nil {
		resultMu.Unlock()
	}
	bs.mu.Lock()
	bs.results[operationID] = snapshot
	bs.mu.Unlock()
}

func cloneBatchResult(in *BatchResult) *BatchResult {
	if in == nil {
		return nil
	}
	out := *in
	out.Items = append([]BatchItem(nil), in.Items...)
	return &out
}

// Cancel aborts an in-flight batch by operation ID.
func (bs *BatchService) Cancel(operationID string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if cancelFn, ok := bs.cancel[operationID]; ok {
		cancelFn()
	}
}

func sortItems(items []BatchItem) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].Path < items[j-1].Path; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
