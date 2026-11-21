package logs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Daily struct {
	Directory string
	mu        sync.Mutex
}

func New(directory string) *Daily { return &Daily{Directory: directory} }

func (d *Daily) Write(source, message string, at time.Time) (string, error) {
	if at.IsZero() {
		at = time.Now()
	}
	at = at.Local()
	message = strings.ReplaceAll(strings.ReplaceAll(message, "\r\n", "\n"), "\r", "\n")
	entry := fmt.Sprintf("[%s] [%s] %s", at.Format("2006-01-02 15:04:05.000"), source, strings.ReplaceAll(message, "\n", "\n    "))
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := os.MkdirAll(d.Directory, 0755); err != nil {
		return entry, err
	}
	f, err := os.OpenFile(filepath.Join(d.Directory, at.Format("2006-01-02")+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return entry, err
	}
	_, writeErr := f.WriteString(entry + "\n")
	closeErr := f.Close()
	if writeErr != nil {
		return entry, writeErr
	}
	return entry, closeErr
}

// Framework logs go directly to disk to avoid recursively emitting UI events.
type Framework struct{ Daily *Daily }

func (f *Framework) write(level, message string) {
	if _, err := f.Daily.Write("framework/"+level, message, time.Time{}); err != nil {
		fmt.Fprintln(os.Stderr, "Log write failed:", err)
	}
}
func (f *Framework) Print(m string)   { f.write("INFO", m) }
func (f *Framework) Trace(m string)   { f.write("TRACE", m) }
func (f *Framework) Debug(m string)   { f.write("DEBUG", m) }
func (f *Framework) Info(m string)    { f.write("INFO", m) }
func (f *Framework) Warning(m string) { f.write("WARN", m) }
func (f *Framework) Error(m string)   { f.write("ERROR", m) }
func (f *Framework) Fatal(m string)   { f.write("FATAL", m); os.Exit(1) }
