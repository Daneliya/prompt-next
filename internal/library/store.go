package library

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type promptStore struct {
	db          *sql.DB
	dir         string
	thumbnailMu sync.Mutex
}

func openPromptStore() (*promptStore, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir, err := resolveDataDirectory(base)
	if err != nil {
		return nil, err
	}
	return openStoreAt(dir)
}

func openStoreAt(dir string) (*promptStore, error) {
	path := filepath.ToSlash(filepath.Join(dir, "prompts.db"))
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	dsn := url.URL{Scheme: "file", Path: path, RawQuery: "_pragma=busy_timeout%285000%29&_txlock=immediate"}
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&journalMode); err != nil {
		db.Close()
		return nil, err
	}
	if journalMode != "wal" {
		db.Close()
		return nil, fmt.Errorf("无法启用 SQLite WAL，当前日志模式为 %s", journalMode)
	}
	store := &promptStore{db: db, dir: dir}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *promptStore) migrate() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err := tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	var migratedImages []string
	migrations := []func(*sql.Tx) error{initialSchema, s.addRecoverableDeletes, func(tx *sql.Tx) error {
		var err error
		migratedImages, err = s.addPromptNumbers(tx)
		return err
	}}
	if version < 0 || version > len(migrations) {
		return fmt.Errorf("数据库版本 %d 高于当前应用支持的版本", version)
	}
	for version < len(migrations) {
		if err := migrations[version](tx); err != nil {
			return fmt.Errorf("数据库迁移 %d 失败：%w", version+1, err)
		}
		version++
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("DELETE FROM deleted_prompts WHERE expires_at <= ?", time.Now().Unix()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// The old copies remain available until the new database layout is committed.
	for _, name := range migratedImages {
		_ = os.Remove(filepath.Join(s.imagesDir(), name))
		_ = os.Remove(s.thumbnailPath(name))
	}
	return nil
}

func initialSchema(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE prompts (
			id TEXT PRIMARY KEY NOT NULL,
			kind TEXT NOT NULL CHECK(kind IN ('persona', 'art')),
			updated_at TEXT NOT NULL,
			record TEXT NOT NULL
		);
		CREATE INDEX prompts_kind_updated ON prompts(kind, updated_at DESC);
		CREATE TABLE settings (key TEXT PRIMARY KEY NOT NULL, value TEXT NOT NULL);
	`)
	if err != nil {
		return err
	}
	now := timestamp()
	samples := []Prompt{
		{ID: "sample-persona", Kind: "persona", Title: "客服助手 · 示例", Body: "你是 {{品牌名}} 的客服助手。先确认用户的问题，再给出清晰、礼貌、可执行的答复。", Category: "实用助手", Tags: []string{"示例"}, CreatedAt: now, UpdatedAt: now},
		{ID: "sample-art", Kind: "art", Title: "雨夜街景 · 示例", Body: "雨夜街景，{{主色调}}霓虹灯，湿润路面，电影感构图。", Negative: "模糊，过曝，水印", Category: "场景", Tags: []string{"示例"}, Parameters: "横向构图，16:9", CreatedAt: now, UpdatedAt: now},
	}
	for _, sample := range samples {
		if _, err := insertPrompt(tx, sample); err != nil {
			return err
		}
	}
	return nil
}

func (s *promptStore) addRecoverableDeletes(tx *sql.Tx) error {
	if _, err := tx.Exec(`CREATE TABLE deleted_prompts (
		batch TEXT NOT NULL, id TEXT NOT NULL, expires_at INTEGER NOT NULL, record TEXT NOT NULL,
		PRIMARY KEY(batch, id)
	)`); err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(s.dir, "custom-image-models.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var models []string
	if err := json.Unmarshal(raw, &models); err != nil {
		return err
	}
	return putSetting(tx, customModelKey, stringsJSON(cleanTags(models)))
}

func timestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z")
}

func insertPrompt(tx *sql.Tx, p Prompt) (bool, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return false, err
	}
	result, err := tx.Exec("INSERT INTO prompts VALUES (?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", p.ID, p.Kind, p.UpdatedAt, string(data))
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func (s *promptStore) list() ([]Prompt, error) {
	return listPrompts(s.db)
}

func listPrompts(db interface {
	Query(string, ...any) (*sql.Rows, error)
}) ([]Prompt, error) {
	rows, err := db.Query("SELECT p.record, n.number FROM prompts p JOIN prompt_numbers n ON n.prompt_id = p.id ORDER BY p.updated_at DESC, p.id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	prompts := []Prompt{}
	for rows.Next() {
		var record string
		var prompt Prompt
		var number int64
		if err := rows.Scan(&record, &number); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(record), &prompt); err != nil {
			return nil, err
		}
		prompt.Number = number
		prompt.normalizeModels()
		prompts = append(prompts, prompt)
	}
	return prompts, rows.Err()
}

func savePrompt(tx *sql.Tx, p Prompt) (Prompt, error) {
	if err := p.validate(); err != nil {
		return Prompt{}, err
	}
	p.UpdatedAt = timestamp()
	if p.ID == "" {
		p.ID = rand.Text()
		p.CreatedAt = p.UpdatedAt
		data, err := json.Marshal(p)
		if err != nil {
			return Prompt{}, err
		}
		_, err = tx.Exec("INSERT INTO prompts VALUES (?, ?, ?, ?)", p.ID, p.Kind, p.UpdatedAt, string(data))
		if err != nil {
			return Prompt{}, err
		}
		p.Number, err = ensurePromptNumber(tx, p.ID, 0)
		return p, err
	}
	if err := tx.QueryRow("SELECT number FROM prompt_numbers WHERE prompt_id = ?", p.ID).Scan(&p.Number); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Prompt{}, errors.New("找不到这条提示词，请刷新列表")
		}
		return Prompt{}, err
	}
	result, err := tx.Exec(`UPDATE prompts SET updated_at = ?,
		record = json_set(record, '$.title', ?, '$.body', ?, '$.description', ?,
			'$.category', ?, '$.tags', json(?), '$.author', ?, '$.negative', ?,
			'$.model', ?, '$.models', json(?), '$.parameters', ?, '$.english', ?, '$.images', json(?), '$.updatedAt', ?)
		WHERE id = ? AND kind = ?`,
		p.UpdatedAt, p.Title, p.Body, p.Description, p.Category, stringsJSON(p.Tags), p.Author,
		p.Negative, p.Model, stringsJSON(p.Models), p.Parameters, p.English, stringsJSON(p.Images), p.UpdatedAt, p.ID, p.Kind)
	return p, requireChanged(result, err)
}

func stringsJSON(tags []string) string {
	data, _ := json.Marshal(tags)
	return string(data)
}

func requireChanged(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return errors.New("找不到这条提示词，请刷新列表")
	}
	return err
}
