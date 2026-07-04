package database

import (
	"database/sql"
	"fmt"
)

// MissingTrack represents a track whose file is no longer available on disk.
type MissingTrack struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

// FindMissingTracks returns all tracks where isAvailable = 0.
func FindMissingTracks() ([]MissingTrack, error) {
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
		SELECT id, COALESCE(title,''), COALESCE(artist,''),
			COALESCE(path,''), COALESCE(filename,'')
		FROM Track
		WHERE isAvailable = 0
		ORDER BY artist, title
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query missing tracks: %w", err)
	}
	defer rows.Close()

	var tracks []MissingTrack
	for rows.Next() {
		var t MissingTrack
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Path, &t.Filename); err != nil {
			continue
		}
		t.Path = ResolveTrackPath(dbPath, t.Path)
		tracks = append(tracks, t)
	}
	if tracks == nil {
		tracks = []MissingTrack{}
	}
	return tracks, nil
}

// RemoveMissingTracks deletes the given track IDs from the database.
// Only tracks that are actually unavailable (isAvailable = 0) will be removed —
// available tracks are silently skipped for safety.
// Returns the number of rows deleted.
func RemoveMissingTracks(ids []int) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	dbPath, err := ResolveLibrary()
	if err != nil {
		return 0, fmt.Errorf("no Engine Library found: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	deleted := 0
	for _, id := range ids {
		res, err := db.Exec(`DELETE FROM Track WHERE id = ? AND isAvailable = 0`, id)
		if err != nil {
			continue
		}
		n, _ := res.RowsAffected()
		deleted += int(n)
	}
	return deleted, nil
}
