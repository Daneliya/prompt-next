package library

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type DailyImageCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type StorageStats struct {
	RecentImages []DailyImageCount `json:"recentImages"`
	Directory    string            `json:"directory"`
	DBBytes      int64             `json:"dbBytes"`
	ImageBytes   int64             `json:"imageBytes"`
	ImageFiles   int               `json:"imageFiles"`
}

func (s *Service) StorageStats() (StorageStats, error) {
	return s.storageStatsAt(time.Now())
}

func (s *Service) storageStatsAt(now time.Time) (StorageStats, error) {
	const days = 14
	stats := StorageStats{Directory: s.store.dir, RecentImages: make([]DailyImageCount, days)}
	dateIndex := make(map[string]int, days)
	for index := range stats.RecentImages {
		date := now.AddDate(0, 0, index-days+1).Format(time.DateOnly)
		stats.RecentImages[index].Date = date
		dateIndex[date] = index
	}
	for _, name := range []string{"prompts.db", "prompts.db-wal", "prompts.db-shm", "prompts.db-journal"} {
		if info, err := os.Stat(filepath.Join(s.store.dir, name)); err == nil {
			stats.DBBytes += info.Size()
		} else if !errors.Is(err, os.ErrNotExist) {
			return stats, err
		}
	}
	firstSaved := make(map[string]time.Time)
	err := s.store.walkImages(func(key string, info fs.FileInfo) {
		stats.ImageBytes += info.Size()
		stats.ImageFiles++
		name := filepath.Base(key)
		if prior, ok := firstSaved[name]; !ok || info.ModTime().Before(prior) {
			firstSaved[name] = info.ModTime()
		}
	})
	for _, date := range firstSaved {
		if index, ok := dateIndex[date.In(now.Location()).Format(time.DateOnly)]; ok {
			stats.RecentImages[index].Count++
		}
	}
	return stats, err
}
