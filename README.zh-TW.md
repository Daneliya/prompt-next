<p align="center"><img src="build/appicon.png" alt="Prompt Next" width="108" /></p>

# Prompt Next

**以本機為主的 AI 繪畫與人設提示詞管理工具。** 在同一個桌面工作台整理中英文提示詞、AI 模型和參考圖，隨時搜尋、預覽、複製及備份。

[简体中文](README.md) · [English](README.en.md) · 繁體中文

[![Go 1.26+](https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Wails v3](https://img.shields.io/badge/Wails-v3-DF0000?style=flat-square)](https://wails.io/)
[![Vue 3](https://img.shields.io/badge/Vue-3-42B883?style=flat-square&logo=vuedotjs&logoColor=white)](https://vuejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Naive UI](https://img.shields.io/badge/Naive%20UI-18A058?style=flat-square)](https://www.naiveui.com/)
[![Vite](https://img.shields.io/badge/Vite-646CFF?style=flat-square&logo=vite&logoColor=white)](https://vite.dev/)
[![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://sqlite.org/)
[![MIT License](https://img.shields.io/badge/License-MIT-357A38?style=flat-square)](LICENSE)

## 功能

- **兩套提示詞庫**。優先顯示 AI 繪畫庫，也保留大肥魚人設庫。可在卡片和三欄清單間切換，支援搜尋、分類、標籤、收藏、篩選和排序，也可批次整理並在 24 小時內復原刪除。
- **繪畫資料管理**。可儲存中英文提示詞、多個模型、負向提示詞與生成參數。參考圖支援多選、放大和拖曳排序，第一張作為封面。
- **複製與移轉**。支援範本變數預設、單筆 Markdown 或選中項目匯出及舊人設匯入。完整備份包含圖片與設定，覆蓋前會自動儲存快照。
- **本機工作台**。查看筆數、最近更新、儲存用量及圖表。可切換淺色、深色與跟隨系統，或在設定中直接開啟資料夾、預覽並清理未引用圖片。

## 安裝與開發

安裝檔目標平台為 **macOS Apple Silicon（DMG）** 與 **Windows x64（EXE）**。有發行版本時，可在 GitHub Releases 下載，也可在本機建置。

開發需要 Go 1.26+、Node.js 24、npm 和 Wails v3。macOS 需要 Xcode Command Line Tools。Windows 執行時使用 WebView2，封裝安裝程式需要 NSIS。

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
wails3 doctor
wails3 dev
```

若找不到 `wails3`，請將 Go 的可執行檔目錄加入 `PATH`。首次啟動會安裝前端依賴並產生呼叫繫結。

```bash
wails3 task lint
wails3 task test
wails3 build
```

| 安裝檔 | 指令 |
| --- | --- |
| macOS DMG | `wails3 task darwin:package:dmg` |
| Windows x64 NSIS | `wails3 task windows:package ARCH=amd64` |

在具備所需工具鏈的系統執行，產物位於 `bin/`。可由 **設定 → 資料管理 → 開啟資料夾** 找到本機資料。

## 文件

- [開發文件](doc/development.zh-TW.md) · 需求、架構、資料和封裝
- [簡體中文開發教學](doc/开发教程.md) · 環境安裝和動手練習

採用 [MIT 授權](LICENSE)。桌面建置腳本源自 Wails 範本。 [Wails 授權聲明](licenses/Wails-MIT.txt)
