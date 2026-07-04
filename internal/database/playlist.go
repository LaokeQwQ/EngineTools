package database

import (
	"database/sql"
	"fmt"
)

type PlaylistInfo struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	ParentID int    `json:"parentId"`
	Count    int    `json:"count"`
}

type TrackInfo struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Album    string  `json:"album"`
	Genre    string  `json:"genre"`
	BPM      float64 `json:"bpm"`
	Length   int     `json:"length"`
	Filename string  `json:"filename"`
	Key      int     `json:"key"`
	KeyName  string  `json:"keyName"`
	Camelot  string  `json:"camelot"`
	Rating   int     `json:"rating"`
}

func ListPlaylists() ([]PlaylistInfo, error) {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro&cache=shared")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT p.id, p.title, p.parentListId,
			COALESCE((SELECT COUNT(*) FROM PlaylistEntity pe WHERE pe.listId = p.id), 0) as trackCount
		FROM Playlist p
		ORDER BY p.title
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query playlists: %w", err)
	}
	defer rows.Close()

	var playlists []PlaylistInfo
	for rows.Next() {
		var p PlaylistInfo
		if err := rows.Scan(&p.ID, &p.Title, &p.ParentID, &p.Count); err != nil {
			continue
		}
		playlists = append(playlists, p)
	}

	if playlists == nil {
		playlists = []PlaylistInfo{}
	}
	return playlists, nil
}

func GetPlaylistTracks(playlistID int) ([]TrackInfo, error) {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro&cache=shared")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT t.id, COALESCE(t.title,''), COALESCE(t.artist,''), COALESCE(t.album,''),
			COALESCE(t.genre,''), COALESCE(t.bpmAnalyzed, t.bpm, 0), COALESCE(t.length, 0),
			COALESCE(t.filename,''), COALESCE(t.key, 0), COALESCE(t.rating, 0)
		FROM PlaylistEntity pe
		JOIN Track t ON pe.trackId = t.id
		WHERE pe.listId = ?
		ORDER BY pe.id
	`, playlistID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks: %w", err)
	}
	defer rows.Close()

	var tracks []TrackInfo
	for rows.Next() {
		var t TrackInfo
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.Genre, &t.BPM, &t.Length, &t.Filename, &t.Key, &t.Rating); err != nil {
			continue
		}
		t.KeyName = KeyName(t.Key)
		t.Camelot = KeyCamelot(t.Key)
		tracks = append(tracks, t)
	}

	if tracks == nil {
		tracks = []TrackInfo{}
	}
	return tracks, nil
}
