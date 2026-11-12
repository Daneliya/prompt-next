package library

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
)

func (s *Service) DeletePrompts(ids []string) (string, error) {
	batch := rand.Text()
	expires := time.Now().Add(24 * time.Hour).Unix()
	err := s.changePrompts(ids, func(tx *sql.Tx, id string) error {
		if err := requireChanged(tx.Exec(`INSERT INTO deleted_prompts (batch,id,expires_at,record)
			SELECT ?,id,?,record FROM prompts WHERE id = ?`, batch, expires, id)); err != nil {
			return err
		}
		_, err := tx.Exec("DELETE FROM prompts WHERE id = ?", id)
		return err
	})
	if err != nil {
		return "", err
	}
	return batch, nil
}

func (s *Service) UndoDelete(batch string) error {
	tx, err := s.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`INSERT INTO prompts (id,kind,updated_at,record)
		SELECT id,json_extract(record,'$.kind'),json_extract(record,'$.updatedAt'),record
		FROM deleted_prompts WHERE batch = ? AND expires_at > ?`, batch, time.Now().Unix())
	if err != nil {
		return errors.New("无法撤销删除，原条目可能已经导入或恢复")
	}
	if err := requireChanged(result, nil); err != nil {
		return errors.New("撤销已过期或记录已恢复")
	}
	if _, err := tx.Exec("DELETE FROM deleted_prompts WHERE batch = ?", batch); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) RecentDeleteBatch() (string, error) {
	var batch string
	err := s.store.db.QueryRow("SELECT batch FROM deleted_prompts WHERE expires_at > ? ORDER BY expires_at DESC, rowid DESC LIMIT 1", time.Now().Unix()).Scan(&batch)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return batch, err
}

func (s *Service) changePrompts(ids []string, change func(*sql.Tx, string) error) error {
	if len(ids) == 0 {
		return errors.New("请先选择提示词")
	}
	tx, err := s.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		if err := change(tx, id); err != nil {
			return err
		}
		seen[id] = true
	}
	return tx.Commit()
}

func (s *Service) BatchFavorite(ids []string, favorite bool) error {
	value := "false"
	if favorite {
		value = "true"
	}
	return s.changePrompts(ids, func(tx *sql.Tx, id string) error {
		return requireChanged(tx.Exec("UPDATE prompts SET record = json_set(record, '$.favorite', json(?)) WHERE id = ?", value, id))
	})
}

func (s *Service) BatchCategory(ids []string, kind, category string) error {
	defaults := defaultCategories(kind)
	if defaults == nil {
		return errors.New("提示词库类型无效")
	}
	now := timestamp()
	checked := false
	return s.changePrompts(ids, func(tx *sql.Tx, id string) error {
		if !checked && category != "" {
			custom, err := s.customCategories(tx, kind)
			if err != nil {
				return err
			}
			if !slices.Contains(append(defaults, custom...), category) {
				return errors.New("请选择当前提示词库的分类")
			}
			checked = true
		}
		return requireChanged(tx.Exec(`UPDATE prompts SET updated_at = ?, record = json_set(record, '$.category', ?, '$.updatedAt', ?)
   WHERE id = ? AND kind = ?`, now, category, now, id, kind))
	})
}

func (s *Service) RenameTag(kind, current, name string) error {
	name = strings.TrimSpace(name)
	if defaultCategories(kind) == nil || current == "" || name == "" || len([]rune(name)) > 60 || strings.ContainsAny(name, ",，") {
		return errors.New("标签须为 1 至 60 个字符，且不能包含逗号")
	}
	tx, err := s.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	items, err := queryPrompts(tx, " WHERE p.kind = ? AND EXISTS (SELECT 1 FROM json_each(p.record, '$.tags') WHERE value = ?)", kind, current)
	if err != nil {
		return err
	}
	for _, item := range items {
		for i, tag := range item.Tags {
			if tag == current || strings.EqualFold(tag, name) {
				item.Tags[i] = name
			}
		}
		item.Tags = cleanTags(item.Tags)
		now := timestamp()
		if _, err := tx.Exec("UPDATE prompts SET updated_at = ?, record = json_set(record, '$.tags', json(?), '$.updatedAt', ?) WHERE id = ?",
			now, stringsJSON(item.Tags), now, item.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Service) VariablePresets(id string) (map[string]map[string]string, error) {
	return variablePresets(s.store.db, id)
}

func variablePresets(db settingsReader, id string) (map[string]map[string]string, error) {
	raw, err := getSetting(db, "variables:"+id)
	presets := map[string]map[string]string{}
	if err != nil || raw == "" {
		return presets, err
	}
	err = json.Unmarshal([]byte(raw), &presets)
	if presets == nil {
		presets = map[string]map[string]string{}
	}
	return presets, err
}

func (s *Service) SetVariablePreset(id, name string, values map[string]string) error {
	name = strings.TrimSpace(name)
	if id == "" || name == "" || len([]rune(name)) > 60 {
		return errors.New("预设名称须为 1 至 60 个字符")
	}
	tx, err := s.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRow("SELECT 1 FROM prompts WHERE id = ?", id).Scan(&exists); err != nil {
		return err
	}
	presets, err := variablePresets(tx, id)
	if err != nil {
		return err
	}
	if values == nil {
		delete(presets, name)
	} else {
		presets[name] = values
	}
	raw, err := json.Marshal(presets)
	if err != nil {
		return err
	}
	if err := putSetting(tx, "variables:"+id, string(raw)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) DeleteVariablePreset(id, name string) error {
	return s.SetVariablePreset(id, name, nil)
}
