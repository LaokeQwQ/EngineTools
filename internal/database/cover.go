package database

import (
	"database/sql"
	"fmt"
)

// TrackPath is the minimal data needed for cover-art processing.
type TrackPath struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

// GetAllTrackPaths returns the file paths of every available, analyzed track.
func GetAllTrackPaths() ([]TrackPath, error) {
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
		SELECT id, COALESCE(path,'')
		FROM Track
		WHERE isAvailable = 1 AND path != ''
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var tracks []TrackPath
	for rows.Next() {
		var t TrackPath
		if rows.Scan(&t.ID, &t.Path) == nil && t.Path != "" {
			tracks = append(tracks, t)
		}
	}
	if tracks == nil {
		tracks = []TrackPath{}
	}
	return tracks, nil
}

// GetPlaylistTrackPaths returns file paths for all tracks in the given playlist.
func GetPlaylistTrackPaths(playlistID int) ([]TrackPath, error) {
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
		SELECT t.id, COALESCE(t.path,'')
		FROM PlaylistEntity pe
		JOIN Track t ON pe.trackId = t.id
		WHERE pe.listId = ? AND t.isAvailable = 1 AND t.path != ''
		ORDER BY pe.id
	`, playlistID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var tracks []TrackPath
	for rows.Next() {
		var t TrackPath
		if rows.Scan(&t.ID, &t.Path) == nil && t.Path != "" {
			tracks = append(tracks, t)
		}
	}
	if tracks == nil {
		tracks = []TrackPath{}
	}
	return tracks, nil
}
