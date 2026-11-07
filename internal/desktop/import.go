package desktop

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/Daneliya/prompt-next/internal/library"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ChosenBackup struct {
	Name    string                `json:"name"`
	Summary library.ImportSummary `json:"summary"`
}

func (s *PromptService) ChooseBackup() (ChosenBackup, error) {
	path, err := application.Get().Dialog.OpenFile().SetTitle("选择要导入的提示词备份").AddFilter("JSON 备份", "*.json").PromptForSingleSelection()
	if err != nil || path == "" {
		return ChosenBackup{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return ChosenBackup{}, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, library.MaxBackupBytes+1))
	if err != nil {
		return ChosenBackup{}, err
	}
	if len(raw) > library.MaxBackupBytes {
		return ChosenBackup{}, errors.New("备份文件不能超过 128 MB")
	}
	text := string(raw)
	summary, err := s.PreviewImport(text)
	if err != nil {
		return ChosenBackup{}, err
	}
	s.importMu.Lock()
	defer s.importMu.Unlock()
	s.chosenImport = text
	return ChosenBackup{Name: filepath.Base(path), Summary: summary}, nil
}

func (s *PromptService) ClearChosenBackup() {
	s.importMu.Lock()
	defer s.importMu.Unlock()
	s.chosenImport = ""
}

func (s *PromptService) ImportChosenBackup(replace bool) (library.ImportSummary, error) {
	s.importMu.Lock()
	defer s.importMu.Unlock()
	if s.chosenImport == "" {
		return library.ImportSummary{}, errors.New("请先选择备份文件")
	}
	summary, err := s.ImportPrompts(s.chosenImport, replace)
	if err == nil {
		s.chosenImport = ""
	}
	return summary, err
}
