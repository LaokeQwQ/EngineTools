//go:build !windows

package metadata

import "errors"

// NewNativeBridge keeps the application facade buildable on development
// hosts that do not ship the Windows TagLib DLL. Production builds use the
// Windows implementation in bridge_windows.go.
func NewNativeBridge(string) (Bridge, error) {
	return nil, errors.New("native bridge requires windows")
}

// Off-Windows hosts have no enginetools_tag.dll; the smoke caller skips.
func newNativeBridgeForTest(string) (Bridge, error) {
	return nil, errors.New("native bridge requires windows")
}
