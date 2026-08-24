//go:build windows

package metadata

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// replaceFile atomically swaps replacement into the existing path using
// ReplaceFileW, which preserves attributes and works even when readers
// hold the file open with share-delete.
func replaceFile(replacement, existing string) error {
	repW, err := windows.UTF16PtrFromString(replacement)
	if err != nil {
		return err
	}
	existW, err := windows.UTF16PtrFromString(existing)
	if err != nil {
		return err
	}
	if _, _, err := procReplaceFile.Call(
		uintptr(unsafe.Pointer(existW)),
		uintptr(unsafe.Pointer(repW)),
		0,   // no backup
		0x1, // REPLACEFILE_WRITE_THROUGH
		0, 0,
	); err != nil && err != windows.ERROR_SUCCESS {
		// Fall back to plain rename when the target does not exist yet.
		if _, statErr := os.Stat(existing); os.IsNotExist(statErr) {
			return os.Rename(replacement, existing)
		}
		return err
	}
	return nil
}

var procReplaceFile = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReplaceFileW")
