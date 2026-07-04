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
	"golang.org/x/sys/windows"
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
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	drives := ListDrives()
	for _, drive := range drives {
		for _, rel := range dbRelPaths {
			candidate := filepath.Join(drive+`\`, rel)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("Engine Library not found")
}

// FindInDrive locates m.db specifically on the given drive letter (e.g. "D:").
// Returns the full path to m.db if found, otherwise an error.
func FindInDrive(drive string) (string, error) {
	for _, rel := range dbRelPaths {
		candidate := filepath.Join(drive+`\`, rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("Engine Library not found on drive %s", drive)
}

// ListDrives returns all mounted drive letters (e.g. "C:", "D:").
func ListDrives() []string {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		// fallback to stat-based detection
		var drives []string
		for c := 'A'; c <= 'Z'; c++ {
			root := string(c) + ":\\"
			if _, err := os.Stat(root); err == nil {
				drives = append(drives, string(c)+":")
			}
		}
		return drives
	}
	var drives []string
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) != 0 {
			drives = append(drives, string(rune('A'+i))+":")
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
		return "", fmt.Errorf("no Engine Library found: %w", err)
	}

	dir, err := backupDir()
	if err != nil {
		return "", err
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("m_%s.db", timestamp)
	if note != "" {
		safeNote := strings.Map(func(r rune) rune {
			if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
				return '_'
			}
			return r
		}, note)
		filename = fmt.Sprintf("m_%s_%s.db", timestamp, safeNote)
	}

	backupPath := filepath.Join(dir, filename)
	if err := CopyFile(dbPath, backupPath); err != nil {
		return "", fmt.Errorf("backup failed: %w", err)
	}

	return backupPath, nil
}

func ListBackups() ([]BackupInfo, error) {
	dir, err := backupDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		parts := strings.SplitN(entry.Name(), "_", 4)
		dateStr := ""
		note := ""
		if len(parts) >= 3 {
			dateStr = parts[1] + " " + strings.Replace(parts[2], "-", ":", -1)
		}
		if len(parts) == 4 {
			note = strings.TrimSuffix(parts[3], ".db")
		}

		backups = append(backups, BackupInfo{
			Filename: entry.Name(),
			Date:     dateStr,
			Size:     info.Size(),
			Note:     note,
		})
	}

	return backups, nil
}

func RestoreDatabase(backupFilename string) error {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return fmt.Errorf("no Engine Library found: %w", err)
	}

	dir, err := backupDir()
	if err != nil {
		return err
	}

	backupPath := filepath.Join(dir, backupFilename)
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	if err := CopyFile(backupPath, dbPath); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	return nil
}

func OptimizeDatabase() error {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return fmt.Errorf("no Engine Library found: %w", err)
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

// CountTracks returns the number of tracks in the given database.
func CountTracks(dbPath string) int {
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro&cache=shared")
	if err != nil {
		return 0
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM Track").Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// RepairDatabase runs integrity checks and repairs on the database.
// Returns a report of what was checked and fixed.
func RepairDatabase() (string, error) {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return "", fmt.Errorf("no Engine Library found: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	report := ""

	// 1. Integrity check
	var integrityResult string
	err = db.QueryRow("PRAGMA integrity_check").Scan(&integrityResult)
	if err != nil {
		return "", fmt.Errorf("integrity check failed: %w", err)
	}

	if integrityResult == "ok" {
		report += "✓ Integrity check: OK\n"
	} else {
		report += "✗ Integrity check: " + integrityResult + "\n"
	}

	// 2. Foreign key check
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		return report, fmt.Errorf("foreign key check failed: %w", err)
	}
	fkErrors := 0
	for rows.Next() {
		fkErrors++
	}
	rows.Close()

	if fkErrors == 0 {
		report += "✓ Foreign key check: OK\n"
	} else {
		report += fmt.Sprintf("✗ Foreign key check: %d errors\n", fkErrors)
	}

	// 3. Reindex
	_, err = db.Exec("REINDEX")
	if err != nil {
		report += "✗ Reindex: failed\n"
	} else {
		report += "✓ Reindex: completed\n"
	}

	// 4. Analyze (update query optimizer statistics)
	_, err = db.Exec("ANALYZE")
	if err != nil {
		report += "✗ Analyze: failed\n"
	} else {
		report += "✓ Analyze: completed\n"
	}

	return report, nil
}
