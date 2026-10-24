package metadata

import (
	"context"
)

// WailsService wraps Bridge operations for exposure through the Wails
// binding layer. All methods return typed results and sanitized public
// errors -- raw causes never cross the boundary.
type WailsService struct {
	bridge  Bridge
	store   *SnapshotStore
	service *Service
	batch   *BatchService
}

func NewWailsService(bridge Bridge, store *SnapshotStore) *WailsService {
	service := NewService(bridge, store)
	return &WailsService{bridge: bridge, store: store, service: service, batch: NewBatchService(service)}
}

func (w *WailsService) MetadataPickFile() string {
	// The frontend uses its own file picker; this is a no-op kept for
	// backward compatibility.
	return ""
}

func (w *WailsService) MetadataCapabilities(path string) (Capabilities, error) {
	if w == nil || w.service == nil {
		return Capabilities{}, PublicError(E("bridge_unavailable", nil))
	}
	caps, err := w.service.Capabilities(path)
	if err != nil {
		return Capabilities{}, PublicError(err)
	}
	return caps, nil
}

func (w *WailsService) MetadataRead(path string) (Document, error) {
	if w == nil || w.bridge == nil {
		return Document{}, PublicError(E("bridge_unavailable", nil))
	}
	doc, err := w.service.Read(path)
	if err != nil {
		return Document{}, PublicError(err)
	}
	if caps, capErr := w.service.Capabilities(path); capErr == nil {
		doc.Capabilities = &caps
	}
	return SanitizeDocumentForUI(doc), nil
}

func (w *WailsService) MetadataApply(ctx context.Context, path string, patch Patch) (WriteResult, error) {
	if w == nil || w.service == nil {
		return WriteResult{}, PublicError(E("bridge_unavailable", nil))
	}
	result, err := w.service.Apply(ctx, path, patch)
	if err != nil {
		return WriteResult{}, PublicError(err)
	}
	return result, nil
}

func (w *WailsService) MetadataPictures(path string) ([]Picture, error) {
	doc, err := w.MetadataRead(path)
	if err != nil {
		return nil, err
	}
	return doc.Pictures, nil
}

func (w *WailsService) MetadataUndo(ctx context.Context, path string, snapshotID ...string) (WriteResult, error) {
	if w == nil || w.service == nil {
		return WriteResult{}, PublicError(E("bridge_unavailable", nil))
	}
	result, err := w.service.Undo(ctx, path, snapshotID...)
	if err != nil {
		return WriteResult{}, PublicError(err)
	}
	return result, nil
}

func (w *WailsService) MetadataBatchApply(ctx context.Context, paths []string, patch Patch) (*BatchResult, error) {
	if w == nil || w.batch == nil {
		return nil, PublicError(E("bridge_unavailable", nil))
	}
	return w.batch.Run(ctx, paths, patch), nil
}

func (w *WailsService) MetadataBatchStart(paths []string, patch Patch) (string, error) {
	if w == nil || w.batch == nil {
		return "", PublicError(E("bridge_unavailable", nil))
	}
	return w.batch.Start(context.Background(), paths, patch), nil
}

func (w *WailsService) MetadataBatchPreview(ctx context.Context, paths []string, patch Patch) (*BatchPreview, error) {
	if w == nil || w.batch == nil {
		return nil, PublicError(E("bridge_unavailable", nil))
	}
	preview, err := w.batch.Preview(ctx, paths, patch)
	if err != nil {
		return nil, PublicError(err)
	}
	return preview, nil
}

func (w *WailsService) MetadataBatchResult(operationID string) (*BatchResult, error) {
	if w == nil || w.batch == nil {
		return nil, PublicError(E("bridge_unavailable", nil))
	}
	result, ok := w.batch.Result(operationID)
	if !ok {
		return nil, PublicError(E("operation_not_found", nil))
	}
	return result, nil
}

func (w *WailsService) MetadataCancel(operationID string) bool {
	if w == nil || w.batch == nil {
		return false
	}
	w.batch.Cancel(operationID)
	return true
}
