package id3

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type TagBackup struct {
	FilePath string `json:"filePath"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Year     string `json:"year"`
	Genre    string `json:"genre"`
}

type BackupData struct {
	Mode      string      `json:"mode"`
	Timestamp string      `json:"timestamp"`
	Dir       string      `json:"dir"`
	Tags      []TagBackup `json:"tags"`
}

func backupPath(dir string) string {
	return filepath.Join(dir, ".id3_backup.json")
}

func AntiPiracyV1(dir string) (int, error) {
	files, err := ListAudioFiles(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to list audio files: %w", err)
	}
	if len(files) == 0 {
		return 0, fmt.Errorf("no audio files found in directory")
	}

	var backups []TagBackup
	for _, f := range files {
		info, err := ReadTag(f)
		if err != nil {
			continue
		}
		backups = append(backups, TagBackup{
			FilePath: f,
			Title:    info.Title,
			Artist:   info.Artist,
			Album:    info.Album,
			Year:     info.Year,
			Genre:    info.Genre,
		})
	}

	bd := BackupData{
		Mode:      "v1",
		Timestamp: time.Now().Format(time.RFC3339),
		Dir:       dir,
		Tags:      backups,
	}
	data, _ := json.MarshalIndent(bd, "", "  ")
	if err := os.WriteFile(backupPath(dir), data, 0644); err != nil {
		return 0, fmt.Errorf("failed to save backup: %w", err)
	}

	count := 0
	for _, f := range files {
		if err := ClearAllTags(f); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

func AntiPiracyV2(dir string) (int, error) {
	files, err := ListAudioFiles(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to list audio files: %w", err)
	}
	if len(files) == 0 {
		return 0, fmt.Errorf("no audio files found in directory")
	}

	var backups []TagBackup
	var tags []TagBackup
	for _, f := range files {
		info, err := ReadTag(f)
		if err != nil {
			continue
		}
		b := TagBackup{
			FilePath: f,
			Title:    info.Title,
			Artist:   info.Artist,
			Album:    info.Album,
			Year:     info.Year,
			Genre:    info.Genre,
		}
		backups = append(backups, b)
		tags = append(tags, b)
	}

	bd := BackupData{
		Mode:      "v2",
		Timestamp: time.Now().Format(time.RFC3339),
		Dir:       dir,
		Tags:      backups,
	}
	data, _ := json.MarshalIndent(bd, "", "  ")
	if err := os.WriteFile(backupPath(dir), data, 0644); err != nil {
		return 0, fmt.Errorf("failed to save backup: %w", err)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := len(tags)
	for i := n - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		tags[i], tags[j] = tags[j], tags[i]
	}

	count := 0
	for i, f := range files {
		if i >= len(tags) {
			break
		}
		shuffled := tags[i]
		err := WriteTag(f, shuffled.Title, shuffled.Artist, shuffled.Album, shuffled.Year, shuffled.Genre)
		if err != nil {
			continue
		}
		count++
	}

	return count, nil
}

func AntiPiracyRestore(dir string) (int, error) {
	bp := backupPath(dir)
	data, err := os.ReadFile(bp)
	if err != nil {
		return 0, fmt.Errorf("no backup found in this directory")
	}

	var bd BackupData
	if err := json.Unmarshal(data, &bd); err != nil {
		return 0, fmt.Errorf("backup file corrupted: %w", err)
	}

	count := 0
	for _, t := range bd.Tags {
		if err := WriteTag(t.FilePath, t.Title, t.Artist, t.Album, t.Year, t.Genre); err != nil {
			continue
		}
		count++
	}

	os.Remove(bp)
	return count, nil
}

func HasAntiPiracyBackup(dir string) bool {
	_, err := os.Stat(backupPath(dir))
	return err == nil
}
