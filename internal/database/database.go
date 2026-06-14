package database

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// dbRelPaths are the standard relative locations of m.db on an external drive.
var dbRelPaths = []string{
	filepath.Join("Engine Library", "Database2", "m.db"),
	filepath.Join("Engine Library", "Database", "m.db"),
}

// manualPath holds a user-selected library path. When set and still present,
// it takes priority over auto-detection so every database operation targets
// the drive the user picked.
var manualPath string

// SetLibraryPath pins the active library to a specific m.db path.
func SetLibraryPath(p string) { manualPath = p }

// ClearLibraryPath drops the manual selection and returns to auto-detection.
func ClearLibraryPath() { manualPath = "" }

// ResolveLibrary returns the active library path: the manual selection if it
// is still valid, otherwise the result of auto-detection.
func ResolveLibrary() (string, error) {
	if manualPath != "" {
		if _, err := os.Stat(manualPath); err == nil {
			return manualPath, nil
		}
	}
	return FindEngineLibrary()
}

// FindEngineLibrary locates the Engine Library m.db. It first checks the
// AppData install locations, then scans every drive root for the standard
// "Engine Library/Database2/m.db" layout that Engine DJ uses on external
// drives and USB sticks.
func FindEngineLibrary() (string, error) {
	candidates := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "EnginePrime", "Database", "m.db"),
		filepath.Join(os.Getenv("APPDATA"), "EnginePrime", "Database", "m.db"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Engine DJ", "Database", "m.db"),
		filepath.Join(os.Getenv("APPDATA"), "Engine DJ", "Database", "m.db"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Scan drive roots (external drives / USB sticks store the library at
	// <drive>:\Engine Library\Database2\m.db).
	for _, root := range driveRoots() {
		if p, ok := findInRoot(root); ok {
			return p, nil
		}
	}

	return "", fmt.Errorf("Engine Library m.db not found")
}

// FindInDrive looks for an Engine Library on a single drive (e.g. "D:").
func FindInDrive(drive string) (string, error) {
	drive = strings.TrimSpace(drive)
	if drive == "" {
		return "", fmt.Errorf("empty drive")
	}
	if !strings.HasSuffix(drive, `\`) {
		drive += `\`
	}
	if p, ok := findInRoot(drive); ok {
		return p, nil
	}
	return "", fmt.Errorf("Engine Library not found on %s", drive)
}

func findInRoot(root string) (string, bool) {
	for _, rel := range dbRelPaths {
		p := filepath.Join(root, rel)
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

// driveRoots returns the root path of every present drive (C:\ .. Z:\).
func driveRoots() []string {
	var roots []string
	for c := 'A'; c <= 'Z'; c++ {
		root := string(c) + `:\`
		if _, err := os.Stat(root); err == nil {
			roots = append(roots, root)
		}
	}
	return roots
}

// ListDrives returns the present drive letters in "C:" form.
func ListDrives() []string {
	var drives []string
	for c := 'A'; c <= 'Z'; c++ {
		root := string(c) + `:\`
		if _, err := os.Stat(root); err == nil {
			drives = append(drives, string(c)+":")
		}
	}
	return drives
}

type BackupInfo struct {
	Filename string `json:"filename"`
	Date     string `json:"date"`
	Size     int64  `json:"size"`
	Note     string `json:"note"`
}

func backupDir() (string, error) {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "EngineTools", "Backups")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}
	return dir, nil
}

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func BackupDatabase(note string) (string, error) {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return "", err
	}

	dir, err := backupDir()
	if err != nil {
		return "", err
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	dstPath := filepath.Join(dir, fmt.Sprintf("engine_library_%s.db", timestamp))
	if err := CopyFile(dbPath, dstPath); err != nil {
		return "", fmt.Errorf("failed to backup database: %w", err)
	}

	if note != "" {
		notePath := filepath.Join(dir, fmt.Sprintf("engine_library_%s.txt", timestamp))
		os.WriteFile(notePath, []byte(note), 0644)
	}

	return dstPath, nil
}

func ListBackups() ([]BackupInfo, error) {
	dir, err := backupDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Filename: e.Name(),
			Date:     info.ModTime().Format("2006-01-02 15:04:05"),
			Size:     info.Size(),
		})
	}
	return backups, nil
}

func RestoreDatabase(filename string) error {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return err
	}

	dir, err := backupDir()
	if err != nil {
		return err
	}

	srcPath := filepath.Join(dir, filename)
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("backup file not found: %s", filename)
	}

	return CopyFile(srcPath, dbPath)
}

func OptimizeDatabase() error {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("failed to optimize database: %w", err)
	}

	_, err = db.Exec("REINDEX")
	if err != nil {
		return fmt.Errorf("failed to reindex database: %w", err)
	}

	return nil
}