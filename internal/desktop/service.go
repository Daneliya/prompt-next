package desktop

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/Daneliya/prompt-next/internal/fileio"
	"github.com/Daneliya/prompt-next/internal/library"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const githubURL = "https://github.com/Daneliya"

type PromptService struct {
	*library.Service
	OnLanguageChange func(string)
	Version          string
	importMu         sync.Mutex
	chosenImport     string
	videoMu          sync.Mutex
	chosenVideos     map[string]string
	locationMu       sync.Mutex
	imageUploadMu    sync.Mutex
	imageUploads     map[string][]byte
	imageUploadBytes int
}

func (s *PromptService) SaveOrderedImages(prompt library.Prompt, attachments []library.ImageInput) (library.Prompt, error) {
	if prompt.Kind == "art" && !slices.ContainsFunc(attachments, func(input library.ImageInput) bool { return !input.Reference }) {
		return library.Prompt{}, errors.New(localizedLabel(ReadLanguage(), "请至少添加一张效果图", "Add at least one result image"))
	}
	uploads, err := s.resolveImageUploads(attachments)
	if err != nil {
		return library.Prompt{}, err
	}
	return s.Service.SaveOrderedImages(prompt, uploads)
}

func (s *PromptService) OpenDataFolder() error {
	return application.Get().Env.OpenFileManager(library.DataDirectory(s.Service), false)
}

func (s *PromptService) OpenWebLink(link string) error {
	if !library.ValidWebLink(link) {
		return errors.New("链接须为有效的 HTTP 或 HTTPS 网址")
	}
	return application.Get().Browser.OpenURL(link)
}

func (s *PromptService) OpenGitHub() error {
	return application.Get().Browser.OpenURL(githubURL)
}

func (s *PromptService) CopyText(text string) error {
	if !application.Get().Clipboard.SetText(text) {
		return errors.New("写入系统剪贴板失败")
	}
	return nil
}

func (s *PromptService) ExportBackup() (bool, error) {
	data, err := library.BackupJSON(s.Service)
	if err != nil {
		return false, err
	}
	return saveExport("大肥鱼提示词备份.json", "JSON 备份", "*.json", data)
}

func (s *PromptService) ExportMarkdown(p library.Prompt) (bool, error) {
	data, err := library.Markdown(p)
	if err != nil {
		return false, err
	}
	return saveExport("提示词.md", "Markdown", "*.md", data)
}

func (s *PromptService) ExportImage(number int64, name, upload string) (bool, error) {
	data, extension, err := library.ImageForExport(s.Service, number, name, upload)
	if err != nil {
		return false, err
	}
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if name == "" || name == "." || name == ".." || name == "/" {
		name = "prompt-image"
	}
	return saveExport(name+extension, localizedLabel(ReadLanguage(), "原图", "Original Image"), "*"+extension, data)
}

func saveExport(name, label, pattern string, data []byte) (bool, error) {
	path, err := application.Get().Dialog.SaveFile().SetFilename(name).AddFilter(label, pattern).PromptForSingleSelection()
	if err != nil || path == "" {
		return false, err
	}
	err = fileio.WriteAtomic(path, data)
	return err == nil, err
}

func (s *PromptService) ExportSelected(ids []string) (bool, error) {
	data, err := library.SelectedJSON(s.Service, ids)
	if err != nil {
		return false, err
	}
	return saveExport("Prompt Next 选中提示词.json", "JSON 提示词", "*.json", data)
}

type AppMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (s *PromptService) AppInfo() AppMetadata {
	return AppMetadata{Name: "Prompt Next", Version: s.Version}
}
