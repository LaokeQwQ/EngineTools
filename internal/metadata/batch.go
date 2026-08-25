package metadata

import (
	"context"
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

// BatchService runs bounded-worker batch metadata edits through the same
// Service.Apply pipeline used for single-file writes.
type BatchService struct {
	Service *Service
	Workers int

	mu      sync.Mutex
	cancel  map[string]context.CancelFunc
	results map[string]*BatchResult
}

func NewBatchService(svc *Service) *BatchService {
	return &BatchService{
		Service: svc,
		Workers: 4,
		cancel:  make(map[string]context.CancelFunc),
		results: make(map[string]*BatchResult),
	}
}

// Run applies patch to every path concurrently, returning per-file results.
func (bs *BatchService) Run(ctx context.Context, paths []string, patch Patch) *BatchResult {
	opID := randHex(16)
	result := &BatchResult{
		OperationID: opID,
		Total:       len(paths),
	}

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	bs.mu.Lock()
	bs.cancel[opID] = cancel
	bs.mu.Unlock()

	// Pre-flight: probe each file and mark blocked ones.
	var eligible []string
	for _, p := range paths {
		caps, err := bs.Service.Bridge.Probe(p)
		if err != nil {
			result.Items = append(result.Items, BatchItem{
				Path: p, Status: "blocked",
				Code: "format_invalid", Detail: err.Error(),
			})
			result.Blocked++
			continue
		}
		_ = caps
		eligible = append(eligible, p)
	}

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
					continue
				default:
				}
				_, aerr := bs.Service.Apply(workerCtx, j.path, patch)
				item := BatchItem{Path: j.path}
				if aerr != nil {
					item.Status = "failed"
					item.Code = CodeOf(aerr)
					item.Detail = aerr.Error()
					mu.Lock()
					result.Items = append(result.Items, item)
					result.Failed++
					mu.Unlock()
				} else {
					item.Status = "updated"
					mu.Lock()
					result.Items = append(result.Items, item)
					result.Updated++
					mu.Unlock()
				}
			}
		}()
	}
	for _, p := range eligible {
		jobs <- job{path: p}
	}
	close(jobs)
	wg.Wait()

	// Sort items by path for stable output.
	sortItems(result.Items)
	return result
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
