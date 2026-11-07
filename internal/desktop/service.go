package desktop

import (
	"errors"
	"sync"

	"github.com/Daneliya/prompt-next/internal/fileio"
	"github.com/Daneliya/prompt-next/internal/library"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const githubURL = "https://github.com/Daneliya"

type PromptService struct {
	*library.Service
	Version      string
	importMu     sync.Mutex
	chosenImport string
}

func (s *PromptService) OpenDataFolder() error {
	return application.Get().Env.OpenFileManager(library.DataDirectory(s.Service), false)
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
