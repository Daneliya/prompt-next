package library

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Numbers remain reserved even after deletion, so undo and restoring a known
// entry do not change its number or allow another entry to use its directory.
func ensurePromptNumber(tx *sql.Tx, id string, preferred int64) (int64, error) {
	var number int64
	err := tx.QueryRow("SELECT number FROM prompt_numbers WHERE prompt_id = ?", id).Scan(&number)
	if !errors.Is(err, sql.ErrNoRows) {
		return number, err
	}
	if preferred > 0 && preferred <= 1_000_000_000 {
		result, err := tx.Exec("INSERT INTO prompt_numbers(number, prompt_id) VALUES (?, ?) ON CONFLICT(number) DO NOTHING", preferred, id)
		if err != nil {
			return 0, err
		}
		count, err := result.RowsAffected()
		if err != nil || count > 0 {
			return preferred, err
		}
	}
	result, err := tx.Exec("INSERT INTO prompt_numbers(prompt_id) VALUES (?)", id)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

const maxPromptNumber = 9_007_199_254_740_991 // JavaScript safe integer limit.

func (s *promptStore) addPromptNumbers(tx *sql.Tx) ([]string, error) {
	_, err := tx.Exec(`
		CREATE TABLE prompt_numbers (
			number INTEGER PRIMARY KEY AUTOINCREMENT,
			prompt_id TEXT NOT NULL UNIQUE
		);
	`)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(`INSERT INTO prompt_numbers(prompt_id)
		SELECT id FROM (
			SELECT id, json_extract(record, '$.createdAt') AS created FROM prompts
			UNION ALL
			SELECT id, json_extract(record, '$.createdAt') FROM deleted_prompts WHERE expires_at > ?
		) GROUP BY id ORDER BY min(created), id`, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	referenced, err := referencedImages(tx)
	if err != nil {
		return nil, err
	}
	legacy := map[string]bool{}
	for key := range referenced {
		name := filepath.Base(key)
		oldPath := filepath.Join(s.imagesDir(), name)
		info, err := os.Lstat(oldPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := s.copyImage(name, key, info); err != nil {
			return nil, err
		}
		legacy[name] = true
	}
	names := make([]string, 0, len(legacy))
	for name := range legacy {
		names = append(names, name)
	}
	return names, nil
}
