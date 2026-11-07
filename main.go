package main

import (
	"embed"
	"log"
	"regexp"
	"runtime"

	"github.com/Daneliya/prompt-next/internal/desktop"
	"github.com/Daneliya/prompt-next/internal/library"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/config.yml
var buildConfig []byte

var versionLine = regexp.MustCompile(`(?m)^  version: "(\d+\.\d+\.\d+)"`)

func appVersion() string {
	if match := versionLine.FindSubmatch(buildConfig); len(match) == 2 {
		return string(match[1])
	}
	return "开发版"
}

func main() {
	prompts, err := library.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer library.Close(prompts)

	app := application.New(application.Options{
		Name:        "Prompt Next",
		Icon:        appIcon,
		Description: "离线管理人设和 AI 绘画提示词",
		Services:    []application.Service{application.NewService(&desktop.PromptService{Service: prompts, Version: appVersion()})},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets), Middleware: library.ImageAssets(prompts)},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Prompt Next", Width: 1280, Height: 800,
		InitialPosition: application.WindowCentered,
		MinWidth:        860, MinHeight: 560, URL: "/",
	})
	desktop.RememberWindow(app, window, library.DataDirectory(prompts))
	menu := app.Menu.New()
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
	}
	fileMenu := menu.AddSubmenu("文件")
	fileMenu.Add("新建提示词").SetAccelerator("CmdOrCtrl+N").OnClick(func(*application.Context) {
		window.EmitEvent("prompt-next:new")
	})
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.EditMenu)
	}
	searchMenu := menu.AddSubmenu("搜索")
	searchMenu.Add("聚焦搜索框").SetAccelerator("CmdOrCtrl+F").OnClick(func(*application.Context) {
		window.EmitEvent("prompt-next:search")
	})
	app.Menu.Set(menu)
	window.SetMenu(menu)
	if err := app.Run(); err != nil {
		log.Print(err)
	}
}
