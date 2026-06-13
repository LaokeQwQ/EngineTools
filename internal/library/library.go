// Package library provides Engine DJ library database scanning and overview file restoration.
package library

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// toFileURI converts a Windows absolute path to a SQLite file URI.
// Spaces and backslashes are handled so "D:\Engine Library\..." works.
func toFileURI(p string) string {
	p = filepath.ToSlash(p)
	p = strings.ReplaceAll(p, " ", "%20")
	return "file:///" + p + "?mode=ro&_busy_timeout=3000"
}

// DBInfo holds the status of a single Engine Library database.
type DBInfo struct {
	Path        string
	Drive       string
	UUID        string
	TotalTracks int
	MissingRGB  int
}

// ScanAll scans drives C–Z for Engine Library databases at the standard path.
func ScanAll() []DBInfo {
	var results []DBInfo
	for c := 'C'; c <= 'Z'; c++ {
		dbPath := fmt.Sprintf(`%c:\Engine Library\Database2\m.db`, c)
		if _, err := os.Stat(dbPath); err != nil {
			continue
		}
		info, err := inspectDB(dbPath, fmt.Sprintf("%c:", c))
		if err != nil {
			continue
		}
		results = append(results, info)
	}
	return results
}

func openDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", toFileURI(dbPath))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("open %s: %w", dbPath, err)
	}
	return db, nil
}

func inspectDB(dbPath, drive string) (DBInfo, error) {
	info := DBInfo{Path: dbPath, Drive: drive}

	db, err := openDB(dbPath)
	if err != nil {
		return info, err
	}
	defer db.Close()

	if err := db.QueryRow("SELECT uuid FROM Information LIMIT 1").Scan(&info.UUID); err != nil {
		return info, fmt.Errorf("read uuid: %w", err)
	}

	if err := db.QueryRow(`
		SELECT COUNT(*) FROM Track t
		JOIN PerformanceData p ON p.trackId = t.id
		WHERE p.overviewWaveFormData IS NOT NULL AND length(p.overviewWaveFormData) > 0
	`).Scan(&info.TotalTracks); err != nil {
		return info, fmt.Errorf("count tracks: %w", err)
	}

	overviewDir := filepath.Join(filepath.Dir(dbPath), "OverviewData", info.UUID)
	rows, err := db.Query(`
		SELECT t.id FROM Track t
		JOIN PerformanceData p ON p.trackId = t.id
		WHERE p.overviewWaveFormData IS NOT NULL AND length(p.overviewWaveFormData) > 0
	`)
	if err != nil {
		return info, fmt.Errorf("query track ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		rgbPath := filepath.Join(overviewDir, fmt.Sprintf("%d.rgb", id))
		fi, err := os.Stat(rgbPath)
		if os.IsNotExist(err) || (err == nil && fi.Size() < 100) {
			info.MissingRGB++
		}
	}

	return info, nil
}

// RestoreResult holds the outcome of a single restore operation.
type RestoreResult struct {
	Written int
	Skipped int
	Errors  int
}

// Restore writes missing .rgb files for a single database by reading
// overviewWaveFormData BLOBs from the PerformanceData table.
// progressFn is called with values in [0,1] during the operation.
func Restore(dbPath string, progressFn func(float64)) (RestoreResult, error) {
	var result RestoreResult

	db, err := openDB(dbPath)
	if err != nil {
		return result, err
	}
	defer db.Close()

	var uuid string
	if err := db.QueryRow("SELECT uuid FROM Information LIMIT 1").Scan(&uuid); err != nil {
		return result, fmt.Errorf("read uuid: %w", err)
	}

	overviewDir := filepath.Join(filepath.Dir(dbPath), "OverviewData", uuid)
	if err := os.MkdirAll(overviewDir, 0755); err != nil {
		return result, fmt.Errorf("mkdir overview dir: %w", err)
	}

	rows, err := db.Query(`
		SELECT t.id, p.overviewWaveFormData FROM Track t
		JOIN PerformanceData p ON p.trackId = t.id
		WHERE p.overviewWaveFormData IS NOT NULL AND length(p.overviewWaveFormData) > 0
	`)
	if err != nil {
		return result, fmt.Errorf("query blobs: %w", err)
	}

	type entry struct {
		id   int
		blob []byte
	}
	var entries []entry
	for rows.Next() {
		var id int
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			continue
		}
		entries = append(entries, entry{id, blob})
	}
	rows.Close()

	total := len(entries)
	if total == 0 {
		return result, nil
	}

	for i, e := range entries {
		if progressFn != nil {
			progressFn(float64(i) / float64(total))
		}
		rgbPath := filepath.Join(overviewDir, fmt.Sprintf("%d.rgb", e.id))
		if fi, err := os.Stat(rgbPath); err == nil && fi.Size() >= int64(len(e.blob)) {
			result.Skipped++
			continue
		}
		if err := os.WriteFile(rgbPath, e.blob, 0644); err != nil {
			result.Errors++
			continue
		}
		result.Written++
	}

	if progressFn != nil {
		progressFn(1.0)
	}

	return result, nil
}
