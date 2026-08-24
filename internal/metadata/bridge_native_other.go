//go:build !windows

package metadata

import "errors"

// Off-Windows hosts have no enginetools_tag.dll; the smoke caller skips.
func newNativeBridgeForTest(string) (Bridge, error) {
	return nil, errors.New("native bridge requires windows")
}
