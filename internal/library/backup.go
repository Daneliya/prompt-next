package library

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Daneliya/prompt-next/internal/fileio"
)

const backupFormat = "bigfish-prompt-manager"

type ImportSummary struct {
	Added      int    `json:"added"`
	Duplicate  int    `json:"duplicate"`
	Invalid    int    `json:"invalid"`
	CanRestore bool   `json:"canRestore"`
	Snapshot   string `json:"snapshot"`
}

type backupFile struct {
	Format   string             `json:"format"`
	Version  int                `json:"version"`
	Items    []Prompt           `json:"items"`
	Images   map[string][]byte  `json:"images,omitempty"`
	Settings *map[string]string `json:"settings,omitempty"`
	Partial  bool               `json:"partial,omitempty"`
}

func encodeBackup(items []Prompt) ([]byte, error) {
	return json.MarshalIndent(backupFile{Format: backupFormat, Version: 1, Items: items}, "", "  ")
}

func BackupJSON(s *Service) ([]byte, error) {
	return s.export(nil)
}

func SelectedJSON(s *Service, ids []string) ([]byte, error) {
	if len(ids) == 0 {
		return nil, errors.New("请先选择提示词")
	}
	return s.export(ids)
}

func (s *Service) export(ids []string) ([]byte, error) {
	tx, err := s.store.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	items, err := listPrompts(tx)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		return s.store.snapshot(tx, items)
	}
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	selected := []Prompt{}
	for _, item := range items {
		if wanted[item.ID] {
			selected = append(selected, item)
		}
	}
	if len(selected) != len(wanted) {
		return nil, errors.New("部分条目已不存在，请刷新列表")
	}
	return s.store.encodeArchive(selected, nil, true)
}

const MaxBackupBytes = 128 << 20

func (s *promptStore) snapshot(tx *sql.Tx, items []Prompt) ([]byte, error) {
	rows, err := tx.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	settings := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s.encodeArchive(items, &settings, false)
}

func (s *promptStore) encodeArchive(items []Prompt, settings *map[string]string, partial bool) ([]byte, error) {
	images := map[string][]byte{}
	size := 0
	for _, item := range items {
		for _, name := range item.Images {
			if _, ok := images[name]; ok {
				continue
			}
			if !imageName.MatchString(name) {
				return nil, errors.New("数据库包含无效图片引用")
			}
			path, err := s.imagePath(scopedImage(item.Number, name))
			if err != nil {
				return nil, err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("无法读取备份图片 %s: %w", name, err)
			}
			size += len(data)
			if size > MaxBackupBytes*3/4 {
				return nil, errors.New("图片超出 128 MB JSON 备份上限，请备份整个应用数据目录")
			}
			images[name] = data
		}
	}
	version := 2
	if settings != nil || partial {
		version = 3
	}
	data, err := json.Marshal(backupFile{Format: backupFormat, Version: version, Items: items, Images: images, Settings: settings, Partial: partial})
	if len(data) > MaxBackupBytes {
		return nil, errors.New("超出 128 MB JSON 备份上限，请备份整个应用数据目录")
	}
	return data, err
}

func (s *promptStore) prepareImport(text string) ([]Prompt, ImportSummary, map[string][]byte, error) {
	items, summary, err := decodeImport(text)
	if err != nil {
		return nil, summary, nil, err
	}
	needed := map[string][]byte{}
	for _, item := range items {
		for _, name := range item.Images {
			needed[name] = nil
		}
	}
	if len(needed) == 0 {
		return items, summary, needed, nil
	}
	var envelope struct {
		Images map[string][]byte `json:"images"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(text, "\ufeff")), &envelope); err != nil {
		return nil, summary, nil, err
	}
	for name := range needed {
		data := envelope.Images[name]
		actual, err := checkedImage(data)
		if err != nil || name != actual {
			return nil, summary, nil, errors.New("备份中的图片缺失或已损坏")
		}
		needed[name] = data
	}
	return items, summary, needed, nil
}

func decodeImport(text string) ([]Prompt, ImportSummary, error) {
	if len(text) > MaxBackupBytes {
		return nil, ImportSummary{}, errors.New("备份文件不能超过 128 MB")
	}
	var envelope struct {
		Format    string             `json:"format"`
		Version   int                `json:"version"`
		Items     []json.RawMessage  `json:"items"`
		Favorites []string           `json:"favorites"`
		Partial   bool               `json:"partial"`
		Settings  *map[string]string `json:"settings"`
	}
	text = strings.TrimSpace(strings.TrimPrefix(text, "\ufeff"))
	var err error
	if strings.HasPrefix(text, "[") {
		err = json.Unmarshal([]byte(text), &envelope.Items)
	} else {
		err = json.Unmarshal([]byte(text), &envelope)
	}
	if err != nil || envelope.Items == nil {
		return nil, ImportSummary{}, errors.New("文件不是可识别的提示词 JSON")
	}
	if (envelope.Format != "" && envelope.Format != backupFormat) ||
		(envelope.Format == backupFormat && (envelope.Version < 1 || envelope.Version > 3)) {
		return nil, ImportSummary{}, errors.New("不支持该备份格式或版本")
	}
	if envelope.Format == backupFormat && envelope.Version == 3 && !envelope.Partial {
		if envelope.Settings == nil {
			return nil, ImportSummary{}, errors.New("备份缺少设置")
		}
		for key, value := range *envelope.Settings {
			if err := validateBackupSetting(key, value); err != nil {
				return nil, ImportSummary{}, err
			}
		}
	}

	favorites := map[string]bool{}
	for _, id := range envelope.Favorites {
		favorites[id] = true
	}
	items := make([]Prompt, 0, len(envelope.Items))
	invalid := 0
	for _, raw := range envelope.Items {
		var p Prompt
		if envelope.Format == backupFormat {
			err = json.Unmarshal(raw, &p)
		} else {
			p, err = legacyPrompt(raw, favorites)
		}
		if err != nil || p.ID == "" || p.validate() != nil {
			invalid++
			continue
		}
		p.CreatedAt = normalizeDate(p.CreatedAt, timestamp())
		p.UpdatedAt = normalizeDate(p.UpdatedAt, p.CreatedAt)
		items = append(items, p)
	}
	return items, ImportSummary{Invalid: invalid, CanRestore: envelope.Format == backupFormat && !envelope.Partial && invalid == 0}, nil
}

func validateBackupSetting(key, value string) error {
	switch {
	case key == "theme":
		if value != "system" && value != "light" && value != "dark" {
			return errors.New("备份中的主题设置无效")
		}
	case key == "layout":
		if value != "cards" && value != "list" {
			return errors.New("备份中的视图设置无效")
		}
	case key == "view-fields:cards" || key == "view-fields:list":
		var fields []string
		if err := json.Unmarshal([]byte(value), &fields); err != nil {
			return err
		}
		return validateViewFields(fields)
	case key == customModelKey || strings.HasPrefix(key, "categories:"):
		var names []string
		if json.Unmarshal([]byte(value), &names) != nil {
			return errors.New("备份中的分类或模型设置无效")
		}
	case strings.HasPrefix(key, "variables:"):
		var presets map[string]map[string]string
		if json.Unmarshal([]byte(value), &presets) != nil {
			return errors.New("备份中的变量预设无效")
		}
	}
	return nil
}

func legacyPrompt(raw json.RawMessage, favorites map[string]bool) (Prompt, error) {
	var old struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Prompt      string `json:"prompt"`
		Description string `json:"desc"`
		Author      string `json:"author"`
		Category    string `json:"category"`
		UpdatedAt   string `json:"updatedAt"`
	}
	if err := json.Unmarshal(raw, &old); err != nil {
		return Prompt{}, err
	}
	key := old.ID
	if key == "" {
		key = old.Title + "\x00" + old.Prompt
	}
	category := old.Category
	if name, ok := map[string]string{"companion": "陪伴型", "roleplay": "角色扮演", "utility": "实用助手", "custom": "自定义"}[category]; ok {
		category = name
	}
	return Prompt{
		ID: fmt.Sprintf("legacy-persona-%x", sha256.Sum256([]byte(key))), Kind: "persona",
		Title: old.Title, Body: old.Prompt, Description: old.Description, Author: old.Author,
		Category: category, UpdatedAt: old.UpdatedAt, Favorite: favorites[old.ID],
	}, nil
}

func normalizeDate(value, fallback string) string {
	for _, format := range []string{time.RFC3339Nano, "2006-01-02"} {
		if date, err := time.Parse(format, value); err == nil {
			return date.UTC().Format("2006-01-02T15:04:05.000000000Z")
		}
	}
	return fallback
}

func (s *Service) PreviewImport(text string) (ImportSummary, error) {
	incoming, summary, _, err := s.store.prepareImport(text)
	if err != nil {
		return summary, err
	}
	existing, err := s.store.list()
	if err != nil {
		return summary, err
	}
	seen := map[string]bool{}
	for _, item := range existing {
		seen[item.ID] = true
	}
	for _, item := range incoming {
		if seen[item.ID] {
			summary.Duplicate++
		} else {
			summary.Added++
			seen[item.ID] = true
		}
	}
	return summary, nil
}

func (s *Service) ImportPrompts(text string, replace bool) (ImportSummary, error) {
	incoming, summary, files, err := s.store.prepareImport(text)
	if err != nil {
		return summary, err
	}
	tx, err := s.store.db.Begin()
	if err != nil {
		return summary, err
	}
	defer tx.Rollback()
	if replace {
		if !summary.CanRestore {
			return ImportSummary{}, errors.New("只有完整的双库备份可以用于覆盖恢复")
		}
		seen := map[string]bool{}
		for _, item := range incoming {
			if seen[item.ID] {
				return ImportSummary{}, errors.New("备份包含重复的条目 ID，无法覆盖恢复")
			}
			seen[item.ID] = true
		}
		existing, err := listPrompts(tx)
		if err != nil {
			return summary, err
		}
		snapshot, err := s.store.snapshot(tx, existing)
		if err != nil {
			return summary, fmt.Errorf("无法创建恢复前安全快照：%w", err)
		}
		backupDir := filepath.Join(s.store.dir, "backups")
		if err := os.MkdirAll(backupDir, 0700); err != nil {
			return summary, err
		}
		summary.Snapshot = filepath.Join(backupDir, "before-restore-"+time.Now().Format("20060102-150405")+"-"+rand.Text()+".json")
		if err := fileio.WriteAtomic(summary.Snapshot, snapshot); err != nil {
			return summary, err
		}
		var archive struct {
			Version  int               `json:"version"`
			Settings map[string]string `json:"settings"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(text, "\ufeff"))), &archive); err != nil {
			return summary, err
		}
		if archive.Version == 3 {
			if _, err := tx.Exec("DELETE FROM settings"); err != nil {
				return summary, err
			}
			for key, value := range archive.Settings {
				if err := putSetting(tx, key, value); err != nil {
					return summary, err
				}
			}
		}

		if _, err := tx.Exec("DELETE FROM prompts"); err != nil {
			return ImportSummary{}, err
		}
	}
	for _, item := range incoming {
		added, err := insertPrompt(tx, item)
		if err != nil {
			return ImportSummary{}, err
		}
		if added {
			number, err := ensurePromptNumber(tx, item.ID, item.Number)
			if err != nil {
				return summary, err
			}
			for _, name := range item.Images {
				if err := s.store.writeImage(scopedImage(number, name), files[name]); err != nil {
					return ImportSummary{}, err
				}
			}
			summary.Added++
		} else {
			summary.Duplicate++
		}
	}
	if err := tx.Commit(); err != nil {
		return ImportSummary{}, err
	}
	return summary, nil
}

func Markdown(p Prompt) ([]byte, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	label := "人设提示词"
	if p.Kind == "art" {
		label = "中文提示词"
	}
	sections := []string{"# " + p.Title}
	for _, field := range []struct{ label, value string }{
		{"说明", p.Description}, {"分类", p.Category}, {"标签", strings.Join(p.Tags, "、")},
		{"作者", p.Author}, {"平台或模型", strings.Join(p.Models, "、")}, {"参数备注", p.Parameters},
		{label, p.Body}, {"英文提示词", p.English}, {"负向提示词", p.Negative},
	} {
		if strings.TrimSpace(field.value) != "" {
			sections = append(sections, fmt.Sprintf("## %s\n\n%s", field.label, field.value))
		}
	}
	return []byte(strings.Join(sections, "\n\n") + "\n"), nil
}
