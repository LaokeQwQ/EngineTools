//go:build windows

package metadata

// newNativeBridgeForTest exposes the production DLL bridge to the portable
// contract tests on Windows.
func newNativeBridgeForTest(dllPath string) (Bridge, error) {
	return NewNativeBridge(dllPath)
}
