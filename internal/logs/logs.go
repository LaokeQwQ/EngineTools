package logs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type LogLevel string

const (
	LevelInfo    LogLevel = "I"
	LevelWarning LogLevel = "W"
	LevelError   LogLevel = "E"
)

type LogEntry struct {
	Level     LogLevel  `json:"level"`
	Timestamp time.Time `json:"timestamp"`
	Thread    string    `json:"thread"`
	Category  string    `json:"category"`
	Message   string    `json:"message"`
}

type LogStats struct {
	TotalFiles    int            `json:"totalFiles"`
	TotalLines    int            `json:"totalLines"`
	InfoCount     int            `json:"infoCount"`
	WarningCount  int            `json:"warningCount"`
	ErrorCount    int            `json:"errorCount"`
	LatestLog     string         `json:"latestLog"`
	TopWarnings   []MessageCount `json:"topWarnings"`
	TopErrors     []MessageCount `json:"topErrors"`
	RecentEntries []LogEntry     `json:"recentEntries"`
}

type MessageCount struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

var logPattern = regexp.MustCompile(`^\[([IWE ])\] (.+?)\s+\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\] \[([^\]]*)\] \[([^\]]*)\]$`)

func GetLogsDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "AIR Music Technology", "EnginePrime", "Logs")
}

func AnalyzeLogs() (*LogStats, error) {
	logsDir := GetLogsDir()

	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("logs directory not found: %s", logsDir)
	}

	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs directory: %w", err)
	}

	var logFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			logFiles = append(logFiles, filepath.Join(logsDir, entry.Name()))
		}
	}

	if len(logFiles) == 0 {
		return nil, fmt.Errorf("no log files found")
	}

	sort.Slice(logFiles, func(i, j int) bool {
		iInfo, _ := os.Stat(logFiles[i])
		jInfo, _ := os.Stat(logFiles[j])
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	stats := &LogStats{
		TotalFiles:    len(logFiles),
		TopWarnings:   []MessageCount{},
		TopErrors:     []MessageCount{},
		RecentEntries: []LogEntry{},
	}

	if len(logFiles) > 0 {
		stats.LatestLog = filepath.Base(logFiles[0])
	}

	warningCounts := make(map[string]int)
	errorCounts := make(map[string]int)

	maxFilesToAnalyze := 5
	if len(logFiles) < maxFilesToAnalyze {
		maxFilesToAnalyze = len(logFiles)
	}

	for i := 0; i < maxFilesToAnalyze; i++ {
		file, err := os.Open(logFiles[i])
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			stats.TotalLines++

			entry := parseLine(line)
			if entry == nil {
				continue
			}

			switch entry.Level {
			case LevelInfo:
				stats.InfoCount++
			case LevelWarning:
				stats.WarningCount++
				msg := normalizeMessage(entry.Message)
				warningCounts[msg]++
			case LevelError:
				stats.ErrorCount++
				msg := normalizeMessage(entry.Message)
				errorCounts[msg]++
			}

			if i == 0 && len(stats.RecentEntries) < 50 {
				if entry.Level == LevelWarning || entry.Level == LevelError {
					stats.RecentEntries = append(stats.RecentEntries, *entry)
				}
			}
		}

		file.Close()
	}

	stats.TopWarnings = topMessages(warningCounts, 10)
	stats.TopErrors = topMessages(errorCounts, 10)

	return stats, nil
}

func parseLine(line string) *LogEntry {
	matches := logPattern.FindStringSubmatch(line)
	if matches == nil || len(matches) < 6 {
		return nil
	}

	timestamp, err := time.Parse(time.RFC3339, matches[3])
	if err != nil {
		return nil
	}

	return &LogEntry{
		Level:     LogLevel(strings.TrimSpace(matches[1])),
		Message:   strings.TrimSpace(matches[2]),
		Timestamp: timestamp,
		Thread:    strings.TrimSpace(matches[4]),
		Category:  strings.TrimSpace(matches[5]),
	}
}

func normalizeMessage(msg string) string {
	msg = strings.TrimSpace(msg)

	msg = regexp.MustCompile(`0x[0-9a-fA-F]+`).ReplaceAllString(msg, "0x...")
	msg = regexp.MustCompile(`'[0-9a-f]{16}'`).ReplaceAllString(msg, "'...'")
	msg = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`).ReplaceAllString(msg, "YYYY-MM-DD")
	msg = regexp.MustCompile(`\d{2}:\d{2}:\d{2}`).ReplaceAllString(msg, "HH:MM:SS")

	if len(msg) > 200 {
		msg = msg[:200] + "..."
	}

	return msg
}

func topMessages(counts map[string]int, limit int) []MessageCount {
	var messages []MessageCount
	for msg, count := range counts {
		messages = append(messages, MessageCount{
			Message: msg,
			Count:   count,
		})
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Count > messages[j].Count
	})

	if len(messages) > limit {
		messages = messages[:limit]
	}

	return messages
}
