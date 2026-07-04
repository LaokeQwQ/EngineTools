package database

import (
	"database/sql"
	"fmt"
	"math"
)

// SyncableTrack holds the analysis data that can be written back to a file's tags.
type SyncableTrack struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Path     string  `json:"path"`
	BPM      float64 `json:"bpm"`
	Key      int     `json:"key"`
	KeyName  string  `json:"keyName"`
	Camelot  string  `json:"camelot"`
}

// SyncResult summarises the outcome of a BPM/Key write-back operation.
type SyncResult struct {
	Total   int      `json:"total"`
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
}

// GetSyncableTracks returns all available, analyzed tracks that have BPM or
// key data from Engine DJ, ready for write-back to ID3 tags.
func GetSyncableTracks() ([]SyncableTrack, error) {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return nil, fmt.Errorf("no Engine Library found: %w", err)
	}

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro&cache=shared")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, COALESCE(title,''), COALESCE(artist,''), COALESCE(path,''),
			COALESCE(bpmAnalyzed, bpm, 0), COALESCE(key, 0)
		FROM Track
		WHERE isAvailable = 1 AND isAnalyzed = 1
			AND (bpmAnalyzed > 0 OR bpm > 0 OR key > 0)
		ORDER BY artist, title
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks: %w", err)
	}
	defer rows.Close()

	var tracks []SyncableTrack
	for rows.Next() {
		var t SyncableTrack
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Path,
			&t.BPM, &t.Key); err != nil {
			continue
		}
		t.BPM = math.Round(t.BPM*100) / 100 // round to 2 decimal places
		t.KeyName = KeyName(t.Key)
		t.Camelot = KeyCamelot(t.Key)
		tracks = append(tracks, t)
	}
	if tracks == nil {
		tracks = []SyncableTrack{}
	}
	return tracks, nil
}
