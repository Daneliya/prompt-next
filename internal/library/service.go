package library

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
)

type Service struct {
	store           *promptStore
	locationWarning string
}

func Open() (*Service, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return openWithLocation(base)
}

func Close(s *Service) error {
	return s.store.db.Close()
}

func DataDirectory(s *Service) string {
	return s.store.dir
}

func ImageAssets(s *Service) func(http.Handler) http.Handler {
	return s.store.imageAssets
}

func (s *Service) ListPrompts() ([]Prompt, error) {
	return s.store.list()
}

func (s *Service) ListPromptsByKind(kind string) ([]Prompt, error) {
	return queryPrompts(s.store.db, " WHERE p.kind = ?", kind)
}

func (s *Service) PromptsByIDs(ids []string) ([]Prompt, error) {
	return queryPrompts(s.store.db, " WHERE p.id IN (SELECT value FROM json_each(?))", stringsJSON(ids))
}

func (s *Service) SavePrompt(prompt Prompt) (Prompt, error) {
	return s.SaveWithImages(prompt, nil)
}

func (s *Service) DeletePrompt(id string) error {
	_, err := s.DeletePrompts([]string{id})
	return err
}

func (s *Service) SetFavorite(id string, favorite bool) error {
	return s.BatchFavorite([]string{id}, favorite)
}

func (s *Service) GetTheme() (string, error) {
	var theme string
	err := s.store.db.QueryRow("SELECT value FROM settings WHERE key = 'theme'").Scan(&theme)
	if errors.Is(err, sql.ErrNoRows) {
		return "system", nil
	}
	return theme, err
}

func (s *Service) SetTheme(theme string) error {
	if theme != "system" && theme != "light" && theme != "dark" {
		return errors.New("主题选项无效")
	}
	_, err := s.store.db.Exec("INSERT INTO settings VALUES ('theme', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", theme)
	return err
}

func (s *Service) GetLayout() (string, error) {
	return getSetting(s.store.db, "layout")
}

func (s *Service) SetLayout(layout string) error {
	if !validLayout(layout) {
		return errors.New("视图选项无效")
	}
	return putSetting(s.store.db, "layout", layout)
}

func validLayout(layout string) bool {
	return layout == "cards" || layout == "list" || layout == "waterfall"
}
