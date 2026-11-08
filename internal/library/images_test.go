package library

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestArtworkImagesAndPortableBackup(t *testing.T) {
	newService := func() *Service {
		dir := t.TempDir()
		db, err := sql.Open("sqlite", filepath.Join(dir, "prompts.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		s := &Service{store: &promptStore{db: db, dir: dir}}
		if err := s.store.migrate(); err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := newService()
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	encoded := base64.StdEncoding.EncodeToString(pngData.Bytes())
	art, err := s.SaveWithImages(Prompt{Kind: "art", Title: "Sunlit garden", English: "Sunlit garden, soft shadows"}, []string{encoded, encoded})
	if err != nil || len(art.Images) != 1 || art.CreatedAt == "" {
		t.Fatalf("save: %+v, %v", art, err)
	}
	details := s.ImageDetails(art.Number, []string{"../prompts.db", art.Images[0]})
	want := ImageDetail{Name: art.Images[0], Bytes: int64(pngData.Len()), Width: 4, Height: 4, Format: "png"}
	if len(details) != 1 || details[0] != want {
		t.Fatalf("image details = %+v, want %+v", details, want)
	}
	for _, upload := range []string{"", encoded} {
		data, extension, err := ImageForExport(s, art.Number, art.Images[0], upload)
		if err != nil || extension != ".png" || !bytes.Equal(data, pngData.Bytes()) {
			t.Fatalf("exported image differs from original: %s, %v", extension, err)
		}
	}
	if _, _, err := ImageForExport(s, art.Number, "../prompts.db", ""); err == nil {
		t.Fatal("export accepted a non-image path")
	}
	art.Body = "阳光下的花园"
	updated, err := s.SaveWithImages(art, nil)
	if err != nil || updated.CreatedAt != art.CreatedAt || updated.UpdatedAt < art.UpdatedAt {
		t.Fatalf("update: %+v, %v", updated, err)
	}
	handler := s.store.imageAssets(http.NotFoundHandler())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, imageRoute+scopedImage(art.Number, art.Images[0]), nil))
	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), pngData.Bytes()) {
		t.Fatalf("serve image: %d, %q", response.Code, response.Body.Bytes())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, imageRoute+"../prompts.db", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("unexpected response for non-image path: %d", response.Code)
	}
	stats, err := s.StorageStats()
	if err != nil || stats.ImageFiles != 1 || stats.ImageBytes != int64(pngData.Len()) || stats.DBBytes == 0 {
		t.Fatalf("stats: %+v, %v", stats, err)
	}

	items, err := s.ListPrompts()
	if err != nil {
		t.Fatal(err)
	}
	legacyItems := slices.DeleteFunc(items, func(item Prompt) bool { return item.Kind == "text" || item.Kind == "video" })
	backup, err := s.store.encodeArchive(legacyItems, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	dest := newService()
	if preview, err := dest.PreviewImport(string(backup)); err != nil || !preview.CanRestore || preview.Added != 1 {
		t.Fatalf("preview: %+v, %v", preview, err)
	}
	if _, err := dest.ImportPrompts(string(backup), true); err != nil {
		t.Fatal(err)
	}
	restored, err := dest.ListPrompts()
	index := slices.IndexFunc(restored, func(item Prompt) bool { return item.ID == art.ID })
	if err != nil || len(restored) != 5 || index < 0 || restored[index].Body != art.Body || restored[index].English != art.English || len(restored[index].Images) != 1 {
		t.Fatalf("restored prompts: %+v, %v", restored, err)
	}
	got, err := os.ReadFile(filepath.Join(dest.store.imagesDir(), scopedImage(restored[index].Number, art.Images[0])))
	if err != nil || !bytes.Equal(got, pngData.Bytes()) {
		t.Fatalf("restored attachment differs: %v", err)
	}
	// A newly selected image can move ahead of one already on disk.
	var secondPNG bytes.Buffer
	if err := png.Encode(&secondPNG, image.NewRGBA(image.Rect(0, 0, 6, 6))); err != nil {
		t.Fatal(err)
	}
	secondName, err := checkedImage(secondPNG.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	edit := restored[index]
	edit.UpdatedAt = ""

	edit.Models = []string{"Midjourney", "FLUX"}
	reordered, err := dest.SaveOrderedImages(edit, []ImageInput{
		{Data: base64.StdEncoding.EncodeToString(secondPNG.Bytes())},
		{Name: art.Images[0]},
	})
	if err != nil || !slices.Equal(reordered.Images, []string{secondName, art.Images[0]}) {
		t.Fatalf("ordered images: %+v, %v", reordered.Images, err)
	}
	saved, err := dest.ListPrompts()
	if err != nil || !slices.Equal(saved[0].Models, edit.Models) || !slices.Equal(saved[0].Images, reordered.Images) {
		t.Fatalf("models or image order were not saved: %+v, %v", saved, err)
	}
	legacy := Prompt{Model: "Adobe Firefly"}
	legacy.normalizeModels()
	if !slices.Equal(legacy.Models, []string{"Adobe Firefly"}) {
		t.Fatalf("legacy model was not loaded: %+v", legacy)
	}

	if err := dest.AddModel("Studio Custom"); err != nil {
		t.Fatal(err)
	}
	if err := dest.AddModel("studio custom"); err == nil {
		t.Fatal("duplicate model was accepted")
	}
	models, err := dest.Models()
	if err != nil || models[len(models)-1] != "Studio Custom" {
		t.Fatalf("models: %+v, %v", models, err)
	}
	if err := dest.RenameModel("Studio Custom", "Studio Next"); err != nil {
		t.Fatal(err)
	}
	if len(models) < 6 || !slices.Equal(models[:6], []string{"Midjourney V6", "Midjourney V8.2", "GPT Image 2.0", "GPT Image 2.5", "Nano Banana 2", "Nano Banana 2.1"}) {
		t.Fatalf("versioned models: %v", models)
	}
	if err := dest.DeleteModel("Midjourney V6"); err == nil {
		t.Fatal("built-in model was deleted")
	}
	if err := dest.DeleteModel("Studio Next"); err != nil {
		t.Fatal(err)
	}
	if custom, err := dest.CustomModels(); err != nil || len(custom) != 0 {
		t.Fatalf("custom models: %+v, %v", custom, err)
	}
	if err := dest.AddCategory("art", "工作测试"); err != nil {
		t.Fatal(err)
	}
	artCategories, err := dest.CategoryOptions("art")
	if err != nil || !slices.Contains(artCategories, "工作测试") {
		t.Fatalf("art categories: %+v, %v", artCategories, err)
	}
	personaCategories, err := dest.CategoryOptions("persona")
	if err != nil || slices.Contains(personaCategories, "工作测试") {
		t.Fatalf("persona categories: %+v, %v", personaCategories, err)
	}

}

func TestReferenceImagesRoundTripAndCleanup(t *testing.T) {
	source := testLibrary(t, t.TempDir())
	imageData := func(width int) string {
		t.Helper()
		var data bytes.Buffer
		if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, width, 3))); err != nil {
			t.Fatal(err)
		}
		return base64.StdEncoding.EncodeToString(data.Bytes())
	}
	item, err := source.SaveOrderedImages(Prompt{Kind: "art", Title: "两组图片", Body: "正文"}, []ImageInput{
		{Data: imageData(3)}, {Data: imageData(4), Reference: true},
	})
	if err != nil || len(item.Images) != 1 || len(item.ReferenceImages) != 1 || item.Images[0] == item.ReferenceImages[0] {
		t.Fatalf("image groups = %v, %v, %v", item.Images, item.ReferenceImages, err)
	}
	refName := item.ReferenceImages[0]
	backup, err := BackupJSON(source)
	if err != nil {
		t.Fatal(err)
	}
	dest := testLibrary(t, t.TempDir())
	if _, err := dest.ImportPrompts(string(backup), true); err != nil {
		t.Fatal(err)
	}
	restored, err := dest.ListPrompts()
	if err != nil || !slices.ContainsFunc(restored, func(p Prompt) bool {
		return p.ID == item.ID && slices.Equal(p.Images, item.Images) && slices.Equal(p.ReferenceImages, item.ReferenceImages)
	}) {
		t.Fatalf("reference group was not restored: %v", err)
	}
	copy, err := dest.SaveOrderedImages(Prompt{Kind: "art", Title: "副本", Body: "正文"}, []ImageInput{
		{Name: item.Images[0], SourceNumber: item.Number}, {Name: refName, SourceNumber: item.Number, Reference: true},
	})
	if err != nil || !slices.Equal(copy.ReferenceImages, item.ReferenceImages) {
		t.Fatalf("reference group was not copied: %v", err)
	}
	batch, err := dest.DeletePrompts([]string{item.ID})
	if err != nil {
		t.Fatal(err)
	}
	if orphaned, err := dest.ListUnreferencedImages(); err != nil || len(orphaned) != 0 {
		t.Fatalf("referenced files listed for cleanup: %+v, %v", orphaned, err)
	}
	if result, err := dest.DeleteTrashPermanently([]TrashRef{{Batch: batch, ID: item.ID}}); err != nil || result.Deleted != 1 || result.CleanupPending {
		t.Fatalf("purge = %+v, %v", result, err)
	}
	for _, check := range []struct {
		number int64
		exists bool
	}{{item.Number, false}, {copy.Number, true}} {
		path, err := dest.store.imagePath(scopedImage(check.number, refName))
		if err != nil {
			t.Fatal(err)
		}
		_, err = os.Stat(path)
		if (check.exists && err != nil) || (!check.exists && !os.IsNotExist(err)) {
			t.Fatalf("reference file for %d: %v", check.number, err)
		}
	}
}
