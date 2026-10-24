package metadata

import "context"

// Bridge is the dependency-injection seam for the native TagLib engine.
// Production wires the Windows DLL implementation; tests inject fakes.
type Bridge interface {
	Probe(path string) (Capabilities, error)
	Read(path string) (Document, error)
	Apply(path string, patch Patch, options WriteOptions) (WriteResult, error)
	AudioPayloadSHA256(path string) ([32]byte, error)
}

// WriteOptions carries per-write correlation data.
type WriteOptions struct {
	OperationID string
}

// WriteResult is the outcome of one Apply through the bridge.
type WriteResult struct {
	OK          bool   `json:"ok"`
	Changed     int    `json:"changed"`
	SnapshotID  string `json:"snapshotId,omitempty"`
	OperationID string `json:"operationId,omitempty"`
}

// capabilitiesForDocument resolves the capability table for whatever
// format the bridge detected.
func capabilitiesForDocument(d Document) (Capabilities, bool) {
	return CapabilitiesFor(d.Format)
}

// ensureContext keeps ctx in the signature of future bridge operations.
func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
