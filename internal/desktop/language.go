package desktop

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Daneliya/prompt-next/internal/fileio"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const languageFile = "prompt-next-language"

func ReadLanguage() string {
	base, err := os.UserConfigDir()
	if err == nil {
		data, err := os.ReadFile(filepath.Join(base, languageFile))
		if err == nil && string(data) == "en-US" {
			return "en-US"
		}
	}
	return "zh-CN"
}

func localizedLabel(language, chinese, english string) string {
	if language == "en-US" {
		return english
	}
	return chinese
}

func (s *PromptService) GetLanguage() string {
	return ReadLanguage()
}

func (s *PromptService) SetLanguage(language string) error {
	if language != "zh-CN" && language != "en-US" {
		return errors.New("Unsupported language")
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(base, 0700); err != nil {
		return err
	}
	if err := fileio.WriteAtomic(filepath.Join(base, languageFile), []byte(language)); err != nil {
		return err
	}
	if s.OnLanguageChange != nil {
		application.InvokeSync(func() { s.OnLanguageChange(language) })
	}
	return nil
}
