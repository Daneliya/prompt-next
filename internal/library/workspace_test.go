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
		source.AddModel("art", "测试模型"), source.AddModel("video", "测试视频模型"), source.SetVariablePreset("sample-art", "霓虹", map[string]string{"主色调": "青色"}),
		source.SetViewFields("cards", []string{"title", "body"}),
		source.SetViewFields("persona:cards", []string{"title", "characters", "updated", "author"}),
		dest.SetTheme("light"), dest.AddModel("art", "旧模型"), dest.AddModel("video", "旧视频模型"),
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
	imageModels, _ := dest.Models("art")
	videoModels, _ := dest.Models("video")
	categories, _ := dest.CustomCategories("art")
	presets, _ := dest.VariablePresets("sample-art")
	if theme != "dark" || layout != "list" || !slices.Equal(imageModels.Custom, []string{"测试模型"}) || !slices.Equal(videoModels.Custom, []string{"测试视频模型"}) ||
		!slices.Equal(categories, []string{"海报"}) || presets["霓虹"]["主色调"] != "青色" {
		t.Fatalf("settings did not round trip: %s %v %v %v", theme, imageModels.Custom, categories, presets)
	}
	cards, err := dest.ViewFields("cards")
	if err != nil || !slices.Equal(cards, []string{"title", "body"}) {
		t.Fatalf("shared card fields = %v, %v", cards, err)
	}
	persona, err := dest.ViewFields("persona:cards")
	if err != nil || !slices.Equal(persona, []string{"title", "characters", "updated", "author"}) {
		t.Fatalf("persona card fields = %v, %v", persona, err)
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
	if _, err := tx.Exec(`UPDATE prompts SET record = json_set(record, '$.author', '旧作者') WHERE id = 'sample-art'`); err != nil {
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
	authors, err := s.ListAuthors()
	if err != nil || !slices.Equal(authors, []Author{{SourceName: "旧作者", Name: "旧作者"}}) {
		t.Fatalf("migrated authors = %+v, %v", authors, err)
	}
	catalog, err := s.Models("art")
	models := catalog.Custom
	if err != nil || !slices.Equal(models, []string{"旧模型"}) {
		t.Fatalf("models = %v, %v", models, err)
	}
	var timeout int
	if err := s.store.db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil || timeout != 5000 {
		t.Fatalf("busy_timeout = %d, %v", timeout, err)
	}
	var version int
	if err := s.store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 7 {
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
	if err != nil || len(items) != 5 || items[0].ID != p.ID || items[0].Category != "" {
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

func TestTextLibraryBackupsAndWaterfallPreferences(t *testing.T) {
	source := testLibrary(t, t.TempDir())
	dest := testLibrary(t, t.TempDir())
	if _, err := source.store.db.Exec("DELETE FROM prompts WHERE kind IN ('text', 'video')"); err != nil {
		t.Fatal(err)
	}
	legacy, err := BackupJSON(source)
	if err != nil {
		t.Fatal(err)
	}
	legacyV3 := strings.Replace(string(legacy), `"version":5`, `"version":3`, 1)

	item, err := source.SavePrompt(Prompt{Kind: "text", Title: "摘要", Body: "用三句话概括 {{文章}}", Category: "写作"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.SaveWithImages(Prompt{Kind: "text", Title: "摘要", Body: "正文"}, []string{"unused"}); err == nil {
		t.Fatal("text prompt accepted an attachment")
	}
	if err := source.SetLayout("waterfall"); err != nil {
		t.Fatal(err)
	}
	if err := source.SetViewFields("waterfall", []string{"title", "tags"}); err != nil {
		t.Fatal(err)
	}
	if err := source.AddCategory("text", "摘要"); err != nil {
		t.Fatal(err)
	}
	if err := source.SetVariablePreset(item.ID, "示例", map[string]string{"文章": "一段正文"}); err != nil {
		t.Fatal(err)
	}
	backup, err := BackupJSON(source)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := dest.ImportPrompts(string(backup), true); err != nil || !result.IncludesText {
		t.Fatalf("restore = %+v, %v", result, err)
	}
	layout, err := dest.GetLayout()
	if err != nil || layout != "waterfall" {
		t.Fatalf("restored layout = %q, %v", layout, err)
	}
	if result, err := dest.ImportPrompts(legacyV3, true); err != nil || result.IncludesText {
		t.Fatalf("legacy restore = %+v, %v", result, err)
	}
	items, err := dest.ListPrompts()
	if err != nil || !slices.ContainsFunc(items, func(saved Prompt) bool {
		return saved.ID == item.ID && saved.Body == item.Body && saved.Kind == "text" && saved.Number == item.Number
	}) {
		t.Fatalf("legacy restore did not preserve the text prompt: %v", err)
	}
	fields, err := dest.ViewFields("waterfall")
	if err != nil || !slices.Equal(fields, []string{"title", "tags"}) {
		t.Fatalf("waterfall fields = %v, %v", fields, err)
	}
	categories, err := dest.CustomCategories("text")
	if err != nil || !slices.Contains(categories, "摘要") {
		t.Fatalf("text categories = %v, %v", categories, err)
	}
	presets, err := dest.VariablePresets(item.ID)
	if err != nil || presets["示例"]["文章"] != "一段正文" {
		t.Fatalf("text presets = %v, %v", presets, err)
	}
}

func TestLibraryExamplesAndPromptNumbers(t *testing.T) {
	s := testLibrary(t, t.TempDir())
	legacy, err := s.ListPrompts()
	if err != nil {
		t.Fatal(err)
	}
	for id, kind := range map[string]string{"sample-text": "text", "sample-video": "video"} {
		if !slices.ContainsFunc(legacy, func(p Prompt) bool {
			return p.ID == id && p.Kind == kind && p.Body != "" && p.Number >= firstNewPromptNumber
		}) {
			t.Fatalf("missing %s example", kind)
		}
	}
	prompt := Prompt{Kind: "text", Title: "编号", Body: "正文"}
	first, err := s.SavePrompt(prompt)
	if err != nil || first.Number != 100003 {
		t.Fatalf("first prompt = %d, %v", first.Number, err)
	}
	if _, err := s.DeletePrompts([]string{first.ID}); err != nil {
		t.Fatal(err)
	}
	second, err := s.SavePrompt(prompt)
	if err != nil || second.Number != first.Number+1 {
		t.Fatalf("second prompt = %d, %v", second.Number, err)
	}
	for _, original := range legacy {
		var number int64
		if err := s.store.db.QueryRow("SELECT number FROM prompt_numbers WHERE prompt_id = ?", original.ID).Scan(&number); err != nil || number != original.Number {
			t.Fatalf("existing prompt %s changed number: %d, %v", original.ID, number, err)
		}
	}
	if _, err := s.DeletePrompts([]string{"sample-text"}); err != nil {
		t.Fatal(err)
	}
	if err := s.store.migrate(); err != nil {
		t.Fatal(err)
	}
	items, err := s.ListPrompts()
	if err != nil || slices.ContainsFunc(items, func(p Prompt) bool { return p.ID == "sample-text" }) {
		t.Fatalf("reopening regenerated a deleted example: %v", err)
	}
}

func TestPromptCollectionsRestoreAndUndo(t *testing.T) {
	source := testLibrary(t, t.TempDir())
	collection, err := source.SaveCollection(PromptCollection{
		Name: "雨夜变体", Description: "测试合集说明", PromptIDs: []string{"sample-art", "sample-persona", "sample-art"},
	})
	if err != nil || collection.ID == "" || collection.UpdatedAt == "" || !slices.Equal(collection.PromptIDs, []string{"sample-art", "sample-persona"}) {
		t.Fatalf("saved collection = %+v, %v", collection, err)
	}
	if _, err := source.SaveCollection(PromptCollection{Name: "无效", PromptIDs: []string{"missing"}}); err == nil {
		t.Fatal("collection accepted a missing new member")
	}
	batch, err := source.DeletePrompts([]string{"sample-art"})
	if err != nil {
		t.Fatal(err)
	}
	collection.Name = "雨夜系列"
	slices.Reverse(collection.PromptIDs)
	collection, err = source.SaveCollection(collection)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.UndoDelete(batch); err != nil {
		t.Fatal(err)
	}
	backup, err := BackupJSON(source)
	if err != nil {
		t.Fatal(err)
	}
	dest := testLibrary(t, t.TempDir())
	if _, err := dest.ImportPrompts(string(backup), true); err != nil {
		t.Fatal(err)
	}
	collections, err := dest.ListCollections()
	if err != nil || len(collections) != 1 || collections[0].Name != collection.Name || collections[0].Description != collection.Description ||
		collections[0].UpdatedAt != collection.UpdatedAt || !slices.Equal(collections[0].PromptIDs, collection.PromptIDs) {
		t.Fatalf("restored collections = %+v, %v", collections, err)
	}
	if err := dest.DeleteCollection(collection.ID); err != nil {
		t.Fatal(err)
	}
	items, err := dest.ListPrompts()
	if err != nil || len(items) != 4 {
		t.Fatalf("deleting collection changed its prompts: %v, %v", items, err)
	}
}

func TestVideoAuthorAndTrashRoundTrip(t *testing.T) {
	source := testLibrary(t, t.TempDir())
	if _, err := source.store.db.Exec("DELETE FROM prompts WHERE kind = 'video'"); err != nil {
		t.Fatal(err)
	}
	legacy, err := BackupJSON(source)
	if err != nil {
		t.Fatal(err)
	}
	video, err := source.SavePrompt(Prompt{
		Kind: "video", Title: "城市分镜", English: "A slow dolly shot",
		Author: "作者", AuthorURL: "https://example.com/author", VideoURL: "https://example.com/video",
	})
	if err != nil {
		t.Fatal(err)
	}
	video.AuthorURL = "https://example.com/updated-author"
	video, err = source.SavePrompt(video)
	if err != nil {
		t.Fatal(err)
	}
	firstBatch, err := source.DeletePrompts([]string{video.ID, "sample-art"})
	if err != nil {
		t.Fatal(err)
	}
	secondBatch, err := source.DeletePrompts([]string{"sample-persona"})
	if err != nil {
		t.Fatal(err)
	}
	trash, err := source.ListTrash()
	if err != nil || len(trash) != 3 {
		t.Fatalf("trash = %+v, %v", trash, err)
	}
	refs := []TrashRef{{Batch: firstBatch, ID: video.ID}, {Batch: secondBatch, ID: "sample-persona"}}
	if count, err := source.RestoreTrash(refs); err != nil || count != 2 {
		t.Fatalf("restore from multiple batches = %d, %v", count, err)
	}
	trash, err = source.ListTrash()
	if err != nil || len(trash) != 1 || trash[0].Ref.ID != "sample-art" {
		t.Fatalf("trash after partial recovery = %+v, %v", trash, err)
	}
	full, err := BackupJSON(source)
	if err != nil {
		t.Fatal(err)
	}
	dest := testLibrary(t, t.TempDir())
	if _, err := dest.ImportPrompts(string(full), true); err != nil {
		t.Fatal(err)
	}
	legacyV4 := strings.Replace(string(legacy), `"version":5`, `"version":4`, 1)
	if _, err := dest.ImportPrompts(legacyV4, true); err != nil {
		t.Fatal(err)
	}
	items, err := dest.ListPrompts()
	if err != nil || !slices.ContainsFunc(items, func(item Prompt) bool {
		return item.ID == video.ID && item.Number == video.Number && item.Author == video.Author &&
			item.AuthorURL == video.AuthorURL && item.VideoURL == video.VideoURL
	}) {
		t.Fatalf("video or attribution was lost: %v", err)
	}
	if _, err := dest.SavePrompt(Prompt{Kind: "text", Title: "链接", Body: "测试", AuthorURL: "javascript:alert(1)"}); err == nil {
		t.Fatal("unsafe author URL accepted")
	}
}

func TestEditableDatesAndRecentPromptCounts(t *testing.T) {
	s := testLibrary(t, t.TempDir())
	local := time.FixedZone("UTC+8", 8*60*60)
	today := time.Date(2031, 6, 14, 10, 0, 0, 0, local)
	var entry Prompt
	for _, kind := range []string{"text", "art", "video", "persona"} {
		saved, err := s.SavePrompt(Prompt{
			Kind: kind, Title: kind, Body: "正文", CreatedAt: "2031-06-13T23:30:00+08:00", UpdatedAt: "2031-06-14T08:30:00+08:00",
		})
		if err != nil || saved.CreatedAt != "2031-06-13T15:30:00.000000000Z" || saved.UpdatedAt != "2031-06-14T00:30:00.000000000Z" {
			t.Fatalf("save dates = %s, %s, %v", saved.CreatedAt, saved.UpdatedAt, err)
		}
		entry = saved
	}
	entry.CreatedAt = "2031-06-13T16:01:00Z" // The next day in the local timezone.
	entry.UpdatedAt = "2031-06-14T01:02:03Z"
	saved, err := s.SavePrompt(entry)
	if err != nil || saved.UpdatedAt != "2031-06-14T01:02:03.000000000Z" {
		t.Fatalf("edit dates = %+v, %v", saved, err)
	}
	items, err := s.ListPrompts()
	if err != nil || items[0].ID != entry.ID || items[0].CreatedAt != saved.CreatedAt || items[0].UpdatedAt != saved.UpdatedAt {
		t.Fatalf("dates not persisted or ordered: %v", err)
	}
	stats, err := s.storageStatsAt(today)
	if err != nil || len(stats.RecentPrompts) != 14 ||
		stats.RecentPrompts[12] != (DailyPromptCount{Date: "2031-06-13", Count: 3, Text: 1, Art: 1, Video: 1}) ||
		stats.RecentPrompts[13] != (DailyPromptCount{Date: "2031-06-14", Count: 1, Persona: 1}) {
		t.Fatalf("recent prompts = %+v, %v", stats.RecentPrompts, err)
	}
	saved.CreatedAt, saved.UpdatedAt = "", ""
	auto, err := s.SavePrompt(saved)
	if err != nil || auto.CreatedAt != "2031-06-13T16:01:00.000000000Z" || auto.UpdatedAt == "" || auto.UpdatedAt == entry.UpdatedAt {
		t.Fatalf("auto dates = %s, %s, %v", auto.CreatedAt, auto.UpdatedAt, err)
	}
	entry.UpdatedAt = "invalid"
	if _, err := s.SavePrompt(entry); err == nil {
		t.Fatal("invalid timestamp accepted")
	}
}

func TestModelCatalogsAreIndependent(t *testing.T) {
	s := testLibrary(t, t.TempDir())
	if err := putSetting(s.store.db, customModelKey, stringsJSON([]string{"Shared Legacy Name"})); err != nil {
		t.Fatal(err)
	}
	art, err := s.Models("art")
	if err != nil || !slices.Equal(art.Custom, []string{"Shared Legacy Name"}) || slices.Contains(art.BuiltIn, "Muse Video") {
		t.Fatalf("image catalog = %+v, %v", art, err)
	}
	video, err := s.Models("video")
	if err != nil || len(video.Custom) != 0 || !slices.Contains(video.BuiltIn, "Seedance 2.5") ||
		!slices.Contains(video.BuiltIn, "Muse Video") || slices.Contains(video.BuiltIn, "GPT Image 2.6") {
		t.Fatalf("video catalog = %+v, %v", video, err)
	}
	if err := s.AddModel("video", "Shared Legacy Name"); err != nil {
		t.Fatal(err)
	}
	if err := s.RenameModel("video", "Shared Legacy Name", "Custom Video"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteModel("art", "Custom Video"); err == nil {
		t.Fatal("video model was visible to image settings")
	}
	art, err = s.Models("art")
	if err != nil || !slices.Equal(art.Custom, []string{"Shared Legacy Name"}) {
		t.Fatalf("video edit changed image settings: %+v, %v", art, err)
	}
	video, err = s.Models("video")
	if err != nil || !slices.Equal(video.Custom, []string{"Custom Video"}) {
		t.Fatalf("video custom model = %+v, %v", video, err)
	}
}

func TestAuthorProfilesKeepPromptSourcesAndSurviveRestore(t *testing.T) {
	source := testLibrary(t, t.TempDir())
	for _, kind := range []string{"text", "art", "video", "persona"} {
		_, err := source.SavePrompt(Prompt{
			Kind: kind, Title: kind, Body: "正文", Author: " 同一作者 ",
			AuthorURL: "https://example.com/prompts/" + kind,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	want := Author{SourceName: "同一作者", Name: "显示名称", Bio: "作者简介"}
	if _, err := source.SaveAuthor(want); err != nil {
		t.Fatal(err)
	}
	backup, err := BackupJSON(source)
	if err != nil {
		t.Fatal(err)
	}
	dest := testLibrary(t, t.TempDir())
	if _, err := dest.ImportPrompts(string(backup), true); err != nil {
		t.Fatal(err)
	}
	authors, err := dest.ListAuthors()
	if err != nil || !slices.Equal(authors, []Author{want}) {
		t.Fatalf("restored author profiles: %+v, %v", authors, err)
	}
	items, err := dest.ListPrompts()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, item := range items {
		if item.Author != want.SourceName {
			continue
		}
		count++
		if item.AuthorURL != "https://example.com/prompts/"+item.Kind {
			t.Fatalf("source link changed for %s: %s", item.Kind, item.AuthorURL)
		}
	}
	if count != 4 {
		t.Fatalf("linked prompts = %d, want 4", count)
	}
}

func TestTargetedPromptQueriesAndStatsWithoutMedia(t *testing.T) {
	s := testLibrary(t, t.TempDir())
	saved, err := s.SavePrompt(Prompt{Kind: "text", Title: "查询样例", Body: "正文"})
	if err != nil {
		t.Fatal(err)
	}
	byKind, err := s.ListPromptsByKind("text")
	if err != nil || !slices.ContainsFunc(byKind, func(p Prompt) bool { return p.ID == saved.ID }) {
		t.Fatalf("query by kind = %+v, %v", byKind, err)
	}
	for _, item := range byKind {
		if item.Kind != "text" {
			t.Fatalf("unexpected kind: %s", item.Kind)
		}
	}
	byID, err := s.PromptsByIDs([]string{saved.ID, "missing", saved.ID})
	if err != nil || len(byID) != 1 || byID[0].ID != saved.ID {
		t.Fatalf("query by ID = %+v, %v", byID, err)
	}
	var cover bytes.Buffer
	if err := png.Encode(&cover, image.NewRGBA(image.Rect(0, 0, 16, 8))); err != nil {
		t.Fatal(err)
	}
	name, err := checkedImage(cover.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.store.writeImage(scopedImage(saved.Number, name), cover.Bytes()); err != nil {
		t.Fatal(err)
	}
	if ratio := s.CoverDimensions(map[int64]string{saved.Number: name})[saved.Number]; ratio != 2 {
		t.Fatalf("cover ratio = %v", ratio)
	}
	snapshot, err := s.StorageSnapshot(false)
	if err != nil || snapshot.DBBytes == 0 || snapshot.ImageBytes != 0 {
		t.Fatalf("snapshot without media = %+v, %v", snapshot, err)
	}
	if full, err := s.StorageSnapshot(true); err != nil || full.ImageFiles != 1 || full.ImageBytes != int64(cover.Len()) {
		t.Fatalf("full media snapshot = %+v, %v", full, err)
	}
}
