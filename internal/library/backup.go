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

const backupFormat = "prompt-next"
const legacyBackupFormat = "bigfish-prompt-manager"

type ImportSummary struct {
	Added         int    `json:"added"`
	Duplicate     int    `json:"duplicate"`
	Invalid       int    `json:"invalid"`
	CanRestore    bool   `json:"canRestore"`
	IncludesVideo bool   `json:"includesVideo"`
	IncludesText  bool   `json:"includesText"`
	Snapshot      string `json:"snapshot"`
}

type backupFile struct {
	Format   string             `json:"format"`
	Version  int                `json:"version"`
	Items    []Prompt           `json:"items"`
	Images   map[string][]byte  `json:"images,omitempty"`
	Videos   map[string][]byte  `json:"videos,omitempty"`
	Settings *map[string]string `json:"settings,omitempty"`
	Authors  *[]Author          `json:"authors,omitempty"`
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
	return s.store.encodeArchive(selected, nil, nil, true)
}

const MaxBackupBytes = 128 << 20

func (s *promptStore) snapshot(tx *sql.Tx, items []Prompt) ([]byte, error) {
	rows, err := tx.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	settings := map[string]string{promptCollectionsKey: "[]"}
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
	authors, err := listAuthors(tx)
	if err != nil {
		return nil, err
	}
	return s.encodeArchive(items, &settings, &authors, false)
}

func (s *promptStore) encodeArchive(items []Prompt, settings *map[string]string, authors *[]Author, partial bool) ([]byte, error) {
	images := map[string][]byte{}
	videos := map[string][]byte{}
	size := 0
	var hasReferences bool
	for _, item := range items {
		hasReferences = hasReferences || len(item.ReferenceImages) > 0
		for _, name := range item.imageNames() {
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
		for _, name := range item.videoNames() {
			if _, exists := videos[name]; exists {
				continue
			}
			path, err := s.videoPath(scopedImage(item.Number, name))
			if err != nil {
				return nil, err
			}
			data, err := readVideoFile(path)
			if err != nil {
				return nil, fmt.Errorf("无法读取备份视频：%w", err)
			}
			size += len(data)
			if size > MaxBackupBytes*3/4 {
				return nil, errors.New("媒体文件超出 128 MB JSON 备份上限，请备份整个应用数据目录")
			}
			videos[name] = data
		}
	}
	version := 2
	if settings != nil || partial {
		version = 5
	}
	if len(videos) > 0 {
		version = 7
	}
	if hasReferences {
		version = 8
	}
	data, err := json.Marshal(backupFile{Format: backupFormat, Version: version, Items: items, Images: images, Videos: videos, Settings: settings, Authors: authors, Partial: partial})
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
		for _, name := range item.imageNames() {
			needed[name] = nil
		}
		for _, name := range item.videoNames() {
			needed[name] = nil
		}
	}
	if len(needed) == 0 {
		return items, summary, needed, nil
	}
	var envelope struct {
		Images map[string][]byte `json:"images"`
		Videos map[string][]byte `json:"videos"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(text, "\ufeff")), &envelope); err != nil {
		return nil, summary, nil, err
	}
	for name := range needed {
		if videoName.MatchString(name) {
			data := envelope.Videos[name]
			actual, err := checkedVideo(data, filepath.Ext(name))
			if err != nil || name != actual {
				return nil, summary, nil, errors.New("备份中的视频缺失或已损坏")
			}
			needed[name] = data
			continue
		}
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
		Authors   []Author           `json:"authors"`
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
	isBackup := envelope.Format == backupFormat || envelope.Format == legacyBackupFormat
	if (envelope.Format != "" && !isBackup) ||
		(isBackup && (envelope.Version < 1 || envelope.Version > 8)) {
		return nil, ImportSummary{}, errors.New("不支持该备份格式或版本")
	}
	for _, author := range envelope.Authors {
		if err := author.validate(); err != nil {
			return nil, ImportSummary{}, err
		}
	}
	if isBackup && envelope.Version >= 3 && !envelope.Partial {
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
		if isBackup {
			err = json.Unmarshal(raw, &p)
		} else {
			p, err = legacyPrompt(raw, favorites)
		}
		if err != nil || p.ID == "" || (p.Kind == "text" && envelope.Version < 4) || (p.Kind == "video" && envelope.Version < 5) || (p.Video != "" && envelope.Version < 6) || (len(p.Videos) > 0 && envelope.Version < 7) || (len(p.ReferenceImages) > 0 && envelope.Version < 8) || p.validate() != nil {
			invalid++
			continue
		}
		p.CreatedAt = normalizeDate(p.CreatedAt, timestamp())
		p.UpdatedAt = normalizeDate(p.UpdatedAt, p.CreatedAt)
		items = append(items, p)
	}
	return items, ImportSummary{Invalid: invalid, IncludesText: envelope.Version >= 4, IncludesVideo: envelope.Version >= 5, CanRestore: isBackup && !envelope.Partial && invalid == 0}, nil
}

func validateBackupSetting(key, value string) error {
	switch {
	case key == promptCollectionsKey:
		_, err := decodeCollections(value)
		return err
	case key == "theme":
		if value != "system" && value != "light" && value != "dark" {
			return errors.New("备份中的主题设置无效")
		}
	case key == "layout":
		if !validLayout(value) {
			return errors.New("备份中的视图设置无效")
		}
	case strings.HasPrefix(key, "view-fields:") && validViewFieldKey(strings.TrimPrefix(key, "view-fields:")):
		var fields []string
		if err := json.Unmarshal([]byte(value), &fields); err != nil {
			return err
		}
		return validateViewFields(fields)
	case key == customModelKey || key == customVideoModelKey || strings.HasPrefix(key, "categories:"):
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
	var archive struct {
		Version  int               `json:"version"`
		Settings map[string]string `json:"settings"`
		Authors  *[]Author         `json:"authors"`
	}
	jsonText := strings.TrimSpace(strings.TrimPrefix(text, "\ufeff"))
	if strings.HasPrefix(jsonText, "{") {
		if err := json.Unmarshal([]byte(jsonText), &archive); err != nil {
			return summary, err
		}
	}
	if replace {
		if !summary.CanRestore {
			return ImportSummary{}, errors.New("只有完整的应用备份可以用于覆盖恢复")
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
		if archive.Version >= 3 {
			if err := restoreSettings(tx, archive.Settings, summary.IncludesText, summary.IncludesVideo); err != nil {
				return summary, err
			}
		}

		if _, err := tx.Exec("DELETE FROM prompts WHERE (kind != 'text' OR ?) AND (kind != 'video' OR ?)", summary.IncludesText, summary.IncludesVideo); err != nil {
			return ImportSummary{}, err
		}
	}
	if archive.Authors != nil {
		if replace {
			if _, err := tx.Exec("DELETE FROM authors"); err != nil {
				return summary, err
			}
		}
		for _, author := range *archive.Authors {
			if err := author.validate(); err != nil {
				return summary, err
			}
			if _, err := tx.Exec(`INSERT INTO authors(source_name, name, bio) VALUES (?, ?, ?)
				ON CONFLICT(source_name) DO NOTHING`, author.SourceName, author.Name, author.Bio); err != nil {
				return summary, err
			}
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
			for _, name := range item.imageNames() {
				if err := s.store.writeImage(scopedImage(number, name), files[name]); err != nil {
					return ImportSummary{}, err
				}
			}
			for _, name := range item.videoNames() {
				if err := s.store.writeVideo(scopedImage(number, name), files[name]); err != nil {
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

func restoreSettings(tx *sql.Tx, settings map[string]string, includesText, includesVideo bool) error {
	if _, err := tx.Exec(`DELETE FROM settings
		WHERE key != ? AND (? OR (key NOT IN ('categories:text', 'view-fields:waterfall') AND key NOT IN
			(SELECT 'variables:' || id FROM prompts WHERE kind = 'text')))
		AND (? OR (key NOT IN ('categories:video', 'models:custom:video') AND key NOT IN
			(SELECT 'variables:' || id FROM prompts WHERE kind = 'video')))`, promptCollectionsKey, includesText, includesVideo); err != nil {
		return err
	}
	for key, value := range settings {
		if err := putSetting(tx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func Markdown(p Prompt) ([]byte, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	label := "提示词"
	switch p.Kind {
	case "art", "video":
		label = "中文提示词"
	case "persona":
		label = "人设提示词"
	}
	sections := []string{"# " + p.Title}
	for _, field := range []struct{ label, value string }{
		{"说明", p.Description}, {"分类", p.Category}, {"标签", strings.Join(p.Tags, "、")},
		{"原作者", p.Author}, {"提示词原链接", p.AuthorURL}, {"平台或模型", strings.Join(p.Models, "、")}, {"参数备注", p.Parameters},
		{label, p.Body}, {"英文提示词", p.English}, {"负向提示词", p.Negative},
	} {
		if strings.TrimSpace(field.value) != "" {
			sections = append(sections, fmt.Sprintf("## %s\n\n%s", field.label, field.value))
		}
	}
	return []byte(strings.Join(sections, "\n\n") + "\n"), nil
}
