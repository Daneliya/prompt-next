package library

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type DailyPromptCount struct {
	Date    string `json:"date"`
	Count   int    `json:"count"`
	Text    int    `json:"text"`
	Art     int    `json:"art"`
	Video   int    `json:"video"`
	Persona int    `json:"persona"`
}

type StorageStats struct {
	RecentPrompts   []DailyPromptCount `json:"recentPrompts"`
	LocationWarning string             `json:"locationWarning"`
	Directory       string             `json:"directory"`
	DBBytes         int64              `json:"dbBytes"`
	ImageBytes      int64              `json:"imageBytes"`
	ImageFiles      int                `json:"imageFiles"`
	VideoBytes      int64              `json:"videoBytes"`
	VideoFiles      int                `json:"videoFiles"`
}

func (s *Service) StorageStats() (StorageStats, error) {
	return s.storageStatsAt(time.Now())
}

func (s *Service) storageStatsAt(now time.Time) (StorageStats, error) {
	return s.storageSnapshotAt(now, true)
}

func (s *Service) StorageSnapshot(includeMedia bool) (StorageStats, error) {
	return s.storageSnapshotAt(time.Now(), includeMedia)
}

func (s *Service) storageSnapshotAt(now time.Time, includeMedia bool) (StorageStats, error) {
	const days = 14
	stats := StorageStats{Directory: s.store.dir, LocationWarning: s.locationWarning, RecentPrompts: make([]DailyPromptCount, days)}
	dateIndex := make(map[string]int, days)
	for index := range stats.RecentPrompts {
		date := now.AddDate(0, 0, index-days+1).Format(time.DateOnly)
		stats.RecentPrompts[index].Date = date
		dateIndex[date] = index
	}
	for _, name := range []string{"prompts.db", "prompts.db-wal", "prompts.db-shm", "prompts.db-journal"} {
		if info, err := os.Stat(filepath.Join(s.store.dir, name)); err == nil {
			stats.DBBytes += info.Size()
		} else if !errors.Is(err, os.ErrNotExist) {
			return stats, err
		}
	}
	rows, err := s.store.db.Query("SELECT kind, coalesce(json_extract(record, '$.createdAt'), '') FROM prompts")
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, created string
		if err := rows.Scan(&kind, &created); err != nil {
			return stats, err
		}
		date, err := time.Parse(time.RFC3339Nano, normalizeDate(created, ""))
		if err == nil {
			if index, ok := dateIndex[date.In(now.Location()).Format(time.DateOnly)]; ok {
				day := &stats.RecentPrompts[index]
				day.Count++
				switch kind {
				case "text":
					day.Text++
				case "art":
					day.Art++
				case "video":
					day.Video++
				case "persona":
					day.Persona++
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return stats, err
	}
	if !includeMedia {
		return stats, nil
	}
	err = s.store.walkImages(func(_ string, info fs.FileInfo) {
		stats.ImageBytes += info.Size()
		stats.ImageFiles++
	})
	if err != nil {
		return stats, err
	}
	err = walkScopedMedia(filepath.Join(s.store.dir, "videos"), videoKey.MatchString, func(_ string, info fs.FileInfo) {
		stats.VideoBytes += info.Size()
		stats.VideoFiles++
	})
	return stats, err
}
