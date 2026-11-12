package library

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
)

const promptCollectionsKey = "prompt-collections"

type PromptCollection struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	PromptIDs   []string `json:"promptIds"`
	UpdatedAt   string   `json:"updatedAt,omitempty"`
}

func decodeCollections(raw string) ([]PromptCollection, error) {
	collections := []PromptCollection{}
	if raw == "" {
		return collections, nil
	}
	if err := json.Unmarshal([]byte(raw), &collections); err != nil {
		return nil, errors.New("合集数据无效")
	}
	seen := map[string]bool{}
	for _, collection := range collections {
		if collection.ID == "" || seen[collection.ID] || !validCollectionName(collection.Name) || len([]rune(collection.Description)) > 1000 {
			return nil, errors.New("合集数据无效")
		}
		seen[collection.ID] = true
	}
	return collections, nil
}

func validCollectionName(name string) bool {
	return strings.TrimSpace(name) != "" && len([]rune(name)) <= 80
}

func readCollections(db settingsReader) ([]PromptCollection, error) {
	raw, err := getSetting(db, promptCollectionsKey)
	if err != nil {
		return nil, err
	}
	return decodeCollections(raw)
}

func (s *Service) ListCollections() ([]PromptCollection, error) {
	return readCollections(s.store.db)
}

func (s *Service) SaveCollection(collection PromptCollection) (PromptCollection, error) {
	collection.Name = strings.TrimSpace(collection.Name)
	collection.Description = strings.TrimSpace(collection.Description)
	if len([]rune(collection.Description)) > 1000 {
		return PromptCollection{}, errors.New("合集说明最多 1000 个字符")
	}
	if !validCollectionName(collection.Name) {
		return PromptCollection{}, errors.New("合集名称须为 1 至 80 个字符")
	}
	collection.PromptIDs = cleanTags(collection.PromptIDs)
	collection.UpdatedAt = timestamp()
	err := s.updateCollections(func(tx *sql.Tx, collections []PromptCollection) ([]PromptCollection, error) {
		index := slices.IndexFunc(collections, func(existing PromptCollection) bool { return existing.ID == collection.ID })
		if collection.ID != "" && index < 0 {
			return nil, errors.New("合集已不存在")
		}
		for _, id := range collection.PromptIDs {
			if index >= 0 && slices.Contains(collections[index].PromptIDs, id) {
				continue
			}
			var present int
			err := tx.QueryRow("SELECT 1 FROM prompts WHERE id = ?", id).Scan(&present)
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.New("合集中的提示词已不存在，请重新选择")
			}
			if err != nil {
				return nil, err
			}
		}
		if index >= 0 {
			collections[index] = collection
			return collections, nil
		}
		collection.ID = rand.Text()
		return append(collections, collection), nil
	})
	return collection, err
}

func (s *Service) DeleteCollection(id string) error {
	return s.updateCollections(func(_ *sql.Tx, collections []PromptCollection) ([]PromptCollection, error) {
		index := slices.IndexFunc(collections, func(collection PromptCollection) bool { return collection.ID == id })
		if index < 0 {
			return nil, errors.New("合集已不存在")
		}
		return slices.Delete(collections, index, index+1), nil
	})
}

func (s *Service) updateCollections(change func(*sql.Tx, []PromptCollection) ([]PromptCollection, error)) error {
	tx, err := s.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	collections, err := readCollections(tx)
	if err != nil {
		return err
	}
	collections, err = change(tx, collections)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(collections)
	if err != nil {
		return err
	}
	if err := putSetting(tx, promptCollectionsKey, string(raw)); err != nil {
		return err
	}
	return tx.Commit()
}
