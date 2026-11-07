package library

import (
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type OrphanImage struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

type ImageCleanup struct {
	Removed int      `json:"removed"`
	Bytes   int64    `json:"bytes"`
	Skipped []string `json:"skipped"`
}

func referencedImages(tx *sql.Tx) (map[string]bool, error) {
	rows, err := tx.Query(`
		SELECT n.number || '/' || j.value
  FROM prompts p JOIN prompt_numbers n ON n.prompt_id = p.id, json_each(p.record,'$.images') j
  WHERE j.type = 'text'
  UNION
  SELECT n.number || '/' || j.value
  FROM deleted_prompts p JOIN prompt_numbers n ON n.prompt_id = p.id, json_each(p.record,'$.images') j
  WHERE j.type = 'text' AND p.expires_at > ?`, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if imageKey.MatchString(name) {
			names[name] = true
		}
	}
	return names, rows.Err()
}

func (s *Service) ListUnreferencedImages() ([]OrphanImage, error) {
	tx, err := s.store.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	referenced, err := referencedImages(tx)
	if err != nil {
		return nil, err
	}
	files := []OrphanImage{}
	err = s.store.walkImages(func(name string, info fs.FileInfo) {
		if !referenced[name] {
			files = append(files, OrphanImage{Name: name, Bytes: info.Size()})
		}
	})
	return files, err
}

func (s *Service) CleanUnreferencedImages(names []string) (ImageCleanup, error) {
	result := ImageCleanup{Skipped: []string{}}
	tx, err := s.store.db.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	referenced, err := referencedImages(tx)
	if err != nil {
		return result, err
	}
	s.store.thumbnailMu.Lock()
	defer s.store.thumbnailMu.Unlock()
	for _, name := range cleanTags(names) {
		if !imageKey.MatchString(name) || referenced[name] {
			result.Skipped = append(result.Skipped, name)
			continue
		}
		path, err := s.store.imagePath(name)
		if err != nil {
			result.Skipped = append(result.Skipped, name)
			continue
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || os.Remove(path) != nil {
			result.Skipped = append(result.Skipped, name)
			continue
		}
		thumb := s.store.thumbnailPath(name)
		_ = os.Remove(thumb)
		if filepath.Dir(path) != s.store.imagesDir() {
			_ = os.Remove(filepath.Dir(path))
			_ = os.Remove(filepath.Dir(thumb))
		}
		result.Removed++
		result.Bytes += info.Size()
	}
	if _, err := tx.Exec("DELETE FROM deleted_prompts WHERE expires_at <= ?", time.Now().Unix()); err != nil {
		return result, err
	}
	return result, tx.Commit()
}

func (s *promptStore) walkImages(visit func(string, fs.FileInfo)) error {
	root := s.imagesDir()
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if path == root && errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if entry.IsDir() {
			if !imageFolder.MatchString(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || !imageKey.MatchString(name) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		visit(name, info)
		return nil
	})
}
