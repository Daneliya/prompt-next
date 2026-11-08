package library

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type TrashRef struct {
	Batch string `json:"batch"`
	ID    string `json:"id"`
}

type TrashEntry struct {
	Ref       TrashRef `json:"ref"`
	Title     string   `json:"title"`
	Kind      string   `json:"kind"`
	Number    int64    `json:"number"`
	ExpiresAt int64    `json:"expiresAt"`
}

func (s *Service) ListTrash() ([]TrashEntry, error) {
	rows, err := s.store.db.Query(`SELECT d.batch, d.id, json_extract(d.record, '$.title'),
		json_extract(d.record, '$.kind'), n.number, d.expires_at
		FROM deleted_prompts d JOIN prompt_numbers n ON n.prompt_id = d.id
		WHERE d.expires_at > ? ORDER BY d.expires_at DESC, d.rowid DESC`, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []TrashEntry{}
	for rows.Next() {
		var entry TrashEntry
		if err := rows.Scan(&entry.Ref.Batch, &entry.Ref.ID, &entry.Title, &entry.Kind, &entry.Number, &entry.ExpiresAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Service) RestoreTrash(refs []TrashRef) (int, error) {
	tx, err := s.store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	restored := 0
	for _, ref := range refs {
		result, err := tx.Exec(`INSERT INTO prompts (id,kind,updated_at,record)
			SELECT id,json_extract(record,'$.kind'),json_extract(record,'$.updatedAt'),record
			FROM deleted_prompts WHERE batch = ? AND id = ? AND expires_at > ?
			ON CONFLICT(id) DO NOTHING`, ref.Batch, ref.ID, now)
		if err != nil {
			return 0, err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		if count == 0 {
			continue
		}
		if _, err := tx.Exec("DELETE FROM deleted_prompts WHERE batch = ? AND id = ?", ref.Batch, ref.ID); err != nil {
			return 0, err
		}
		restored++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return restored, nil
}

type TrashDeletion struct {
	Deleted        int  `json:"deleted"`
	CleanupPending bool `json:"cleanupPending"`
}

func (s *Service) DeleteTrashPermanently(refs []TrashRef) (TrashDeletion, error) {
	tx, err := s.store.db.Begin()
	if err != nil {
		return TrashDeletion{}, err
	}
	defer tx.Rollback()
	removed := []Prompt{}
	for _, ref := range refs {
		var record string
		err := tx.QueryRow("DELETE FROM deleted_prompts WHERE batch = ? AND id = ? RETURNING record", ref.Batch, ref.ID).Scan(&record)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return TrashDeletion{}, err
		}
		var prompt Prompt
		if err := json.Unmarshal([]byte(record), &prompt); err != nil {
			return TrashDeletion{}, err
		}
		if err := tx.QueryRow("SELECT number FROM prompt_numbers WHERE prompt_id = ?", ref.ID).Scan(&prompt.Number); err != nil {
			return TrashDeletion{}, err
		}
		prompt.ID = ref.ID
		removed = append(removed, prompt)
	}
	if err := tx.Commit(); err != nil {
		return TrashDeletion{}, err
	}
	result := TrashDeletion{Deleted: len(removed)}
	if result.Deleted > 0 {
		result.CleanupPending = s.store.removeTrashMedia(removed) != nil
	}
	return result, nil
}

func (s *promptStore) removeTrashMedia(removed []Prompt) error {
	// Recheck after the deletion commits, holding the write lock against import and restore.
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	s.thumbnailMu.Lock()
	defer s.thumbnailMu.Unlock()

	var cleanupErr error
	for _, prompt := range removed {
		images, videos, err := remainingTrashMedia(tx, prompt.ID)
		if err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
			continue
		}
		for _, name := range prompt.imageNames() {
			if images[name] {
				continue
			}
			key := scopedImage(prompt.Number, name)
			path, err := s.imagePath(key)
			if err == nil {
				err = removeTrashFile(path)
			}
			if err == nil {
				_ = os.Remove(s.thumbnailPath(key))
			}
			cleanupErr = errors.Join(cleanupErr, err)
		}
		for _, name := range prompt.videoNames() {
			if videos[name] {
				continue
			}
			key := scopedImage(prompt.Number, name)
			path, err := s.videoPath(key)
			if err == nil {
				err = removeTrashFile(path)
			}
			if err == nil {
				_ = os.Remove(s.thumbnailPath(key))
			}
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
}

func remainingTrashMedia(tx *sql.Tx, id string) (map[string]bool, map[string]bool, error) {
	rows, err := tx.Query("SELECT record FROM prompts WHERE id = ? UNION ALL SELECT record FROM deleted_prompts WHERE id = ? AND expires_at > ?", id, id, time.Now().Unix())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	images, videos := map[string]bool{}, map[string]bool{}
	for rows.Next() {
		var record string
		if err := rows.Scan(&record); err != nil {
			return nil, nil, err
		}
		var prompt Prompt
		if err := json.Unmarshal([]byte(record), &prompt); err != nil {
			return nil, nil, err
		}
		for _, name := range prompt.imageNames() {
			images[name] = true
		}
		for _, name := range prompt.videoNames() {
			videos[name] = true
		}
	}
	return images, videos, rows.Err()
}

func removeTrashFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("附件路径不是普通文件")
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_ = os.Remove(filepath.Dir(path))
	return nil
}
