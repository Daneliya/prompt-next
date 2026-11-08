package library

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"slices"
	"strings"
)

//go:embed image-models.json
var builtInModels []byte

func (s *Service) Models() ([]string, error) {
	var defaults []string
	if err := json.Unmarshal(builtInModels, &defaults); err != nil {
		return nil, err
	}
	custom, err := s.customModels()
	return cleanTags(append(defaults, custom...)), err
}

const customModelKey = "models:custom"

type settingsReader interface {
	QueryRow(string, ...any) *sql.Row
}

func getSetting(db settingsReader, key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

func putSetting(db interface {
	Exec(string, ...any) (sql.Result, error)
}, key, value string) error {
	_, err := db.Exec("INSERT INTO settings VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}

func settingList(db settingsReader, key string) ([]string, error) {
	raw, err := getSetting(db, key)
	if err != nil || raw == "" {
		return nil, err
	}
	var values []string
	err = json.Unmarshal([]byte(raw), &values)
	return values, err
}

func (s *Service) customModels() ([]string, error) {
	return settingList(s.store.db, customModelKey)
}

func (s *Service) CustomModels() ([]string, error) {
	return s.customModels()
}

func (s *Service) AddModel(name string) error {
	return s.updateModel("", name)
}

func (s *Service) RenameModel(current, name string) error {
	if current == "" {
		return errors.New("请选择自定义模型")
	}
	return s.updateModel(current, name)
}

func (s *Service) updateModel(current, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 100 {
		return errors.New("模型名称须为 1 至 100 个字符")
	}
	tx, err := s.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	custom, err := settingList(tx, customModelKey)
	if err != nil {
		return err
	}
	index := slices.Index(custom, current)
	if current != "" && index < 0 {
		return errors.New("找不到该自定义模型")
	}
	var defaults []string
	if err := json.Unmarshal(builtInModels, &defaults); err != nil {
		return err
	}
	for _, model := range defaults {
		if strings.EqualFold(name, model) {
			return errors.New("该模型已存在")
		}
	}
	for i, model := range custom {
		if i != index && strings.EqualFold(name, model) {
			return errors.New("该模型已存在")
		}
	}
	if current == "" {
		custom = append(custom, name)
	} else {
		custom[index] = name
	}
	if err := putSetting(tx, customModelKey, stringsJSON(custom)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) DeleteModel(name string) error {
	tx, err := s.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	custom, err := settingList(tx, customModelKey)
	if err != nil {
		return err
	}
	index := slices.Index(custom, name)
	if index < 0 {
		return errors.New("找不到该自定义模型")
	}
	if err := putSetting(tx, customModelKey, stringsJSON(slices.Delete(custom, index, index+1))); err != nil {
		return err
	}
	return tx.Commit()
}

func defaultCategories(kind string) []string {
	switch kind {
	case "text":
		return []string{"效率工具", "编程开发", "创意写作", "数据分析", "教育学习", "商业营销", "生活娱乐", "翻译润色", "其他", "写作", "翻译", "学习", "办公", "编程"}
	case "art":
		return []string{"国风美学", "海报设计", "广告创意", "商品设计", "人像创作", "动物萌宠", "生活分享", "赛博科幻", "壁纸海报", "绘本插画", "动漫动画", "风景摄影", "建筑空间", "Logo设计", "UI界面", "场景", "人物", "产品", "插画", "摄影", "设计", "其他"}
	case "video":
		return []string{"短视频制作", "广告宣传", "动画制作", "特效制作", "纪录影片", "教程视频"}
	case "persona":
		return []string{"实用助手", "陪伴型", "角色扮演", "自定义"}
	default:
		return nil
	}
}

func (s *Service) CategoryOptions(kind string) ([]string, error) {
	defaults := defaultCategories(kind)
	if defaults == nil {
		return nil, errors.New("提示词库类型无效")
	}
	custom, err := s.customCategories(s.store.db, kind)
	return cleanTags(append(defaults, custom...)), err
}

func (s *Service) customCategories(db settingsReader, kind string) ([]string, error) {
	return settingList(db, "categories:"+kind)
}

func (s *Service) AddCategory(kind, name string) error {
	defaults := defaultCategories(kind)
	name = strings.TrimSpace(name)
	if defaults == nil || name == "" || len([]rune(name)) > 60 {
		return errors.New("分类名称须为 1 至 60 个字符，且属于有效提示词库")
	}
	tx, err := s.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	custom, err := s.customCategories(tx, kind)
	if err != nil {
		return err
	}
	for _, existing := range append(defaults, custom...) {
		if strings.EqualFold(existing, name) {
			return errors.New("该分类已存在")
		}
	}
	_, err = tx.Exec("INSERT INTO settings VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", "categories:"+kind, stringsJSON(append(custom, name)))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) CustomCategories(kind string) ([]string, error) {
	if defaultCategories(kind) == nil {
		return nil, errors.New("提示词库类型无效")
	}
	return s.customCategories(s.store.db, kind)
}

func (s *Service) RenameCategory(kind, current, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 60 {
		return errors.New("分类名称须为 1 至 60 个字符")
	}
	return s.updateCategory(kind, current, name)
}

func (s *Service) DeleteCategory(kind, name string) error {
	return s.updateCategory(kind, name, "")
}

func (s *Service) updateCategory(kind, current, name string) error {
	defaults := defaultCategories(kind)
	if defaults == nil {
		return errors.New("提示词库类型无效")
	}
	tx, err := s.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	custom, err := s.customCategories(tx, kind)
	if err != nil {
		return err
	}
	index := slices.Index(custom, current)
	if index < 0 {
		return errors.New("请选择自定义分类")
	}
	if name == "" {
		custom = slices.Delete(custom, index, index+1)
	} else {
		for _, other := range append(defaults, custom...) {
			if other != current && strings.EqualFold(other, name) {
				return errors.New("该分类已存在")
			}
		}
		custom[index] = name
	}
	if err := putSetting(tx, "categories:"+kind, stringsJSON(custom)); err != nil {
		return err
	}
	now := timestamp()
	if _, err := tx.Exec(`UPDATE prompts SET updated_at = ?, record = json_set(record, '$.category', ?, '$.updatedAt', ?)
		WHERE kind = ? AND json_extract(record, '$.category') = ?`, now, name, now, kind, current); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ViewFields(layout string) ([]string, error) {
	if !validViewFieldKey(layout) {
		return nil, errors.New("视图选项无效")
	}
	return settingList(s.store.db, "view-fields:"+layout)
}

func (s *Service) SetViewFields(layout string, fields []string) error {
	if !validViewFieldKey(layout) {
		return errors.New("视图选项无效")
	}
	fields = cleanTags(fields)
	if err := validateViewFields(fields); err != nil {
		return err
	}
	return putSetting(s.store.db, "view-fields:"+layout, stringsJSON(fields))
}

func validViewFieldKey(key string) bool {
	return validLayout(strings.TrimPrefix(key, "persona:"))
}

func validateViewFields(fields []string) error {
	for _, field := range fields {
		switch field {
		case "title", "body", "description", "models", "category", "tags", "number", "created", "updated", "characters":
		default:
			return errors.New("展示字段无效")
		}
	}
	return nil
}
