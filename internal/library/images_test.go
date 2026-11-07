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
	"time"
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
	location := time.FixedZone("UTC+8", 8*60*60)
	today := time.Date(2026, time.November, 7, 10, 0, 0, 0, location)
	yesterday := today.AddDate(0, 0, -1)
	if err := os.Chtimes(filepath.Join(s.store.imagesDir(), scopedImage(art.Number, art.Images[0])), yesterday, yesterday); err != nil {
		t.Fatal(err)
	}
	trend, err := s.storageStatsAt(today)
	if err != nil || len(trend.RecentImages) != 14 ||
		trend.RecentImages[12] != (DailyImageCount{Date: "2026-11-06", Count: 1}) ||
		trend.RecentImages[13] != (DailyImageCount{Date: "2026-11-07", Count: 0}) {
		t.Fatalf("unexpected daily image counts: %+v, %v", trend.RecentImages, err)
	}

	items, err := s.ListPrompts()
	if err != nil {
		t.Fatal(err)
	}
	backup, err := s.store.encodeArchive(items, nil, false)
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
	if err != nil || len(restored) != 3 || restored[0].Body != art.Body || restored[0].English != art.English || len(restored[0].Images) != 1 {
		t.Fatalf("restored prompts: %+v, %v", restored, err)
	}
	got, err := os.ReadFile(filepath.Join(dest.store.imagesDir(), scopedImage(restored[0].Number, art.Images[0])))
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
	edit := restored[0]
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
	if err := dest.DeleteModel("Midjourney"); err == nil {
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
