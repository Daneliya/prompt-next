package library

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyImportAndBackupRestore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompts.db")
	open := func() *Service {
		t.Helper()
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		store := &promptStore{db: db, dir: filepath.Dir(path)}
		if err := store.migrate(); err != nil {
			t.Fatal(err)
		}
		return &Service{store: store}
	}
	s := open()
	legacy := `{"version":1,"favorites":["old"],"items":[
		{"id":"old","title":"客服助手","prompt":"你好","category":"utility"},
		{"id":"blank","title":"缺正文"}
	]}`
	preview, err := s.PreviewImport(legacy)
	if err != nil || preview.Added != 1 || preview.Invalid != 1 || preview.CanRestore {
		t.Fatalf("preview = %+v, %v", preview, err)
	}
	if result, err := s.ImportPrompts(legacy, false); err != nil || result.Added != 1 {
		t.Fatalf("import = %+v, %v", result, err)
	}
	items, err := s.ListPrompts()
	if err != nil || len(items) != 3 {
		t.Fatalf("list = %+v, %v", items, err)
	}
	imported := items[0]
	if !imported.Favorite || imported.Category != "实用助手" {
		t.Fatalf("legacy fields were not preserved: %+v", imported)
	}
	imported.Body = "保留用户编辑"
	if _, err := s.SavePrompt(imported); err != nil {
		t.Fatal(err)
	}
	if result, err := s.ImportPrompts(legacy, false); err != nil || result.Duplicate != 1 {
		t.Fatalf("reimport = %+v, %v", result, err)
	}
	items, err = s.ListPrompts()
	if err != nil || items[0].Body != imported.Body {
		t.Fatalf("reimport overwrote edits: %+v, %v", items, err)
	}
	if err := s.SetTheme("dark"); err != nil {
		t.Fatal(err)
	}
	if err := s.store.db.Close(); err != nil {
		t.Fatal(err)
	}
	s = open()
	if theme, err := s.GetTheme(); err != nil || theme != "dark" {
		t.Fatalf("theme = %q, %v", theme, err)
	}
	if err := s.SetFavorite(imported.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportPrompts(legacy, true); err == nil {
		t.Fatal("legacy import unexpectedly replaced both libraries")
	}

	backup, err := encodeBackup(items)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := s.ImportPrompts(string(backup), true); err != nil || result.Added != len(items) {
		t.Fatalf("restore = %+v, %v", result, err)
	}
	restored, err := s.ListPrompts()
	if err != nil || len(restored) != len(items) || !restored[0].Favorite || restored[0].Body != imported.Body {
		t.Fatalf("restored = %+v, %v", restored, err)
	}
}

func TestLegacyPersonaExportFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "legacy-persona-export.json"))
	if err != nil {
		t.Fatal(err)
	}
	items, summary, err := decodeImport(string(data))
	if err != nil || len(items) != 1 || summary.Invalid != 1 || items[0].Category != "角色扮演" {
		t.Fatalf("legacy persona export: %d valid, %d invalid, %v", len(items), summary.Invalid, err)
	}
	t.Logf("legacy persona export: %d valid, %d skipped", len(items), summary.Invalid)
}
