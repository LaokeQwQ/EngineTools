//go:build !windows

package metadata

import "os"

// POSIX rename is atomic over an existing target on the same volume.
func replaceFile(replacement, existing string) error {
	return os.Rename(replacement, existing)
}
