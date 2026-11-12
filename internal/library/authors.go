package library

import (
	"database/sql"
	"errors"
	"strings"
)

type Author struct {
	SourceName string `json:"sourceName"`
	Name       string `json:"name"`
	Bio        string `json:"bio"`
}

func addAuthors(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE authors (
			source_name TEXT PRIMARY KEY NOT NULL,
			name TEXT NOT NULL,
			bio TEXT NOT NULL DEFAULT ''
		);
		INSERT OR IGNORE INTO authors(source_name, name)
			SELECT trim(json_extract(record, '$.author')), trim(json_extract(record, '$.author'))
			FROM prompts WHERE trim(json_extract(record, '$.author')) <> '';
		CREATE TRIGGER prompts_author_insert AFTER INSERT ON prompts
		WHEN trim(json_extract(new.record, '$.author')) <> ''
		BEGIN
			INSERT INTO authors(source_name, name)
			VALUES (trim(json_extract(new.record, '$.author')), trim(json_extract(new.record, '$.author')))
			ON CONFLICT(source_name) DO NOTHING;
		END;
		CREATE TRIGGER prompts_author_update AFTER UPDATE OF record ON prompts
		WHEN trim(json_extract(new.record, '$.author')) <> ''
			AND json_extract(new.record, '$.author') IS NOT json_extract(old.record, '$.author')
		BEGIN
			INSERT INTO authors(source_name, name)
			VALUES (trim(json_extract(new.record, '$.author')), trim(json_extract(new.record, '$.author')))
			ON CONFLICT(source_name) DO NOTHING;
		END;
	`)
	return err
}

func (s *Service) ListAuthors() ([]Author, error) {
	return listAuthors(s.store.db)
}

func listAuthors(db interface {
	Query(string, ...any) (*sql.Rows, error)
}) ([]Author, error) {
	rows, err := db.Query("SELECT source_name, name, bio FROM authors ORDER BY name, source_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	authors := []Author{}
	for rows.Next() {
		var author Author
		if err := rows.Scan(&author.SourceName, &author.Name, &author.Bio); err != nil {
			return nil, err
		}
		authors = append(authors, author)
	}
	return authors, rows.Err()
}

func (a *Author) validate() error {
	a.SourceName = strings.TrimSpace(a.SourceName)
	a.Name = strings.TrimSpace(a.Name)
	a.Bio = strings.TrimSpace(a.Bio)
	if a.SourceName == "" || a.Name == "" || len([]rune(a.Name)) > 200 || len([]rune(a.Bio)) > 2000 {
		return errors.New("作者名称必填且最多 200 字，简介最多 2000 字")
	}
	return nil
}

func (s *Service) SaveAuthor(author Author) (Author, error) {
	if err := author.validate(); err != nil {
		return Author{}, err
	}
	result, err := s.store.db.Exec("UPDATE authors SET name = ?, bio = ? WHERE source_name = ?", author.Name, author.Bio, author.SourceName)
	if err != nil {
		return Author{}, err
	}
	if count, err := result.RowsAffected(); err != nil || count == 0 {
		if err != nil {
			return Author{}, err
		}
		return Author{}, errors.New("找不到这位作者，请刷新列表")
	}
	return author, nil
}
