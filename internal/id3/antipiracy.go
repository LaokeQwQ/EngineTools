package id3

import (
	"EngineTools/internal/storage"
	"crypto/sha256"
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

func backupPath(dir string) (string, error) {
	return backupKeyPath(dir)
}

// backupKeyPath returns the backup file path given an arbitrary key.
// All new backups live beside the executable. Directory keys include a hash
// so different music directories with the same name cannot collide.
func backupKeyPath(key string) (string, error) {
	dir := storage.Path("Backup", "ID3")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if info, err := os.Stat(key); err == nil && info.IsDir() {
		absolute, err := filepath.Abs(key)
		if err != nil {
			return "", err
		}
		digest := sha256.Sum256([]byte(filepath.Clean(absolute)))
		target := filepath.Join(dir, fmt.Sprintf("directory-%x_backup.json", digest[:12]))
		// Import a legacy per-directory backup without removing the original.
		if err := storage.Migrate(filepath.Join(absolute, ".id3_backup.json"), target); err != nil {
			return "", err
		}
		return target, nil
	}
	safe := filepath.Base(key)
	if safe == "" || safe == "." {
		safe = "default"
	}
	return filepath.Join(dir, safe+"_backup.json"), nil
}

// AntiPiracyV1Files clears all ID3 tags for the given list of file paths.
// A backup is stored using backupKey as the identifier.
func AntiPiracyV1Files(paths []string, backupKey string) (int, error) {
	if len(paths) == 0 {
		return 0, fmt.Errorf("no files provided")
	}
	var backups []TagBackup
	for _, f := range paths {
		info, err := ReadTag(f)
		if err != nil {
			continue
		}
		backups = append(backups, TagBackup{FilePath: f, Title: info.Title, Artist: info.Artist, Album: info.Album, Year: info.Year, Genre: info.Genre})
	}
	bd := BackupData{Mode: "v1", Timestamp: time.Now().Format(time.RFC3339), Dir: backupKey, Tags: backups}
	data, _ := json.MarshalIndent(bd, "", "  ")
	path, err := backupKeyPath(backupKey)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return 0, fmt.Errorf("backup failed: %w", err)
	}
	count := 0
	for _, f := range paths {
		if err := ClearAllTags(f); err == nil {
			count++
		}
	}
	return count, nil
}

// AntiPiracyV2Files shuffles ID3 tags among the given file paths.
func AntiPiracyV2Files(paths []string, backupKey string) (int, error) {
	if len(paths) == 0 {
		return 0, fmt.Errorf("no files provided")
	}
	var tags []TagBackup
	for _, f := range paths {
		info, err := ReadTag(f)
		if err != nil {
			continue
		}
		tags = append(tags, TagBackup{FilePath: f, Title: info.Title, Artist: info.Artist, Album: info.Album, Year: info.Year, Genre: info.Genre})
	}
	bd := BackupData{Mode: "v2", Timestamp: time.Now().Format(time.RFC3339), Dir: backupKey, Tags: tags}
	data, _ := json.MarshalIndent(bd, "", "  ")
	path, err := backupKeyPath(backupKey)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return 0, fmt.Errorf("backup failed: %w", err)
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	shuffled := make([]TagBackup, len(tags))
	copy(shuffled, tags)
	for i := len(shuffled) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	count := 0
	for i, f := range paths {
		if i >= len(shuffled) {
			break
		}
		s := shuffled[i]
		if err := WriteTag(f, s.Title, s.Artist, s.Album, s.Year, s.Genre); err == nil {
			count++
		}
	}
	return count, nil
}

// AntiPiracyRestoreFiles restores tags from the backup identified by backupKey.
func AntiPiracyRestoreFiles(backupKey string) (int, error) {
	path, err := backupKeyPath(backupKey)
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("no backup found for key %q", backupKey)
	}
	var bd BackupData
	if err := json.Unmarshal(data, &bd); err != nil {
		return 0, fmt.Errorf("backup corrupted: %w", err)
	}
	count := 0
	for _, t := range bd.Tags {
		if err := WriteTag(t.FilePath, t.Title, t.Artist, t.Album, t.Year, t.Genre); err == nil {
			count++
		}
	}
	os.Remove(path)
	return count, nil
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
	path, err := backupPath(dir)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
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
	path, err := backupPath(dir)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
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
	bp, err := backupPath(dir)
	if err != nil {
		return 0, err
	}
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
	path, err := backupPath(dir)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
