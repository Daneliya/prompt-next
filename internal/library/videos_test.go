package library

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestVideoAttachmentRangeAndBackup(t *testing.T) {
	source := testLibrary(t, t.TempDir())
	// A minimal MP4 container signature suffices for testing byte preservation.
	data := []byte("\x00\x00\x00\x14ftypisom\x00\x00\x02\x00isom")
	path := filepath.Join(t.TempDir(), "sample.mp4")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(t.TempDir(), "second.mp4")
	secondData := append(bytes.Clone(data), '2')
	if err := os.WriteFile(second, secondData, 0600); err != nil {
		t.Fatal(err)
	}
	item, err := SaveVideoPrompt(source, Prompt{Kind: "video", Title: "视频", Body: "分镜"}, nil, 0, []string{path, second})
	if err != nil || len(item.Videos) != 2 {
		t.Fatalf("saved videos = %v, %v", item.Videos, err)
	}
	copy, err := SaveVideoPrompt(source, Prompt{Kind: "video", Title: "副本", Body: "分镜", Videos: item.Videos}, nil, item.Number, nil)
	if err != nil || copy.Number == item.Number || !slices.Equal(copy.Videos, item.Videos) {
		t.Fatalf("copied video = %+v, %v", copy, err)
	}
	archive, err := BackupJSON(source)
	if err != nil {
		t.Fatal(err)
	}
	dest := testLibrary(t, t.TempDir())
	if _, err := dest.ImportPrompts(string(archive), true); err != nil {
		t.Fatal(err)
	}
	stats, err := dest.StorageStats()
	if err != nil || stats.VideoFiles != 4 || stats.VideoBytes != int64(2*(len(data)+len(secondData))) {
		t.Fatalf("video storage = %+v, %v", stats, err)
	}
	details := dest.VideoDetails(item.Number, item.Videos)
	if len(details) != 2 || details[0].Bytes != int64(len(data)) || details[1].Bytes != int64(len(secondData)) {
		t.Fatalf("video details = %+v", details)
	}
	kept := item.Videos[1]
	item.Videos = []string{kept}
	if updated, err := SaveVideoPrompt(dest, item, nil, item.Number, nil); err != nil || !slices.Equal(updated.Videos, []string{kept}) {
		t.Fatalf("edit video attachments = %v, %v", updated.Videos, err)
	}
	req := httptest.NewRequest(http.MethodGet, videoRoute+scopedImage(item.Number, kept), nil)
	req.Header.Set("Range", "bytes=4-11")
	response := httptest.NewRecorder()
	ImageAssets(dest)(http.NotFoundHandler()).ServeHTTP(response, req)
	if response.Code != http.StatusPartialContent || response.Header().Get("Content-Type") != "video/mp4" ||
		!bytes.Equal(response.Body.Bytes(), data[4:12]) {
		t.Fatalf("range response = %d %v %q", response.Code, response.Header(), response.Body.Bytes())
	}
}

func TestPurgeTrashPreservesMediaForRestoredCopy(t *testing.T) {
	s := testLibrary(t, t.TempDir())
	data := []byte("\x00\x00\x00\x14ftypisom\x00\x00\x02\x00isom")
	file := filepath.Join(t.TempDir(), "clip.mp4")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	item, err := SaveVideoPrompt(s, Prompt{Kind: "video", Title: "影片", Body: "分镜"}, nil, 0, []string{file})
	if err != nil {
		t.Fatal(err)
	}
	backup, err := BackupJSON(s)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := s.DeletePrompts([]string{item.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportPrompts(string(backup), false); err != nil {
		t.Fatal(err)
	}
	ref := TrashRef{Batch: batch, ID: item.ID}
	if result, err := s.DeleteTrashPermanently([]TrashRef{ref}); err != nil || result.Deleted != 1 || result.CleanupPending {
		t.Fatalf("purge = %+v, %v", result, err)
	}
	path, err := s.store.videoPath(scopedImage(item.Number, item.Videos[0]))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("active media was removed: %v", err)
	}
	batch, err = s.DeletePrompts([]string{item.ID})
	if err != nil {
		t.Fatal(err)
	}
	ref.Batch = batch
	if result, err := s.DeleteTrashPermanently([]TrashRef{ref}); err != nil || result.Deleted != 1 || result.CleanupPending {
		t.Fatalf("second purge = %+v, %v", result, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unreferenced video remains: %v", err)
	}
	if count, err := s.RestoreTrash([]TrashRef{ref}); err != nil || count != 0 {
		t.Fatalf("permanently deleted record restored: %d, %v", count, err)
	}
}

func TestLegacySingleVideoRecordAndBackup(t *testing.T) {
	source := testLibrary(t, t.TempDir())
	data := []byte("\x00\x00\x00\x14ftypisom\x00\x00\x02\x00isom")
	file := filepath.Join(t.TempDir(), "clip.mp4")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	item, err := SaveVideoPrompt(source, Prompt{Kind: "video", Title: "旧版视频", Body: "分镜"}, nil, 0, []string{file})
	if err != nil {
		t.Fatal(err)
	}
	legacy := item
	legacy.Video, legacy.Videos = item.Videos[0], nil
	record, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.store.db.Exec("UPDATE prompts SET record = ? WHERE id = ?", string(record), legacy.ID); err != nil {
		t.Fatal(err)
	}
	check := func(s *Service) {
		t.Helper()
		items, err := s.ListPrompts()
		if err != nil || !slices.ContainsFunc(items, func(p Prompt) bool {
			return p.ID == legacy.ID && slices.Equal(p.Videos, item.Videos)
		}) {
			t.Fatalf("legacy attachment missing: %v", err)
		}
	}
	check(source)
	settings := map[string]string{}
	backup, err := json.Marshal(backupFile{
		Format: legacyBackupFormat, Version: 6, Settings: &settings, Items: []Prompt{legacy},
		Videos: map[string][]byte{legacy.Video: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	dest := testLibrary(t, t.TempDir())
	if _, err := dest.ImportPrompts(string(backup), true); err != nil {
		t.Fatal(err)
	}
	check(dest)
}
