package desktop

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Daneliya/prompt-next/internal/library"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ChosenVideo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

func (s *PromptService) ChooseVideoFiles(keptIDs []string, existingCount int) ([]ChosenVideo, error) {
	paths, err := application.Get().Dialog.OpenFile().SetTitle(localizedLabel(ReadLanguage(), "选择本地视频", "Choose Local Videos")).
		AddFilter(localizedLabel(ReadLanguage(), "MP4 或 WebM", "MP4 or WebM"), "*.mp4;*.webm").PromptForMultipleSelection()
	if err != nil || len(paths) == 0 {
		return nil, err
	}
	if existingCount < 0 || existingCount > library.MaxPromptVideos ||
		len(keptIDs)+len(paths) > library.MaxPromptVideos-existingCount {
		return nil, errors.New("每条视频提示词最多可附加 12 段视频")
	}
	s.videoMu.Lock()
	defer s.videoMu.Unlock()
	pending := make(map[string]string)
	for _, id := range keptIDs {
		if path := s.chosenVideos[id]; path != "" {
			pending[id] = path
		}
	}
	selected := make([]ChosenVideo, 0, len(paths))
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > library.MaxVideoBytes || (ext != ".mp4" && ext != ".webm") {
			return nil, errors.New("请选择 64 MB 以内的 MP4 或 WebM 文件")
		}
		id := rand.Text()
		pending[id] = path
		selected = append(selected, ChosenVideo{ID: id, Name: filepath.Base(path), Bytes: info.Size()})
	}
	s.chosenVideos = pending
	return selected, nil
}

func (s *PromptService) SaveVideoWithFiles(prompt library.Prompt, attachments []library.ImageInput, sourceNumber int64, chosenIDs []string) (library.Prompt, error) {
	if prompt.Kind == "video" && len(prompt.Videos) == 0 && (prompt.Videos != nil || prompt.Video == "") && len(chosenIDs) == 0 {
		return library.Prompt{}, errors.New(localizedLabel(ReadLanguage(), "请至少添加一段效果视频", "Add at least one result video"))
	}
	s.videoMu.Lock()
	defer s.videoMu.Unlock()
	paths := make([]string, 0, len(chosenIDs))
	for _, id := range chosenIDs {
		path := s.chosenVideos[id]
		if path == "" {
			return library.Prompt{}, errors.New("请重新选择视频文件")
		}
		paths = append(paths, path)
	}
	uploads, err := s.resolveImageUploads(attachments)
	if err != nil {
		return library.Prompt{}, err
	}
	saved, err := library.SaveVideoPrompt(s.Service, prompt, uploads, sourceNumber, paths)
	if err == nil {
		s.chosenVideos = nil
	}
	return saved, err
}
