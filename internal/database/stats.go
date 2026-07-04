package database

import (
	"database/sql"
	"fmt"
)

// LibraryStats holds aggregate statistics about the Engine Library.
type LibraryStats struct {
	TotalTracks     int             `json:"totalTracks"`
	TotalDuration   int             `json:"totalDuration"`   // seconds
	TotalFileBytes  int64           `json:"totalFileBytes"`  // bytes
	AnalyzedTracks  int             `json:"analyzedTracks"`
	MissingTracks   int             `json:"missingTracks"`
	NeverPlayed     int             `json:"neverPlayed"`
	FileTypes       []KVCount       `json:"fileTypes"`
	TopGenres       []KVCount       `json:"topGenres"`
	BPMDistribution []BPMBucket     `json:"bpmDistribution"`
	RecentlyAdded   []TrackSummary  `json:"recentlyAdded"`
}

// PlayStats holds play-history statistics.
type PlayStats struct {
	MostPlayed    []TrackSummary `json:"mostPlayed"`
	RecentPlayed  []TrackSummary `json:"recentPlayed"`
	NeverPlayed   []TrackSummary `json:"neverPlayed"`
}

type KVCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type BPMBucket struct {
	Range string `json:"range"`
	Count int    `json:"count"`
}

type TrackSummary struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	BPM         float64 `json:"bpm"`
	Key         int    `json:"key"`
	KeyName     string `json:"keyName"`
	PlayCount   int    `json:"playCount"`
	LastPlayed  string `json:"lastPlayed"`
	DateAdded   string `json:"dateAdded"`
}

// GetLibraryStats returns aggregate statistics for the active library.
func GetLibraryStats() (*LibraryStats, error) {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return nil, fmt.Errorf("no Engine Library found: %w", err)
	}

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro&cache=shared")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	stats := &LibraryStats{
		FileTypes:       []KVCount{},
		TopGenres:       []KVCount{},
		BPMDistribution: []BPMBucket{},
		RecentlyAdded:   []TrackSummary{},
	}

	// Totals
	err = db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(length), 0),
			COALESCE(SUM(fileBytes), 0),
			COALESCE(SUM(CASE WHEN isAnalyzed = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN isAvailable = 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN playedIndicator = 0 THEN 1 ELSE 0 END), 0)
		FROM Track
	`).Scan(&stats.TotalTracks, &stats.TotalDuration, &stats.TotalFileBytes,
		&stats.AnalyzedTracks, &stats.MissingTracks, &stats.NeverPlayed)
	if err != nil {
		return nil, fmt.Errorf("failed to query totals: %w", err)
	}

	// File type breakdown
	rows, err := db.Query(`
		SELECT UPPER(COALESCE(fileType, 'Unknown')), COUNT(*)
		FROM Track
		GROUP BY fileType
		ORDER BY COUNT(*) DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var kv KVCount
			if rows.Scan(&kv.Key, &kv.Count) == nil {
				stats.FileTypes = append(stats.FileTypes, kv)
			}
		}
	}

	// Top genres
	rows2, err := db.Query(`
		SELECT COALESCE(NULLIF(genre,''), 'Unknown'), COUNT(*)
		FROM Track
		GROUP BY genre
		ORDER BY COUNT(*) DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var kv KVCount
			if rows2.Scan(&kv.Key, &kv.Count) == nil {
				stats.TopGenres = append(stats.TopGenres, kv)
			}
		}
	}

	// BPM distribution (buckets of 10)
	bpmRanges := [][2]int{{60, 70}, {70, 80}, {80, 90}, {90, 100}, {100, 110},
		{110, 120}, {120, 130}, {130, 140}, {140, 150}, {150, 160}, {160, 180}}
	for _, r := range bpmRanges {
		var count int
		db.QueryRow(`
			SELECT COUNT(*) FROM Track
			WHERE bpmAnalyzed >= ? AND bpmAnalyzed < ?
		`, r[0], r[1]).Scan(&count)
		if count > 0 {
			stats.BPMDistribution = append(stats.BPMDistribution, BPMBucket{
				Range: fmt.Sprintf("%d-%d", r[0], r[1]),
				Count: count,
			})
		}
	}

	// Recently added (last 30 tracks by dateAdded)
	rows3, err := db.Query(`
		SELECT id, COALESCE(title,''), COALESCE(artist,''),
			COALESCE(bpmAnalyzed, bpm, 0), COALESCE(key, 0),
			COALESCE(dateAdded, '')
		FROM Track
		WHERE isAvailable = 1
		ORDER BY dateAdded DESC
		LIMIT 20
	`)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var t TrackSummary
			if rows3.Scan(&t.ID, &t.Title, &t.Artist, &t.BPM, &t.Key, &t.DateAdded) == nil {
				t.KeyName = KeyName(t.Key)
				stats.RecentlyAdded = append(stats.RecentlyAdded, t)
			}
		}
	}

	return stats, nil
}

// GetPlayStats returns play history statistics.
func GetPlayStats() (*PlayStats, error) {
	dbPath, err := ResolveLibrary()
	if err != nil {
		return nil, fmt.Errorf("no Engine Library found: %w", err)
	}

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro&cache=shared")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	result := &PlayStats{
		MostPlayed:   []TrackSummary{},
		RecentPlayed: []TrackSummary{},
		NeverPlayed:  []TrackSummary{},
	}

	scanTracks := func(rows *sql.Rows) []TrackSummary {
		var list []TrackSummary
		for rows.Next() {
			var t TrackSummary
			if rows.Scan(&t.ID, &t.Title, &t.Artist, &t.BPM, &t.Key,
				&t.PlayCount, &t.LastPlayed) == nil {
				t.KeyName = KeyName(t.Key)
				list = append(list, t)
			}
		}
		if list == nil {
			list = []TrackSummary{}
		}
		return list
	}

	// Most played
	if rows, err := db.Query(`
		SELECT id, COALESCE(title,''), COALESCE(artist,''),
			COALESCE(bpmAnalyzed, bpm, 0), COALESCE(key, 0),
			COALESCE(playedIndicator, 0), COALESCE(timeLastPlayed, '')
		FROM Track
		WHERE isAvailable = 1 AND playedIndicator > 0
		ORDER BY playedIndicator DESC
		LIMIT 20
	`); err == nil {
		defer rows.Close()
		result.MostPlayed = scanTracks(rows)
	}

	// Recently played
	if rows, err := db.Query(`
		SELECT id, COALESCE(title,''), COALESCE(artist,''),
			COALESCE(bpmAnalyzed, bpm, 0), COALESCE(key, 0),
			COALESCE(playedIndicator, 0), COALESCE(timeLastPlayed, '')
		FROM Track
		WHERE isAvailable = 1 AND timeLastPlayed IS NOT NULL AND timeLastPlayed != ''
		ORDER BY timeLastPlayed DESC
		LIMIT 20
	`); err == nil {
		defer rows.Close()
		result.RecentPlayed = scanTracks(rows)
	}

	// Never played
	if rows, err := db.Query(`
		SELECT id, COALESCE(title,''), COALESCE(artist,''),
			COALESCE(bpmAnalyzed, bpm, 0), COALESCE(key, 0),
			0, ''
		FROM Track
		WHERE isAvailable = 1 AND (playedIndicator IS NULL OR playedIndicator = 0)
		ORDER BY dateAdded DESC
		LIMIT 50
	`); err == nil {
		defer rows.Close()
		result.NeverPlayed = scanTracks(rows)
	}

	return result, nil
}
