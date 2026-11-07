package library

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func testLibrary(t *testing.T, dir string) *Service {
	t.Helper()
	store, err := openStoreAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.db.Close() })
	return &Service{store: store}
}

func TestBackupRestoresSettingsAndSavesSafetySnapshot(t *testing.T) {
	source := testLibrary(t, t.TempDir())
	dest := testLibrary(t, t.TempDir())
	for _, err := range []error{
		source.SetTheme("dark"), source.SetLayout("list"), source.AddCategory("art", "海报"),
		source.AddModel("测试模型"), source.SetVariablePreset("sample-art", "霓虹", map[string]string{"主色调": "青色"}),
		dest.SetTheme("light"), dest.AddModel("旧模型"),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	backup, err := BackupJSON(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dest.ImportPrompts(string(backup), false); err != nil {
		t.Fatal(err)
	}
	if theme, _ := dest.GetTheme(); theme != "light" {
		t.Fatalf("merge changed theme: %q", theme)
	}
	result, err := dest.ImportPrompts(string(backup), true)
	if err != nil || filepath.Dir(result.Snapshot) != filepath.Join(dest.store.dir, "backups") {
		t.Fatalf("restore = %+v, %v", result, err)
	}
	theme, _ := dest.GetTheme()
	layout, _ := dest.GetLayout()
	models, _ := dest.CustomModels()
	categories, _ := dest.CustomCategories("art")
	presets, _ := dest.VariablePresets("sample-art")
	if theme != "dark" || layout != "list" || !slices.Equal(models, []string{"测试模型"}) ||
		!slices.Equal(categories, []string{"海报"}) || presets["霓虹"]["主色调"] != "青色" {
		t.Fatalf("settings did not round trip: %s %v %v %v", theme, models, categories, presets)
	}
	snapshot, err := os.ReadFile(result.Snapshot)
	if err != nil || !bytes.Contains(snapshot, []byte("旧模型")) {
		t.Fatalf("snapshot lacks original settings: %v", err)
	}
	selected, err := SelectedJSON(source, []string{"sample-art"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dest.ImportPrompts(string(selected), true); err == nil {
		t.Fatal("selected export must not replace both libraries")
	}
	if _, err := dest.ImportPrompts(strings.Replace(string(backup), `"settings":{`, `"settings_invalid":{`, 1), true); err == nil {
		t.Fatal("v3 without settings was accepted")
	}
}

func TestCleanupRechecksReferencesAndProtectsUndo(t *testing.T) {
	s := testLibrary(t, t.TempDir())
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 800, 200))); err != nil {
		t.Fatal(err)
	}
	art, err := s.SaveWithImages(Prompt{Kind: "art", Title: "图片", Body: "提示词"}, []string{base64.StdEncoding.EncodeToString(pngData.Bytes())})
	if err != nil {
		t.Fatal(err)
	}
	name := art.Images[0]
	key := scopedImage(art.Number, name)
	handler := s.store.imageAssets(http.NotFoundHandler())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, thumbnailRoute+key, nil))
	config, err := jpeg.DecodeConfig(response.Body)
	if response.Code != 200 || err != nil || config.Width != 400 || config.Height != 100 {
		t.Fatalf("thumbnail = %+v, %v, HTTP %d", config, err, response.Code)
	}
	batch, err := s.DeletePrompts([]string{art.ID})
	if err != nil {
		t.Fatal(err)
	}
	if files, err := s.ListUnreferencedImages(); err != nil || len(files) != 0 {
		t.Fatalf("undo images should be protected: %+v, %v", files, err)
	}
	if err := s.UndoDelete(batch); err != nil {
		t.Fatal(err)
	}
	art.Images = nil
	if _, err := s.SavePrompt(art); err != nil {
		t.Fatal(err)
	}
	files, err := s.ListUnreferencedImages()
	if err != nil || len(files) != 1 || files[0].Name != key {
		t.Fatalf("cleanup preview = %+v, %v", files, err)
	}
	art.Images = []string{name}
	if _, err := s.SavePrompt(art); err != nil {
		t.Fatal(err)
	}
	if result, err := s.CleanUnreferencedImages([]string{key}); err != nil || result.Removed != 0 || len(result.Skipped) != 1 {
		t.Fatalf("stale cleanup was not skipped: %+v, %v", result, err)
	}
	batch, err = s.DeletePrompts([]string{art.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.store.db.Exec("UPDATE deleted_prompts SET expires_at = ? WHERE batch = ?", time.Now().Unix()-1, batch); err != nil {
		t.Fatal(err)
	}
	result, err := s.CleanUnreferencedImages([]string{key, "../prompts.db"})
	if err != nil || result.Removed != 1 || result.Bytes != int64(pngData.Len()) || len(result.Skipped) != 1 {
		t.Fatalf("cleanup = %+v, %v", result, err)
	}
	for _, path := range []string{filepath.Join(s.store.imagesDir(), key), s.store.thumbnailPath(key)} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists: %v", path, err)
		}
	}
}

func TestMigrateExistingWorkspace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "含空格与符号 #")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "prompts.db"))
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := initialSchema(tx); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("PRAGMA user_version=1"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if err := os.WriteFile(filepath.Join(dir, "custom-image-models.json"), []byte(`["旧模型"]`), 0600); err != nil {
		t.Fatal(err)
	}
	s := testLibrary(t, dir)
	models, err := s.CustomModels()
	if err != nil || !slices.Equal(models, []string{"旧模型"}) {
		t.Fatalf("models = %v, %v", models, err)
	}
	var timeout int
	if err := s.store.db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil || timeout != 5000 {
		t.Fatalf("busy_timeout = %d, %v", timeout, err)
	}
	var version int
	if err := s.store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 3 {
		t.Fatalf("user_version = %d, %v", version, err)
	}
}

func TestCategoryAndTagChangesPreserveEntries(t *testing.T) {
	s := testLibrary(t, t.TempDir())
	if err := s.AddCategory("art", "临时"); err != nil {
		t.Fatal(err)
	}
	p, err := s.SavePrompt(Prompt{Kind: "art", Title: "测试", Body: "正文", Category: "临时", Tags: []string{"国风", "中国风"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.BatchFavorite([]string{p.ID, "missing"}, true); err == nil {
		t.Fatal("partial batch unexpectedly succeeded")
	}
	if err := s.RenameCategory("art", "临时", "新分类"); err != nil {
		t.Fatal(err)
	}
	if err := s.RenameTag("art", "国风", "中国风"); err != nil {
		t.Fatal(err)
	}
	items, err := s.ListPrompts()
	if err != nil || items[0].Category != "新分类" || items[0].Favorite || !slices.Equal(items[0].Tags, []string{"中国风"}) {
		t.Fatalf("renamed record = %+v, %v", items, err)
	}
	if err := s.DeleteCategory("art", "新分类"); err != nil {
		t.Fatal(err)
	}
	items, err = s.ListPrompts()
	if err != nil || len(items) != 3 || items[0].ID != p.ID || items[0].Category != "" {
		t.Fatalf("category removal = %+v, %v", items, err)
	}
	md, err := Markdown(Prompt{Kind: "art", Title: "测试", Body: "正文", Parameters: "16:9", Negative: "水印"})
	if err != nil || !bytes.Contains(md, []byte("## 参数备注\n\n16:9")) || !bytes.Contains(md, []byte("## 负向提示词\n\n水印")) {
		t.Fatalf("markdown = %s, %v", md, err)
	}
}

func TestPromptFoldersMigrateSharedImagesAndKeepNumbers(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "prompts.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	legacy := &promptStore{db: db, dir: dir}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := initialSchema(tx); err != nil {
		t.Fatal(err)
	}
	if err := legacy.addRecoverableDeletes(tx); err != nil {
		t.Fatal(err)
	}

	var raw bytes.Buffer
	if err := png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	name, err := checkedImage(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.writeImage(name, raw.Bytes()); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"shared-a", "shared-b"} {
		p := Prompt{ID: id, Kind: "art", Title: id, Body: "正文", Images: []string{name}, CreatedAt: timestamp(), UpdatedAt: timestamp()}
		if _, err := insertPrompt(tx, p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tx.Exec(`INSERT INTO deleted_prompts(batch,id,expires_at,record)
		SELECT 'legacy-batch',id,?,record FROM prompts WHERE id = 'shared-b';`, time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("DELETE FROM prompts WHERE id = 'shared-b'; PRAGMA user_version=2"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s := testLibrary(t, dir)
	if err := s.UndoDelete("legacy-batch"); err != nil {
		t.Fatal(err)
	}
	items, err := s.ListPrompts()
	if err != nil {
		t.Fatal(err)
	}
	shared := map[string]Prompt{}
	for _, item := range items {
		if item.ID != "shared-a" && item.ID != "shared-b" {
			continue
		}
		shared[item.ID] = item
		path := filepath.Join(s.store.imagesDir(), scopedImage(item.Number, name))
		if data, err := os.ReadFile(path); err != nil || !bytes.Equal(data, raw.Bytes()) {
			t.Fatalf("migrated picture for %s: %v", item.ID, err)
		}
	}
	a, b := shared["shared-a"], shared["shared-b"]
	if a.Number <= 0 || b.Number <= 0 || a.Number == b.Number {
		t.Fatalf("numbers are not distinct: %d, %d", a.Number, b.Number)
	}
	if _, err := os.Stat(filepath.Join(s.store.imagesDir(), name)); !os.IsNotExist(err) {
		t.Fatalf("old shared path remains: %v", err)
	}
	duplicate, err := s.SaveOrderedImages(Prompt{Kind: "art", Title: "副本", Body: "正文"}, []ImageInput{{Name: name, SourceNumber: a.Number}})
	if err != nil || duplicate.Number <= max(a.Number, b.Number) {
		t.Fatalf("copy number = %d, error = %v", duplicate.Number, err)
	}
	if _, err := os.Stat(filepath.Join(s.store.imagesDir(), scopedImage(duplicate.Number, name))); err != nil {
		t.Fatal(err)
	}
	a.Title = "改名"
	a.Number = b.Number // The caller cannot change the saved number.
	updated, err := s.SavePrompt(a)
	if err != nil || updated.Number != shared[a.ID].Number {
		t.Fatalf("changed number after edit: %d, %v", updated.Number, err)
	}
	batch, err := s.DeletePrompts([]string{updated.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UndoDelete(batch); err != nil {
		t.Fatal(err)
	}
	backup, err := BackupJSON(s)
	if err != nil {
		t.Fatal(err)
	}
	dest := testLibrary(t, t.TempDir())
	if _, err := dest.ImportPrompts(string(backup), true); err != nil {
		t.Fatal(err)
	}
	for id, original := range shared {
		var number int64
		if err := dest.store.db.QueryRow("SELECT number FROM prompt_numbers WHERE prompt_id = ?", id).Scan(&number); err != nil || number != original.Number {
			t.Fatalf("restored %s has number %d, want %d: %v", id, number, original.Number, err)
		}
	}
}
