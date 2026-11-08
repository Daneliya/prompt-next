package desktop

import (
	"os"
	"os/exec"
	"strings"

	"github.com/Daneliya/prompt-next/internal/library"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func (s *PromptService) ChooseDataFolder() (string, error) {
	return application.Get().Dialog.OpenFile().
		SetTitle(localizedLabel(ReadLanguage(), "选择数据存放文件夹", "Choose Data Folder")).
		SetDirectory(library.DataDirectory(s.Service)).
		CanChooseFiles(false).CanChooseDirectories(true).
		PromptForSingleSelection()
}

func (s *PromptService) ChangeDataFolder(dir string) (bool, error) {
	return s.updateDataFolder(func() (bool, error) {
		return library.SwitchDataDirectory(s.Service, dir, launchReplacement)
	})
}

func (s *PromptService) ResetDataFolder() (bool, error) {
	return s.updateDataFolder(func() (bool, error) {
		return library.ResetDataDirectory(launchReplacement)
	})
}

func (s *PromptService) updateDataFolder(update func() (bool, error)) (bool, error) {
	s.locationMu.Lock()
	defer s.locationMu.Unlock()
	switched, err := update()
	if switched {
		go application.Get().Quit()
	}
	return switched, err
}

func launchReplacement() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(executable, os.Args[1:]...)
	// Use the embedded frontend once the development watcher exits.
	cmd.Env = []string{}
	for _, variable := range os.Environ() {
		if !strings.HasPrefix(variable, "FRONTEND_DEVSERVER_URL=") {
			cmd.Env = append(cmd.Env, variable)
		}
	}
	detachReplacement(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	return nil
}
