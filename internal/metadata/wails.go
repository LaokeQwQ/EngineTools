package metadata

import (
	"context"
)

// WailsService wraps Bridge operations for exposure through the Wails
// binding layer. All methods return typed results and sanitized public
// errors -- raw causes never cross the boundary.
type WailsService struct {
	bridge Bridge
	store  *SnapshotStore
}

func NewWailsService(bridge Bridge, store *SnapshotStore) *WailsService {
	return &WailsService{bridge: bridge, store: store}
}

func (w *WailsService) MetadataPickFile() string {
	// The frontend uses its own file picker; this is a no-op kept for
	// backward compatibility.
	return ""
}

func (w *WailsService) MetadataRead(path string) (Document, error) {
	return w.bridge.Read(path)
}

func (w *WailsService) MetadataApply(ctx context.Context, path string, patch Patch) (WriteResult, error) {
	return w.bridge.Apply(path, patch, WriteOptions{})
}

func (w *WailsService) MetadataPictures(path string) ([]Picture, error) {
	doc, err := w.bridge.Read(path)
	if err != nil {
		return nil, err
	}
	return doc.Pictures, nil
}
