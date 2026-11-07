package desktop

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/Daneliya/prompt-next/internal/fileio"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func RememberWindow(app *application.App, window *application.WebviewWindow, dir string) {
	path := filepath.Join(dir, "window.json")
	var once sync.Once
	window.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		once.Do(func() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return
			}
			var saved application.Rect
			if json.Unmarshal(raw, &saved) != nil || saved.Width < 860 || saved.Height < 560 {
				return
			}
			if bounds, ok := fitWindow(saved, app.Screen.GetAll()); ok {
				window.SetBounds(bounds)
			}
		})
	})
	save := func(*application.WindowEvent) {
		if window.IsMaximised() || window.IsMinimised() || window.IsFullscreen() {
			return
		}
		bounds := window.Bounds()
		if bounds.Width < 860 || bounds.Height < 560 {
			return
		}
		raw, err := json.Marshal(bounds)
		if err == nil {
			err = fileio.WriteAtomic(path, raw)
		}
		if err != nil {
			log.Printf("保存窗口位置失败：%v", err)
		}
	}
	window.RegisterHook(events.Common.WindowClosing, save)
	app.OnShutdown(func() { save(nil) })
}

func fitWindow(saved application.Rect, screens []*application.Screen) (application.Rect, bool) {
	var target application.Rect
	for _, screen := range screens {
		work := screen.WorkArea
		if target.Width == 0 || screen.IsPrimary {
			target = work
		}
		if saved.X >= work.X && saved.X < work.X+work.Width &&
			saved.Y >= work.Y && saved.Y < work.Y+work.Height {
			target = work
			break
		}
	}
	if target.Width < 860 || target.Height < 560 {
		return application.Rect{}, false
	}
	saved.Width = min(saved.Width, target.Width)
	saved.Height = min(saved.Height, target.Height)
	saved.X = max(target.X, min(saved.X, target.X+target.Width-saved.Width))
	saved.Y = max(target.Y, min(saved.Y, target.Y+target.Height-saved.Height))
	return saved, true
}
