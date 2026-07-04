package experimental

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func xmlPath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "EngineTools", "experimental_allow.xml")
}

// LogEnabled creates or appends a timestamp entry to experimental_allow.xml.
// The file is kept hidden + read-only. Multiple calls append new entries.
func LogEnabled() error {
	p := xmlPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	ts := time.Now().Format(time.RFC3339)
	newEntry := fmt.Sprintf("  <activation timestamp=%q/>\n", ts)

	if _, err := os.Stat(p); os.IsNotExist(err) {
		// First time: create file
		content := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
			"<experimental-features>\n" + newEntry + "</experimental-features>\n"
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			return fmt.Errorf("write: %w", err)
		}
	} else {
		// Subsequent times: remove readonly, append new entry, re-apply attrs
		if err := removeReadonly(p); err != nil {
			return fmt.Errorf("removeReadonly: %w", err)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		content := strings.TrimRight(string(raw), "\r\n\t ")
		const closing = "</experimental-features>"
		if strings.HasSuffix(content, closing) {
			content = content[:len(content)-len(closing)]
		}
		content += "\n" + newEntry + closing + "\n"
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			return fmt.Errorf("rewrite: %w", err)
		}
	}
	return setHiddenReadonly(p)
}

func setHiddenReadonly(path string) error {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attrs, err := syscall.GetFileAttributes(ptr)
	if err != nil {
		return err
	}
	attrs |= syscall.FILE_ATTRIBUTE_HIDDEN | syscall.FILE_ATTRIBUTE_READONLY
	return syscall.SetFileAttributes(ptr, attrs)
}

func removeReadonly(path string) error {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attrs, err := syscall.GetFileAttributes(ptr)
	if err != nil {
		return err
	}
	attrs &^= syscall.FILE_ATTRIBUTE_READONLY
	return syscall.SetFileAttributes(ptr, attrs)
}
