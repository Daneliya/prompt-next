<p align="center"><img src="build/appicon.png" alt="Prompt Next" width="108" /></p>

# Prompt Next

**以本機為主的桌面提示詞管理工具。** 在同一個工作台整理文字、AI 繪畫、影片與人設提示詞。

[English](README.en.md) · [简体中文](README.md) · **繁體中文**

[![Go 1.26+](https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Wails v3](https://img.shields.io/badge/Wails-v3-DF0000?style=flat-square)](https://wails.io/)
[![Vue 3](https://img.shields.io/badge/Vue-3-42B883?style=flat-square&logo=vuedotjs&logoColor=white)](https://vuejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Naive UI](https://img.shields.io/badge/Naive%20UI-18A058?style=flat-square)](https://www.naiveui.com/)
[![Vite](https://img.shields.io/badge/Vite-646CFF?style=flat-square&logo=vite&logoColor=white)](https://vite.dev/)
[![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://sqlite.org/)
[![MIT License](https://img.shields.io/badge/License-MIT-357A38?style=flat-square)](LICENSE)

## 工作台預覽

![Prompt Next](doc/assets/home.png)

## 功能

| 功能 | 說明 |
| --- | --- |
| 四個獨立提示詞庫 | 支援文字、繪畫、影片和人設，可整理分類、標籤、原作者和時間。繪畫和影片也支援中英文提示詞、負向提示詞、參數和多個模型。 |
| 圖片和影片預覽 | 效果圖可排序，第一張作為封面。可另附參考圖片，並在彈窗縮放、查看檔案資訊或下載原圖。影片條目可加入多段 MP4 或 WebM，在彈窗播放。 |
| 搜尋和整理 | 可切換卡片、清單和瀑布流，自訂顯示欄位，以標題、正文、編號搜尋，並用合集歸納提示詞變體。 |
| 複用和復原 | 支援範本變數預設、複製提示詞、匯出 Markdown 和批次操作。可在 24 小時內從資源回收筒復原，或永久刪除。 |
| 本機工作台 | 查看近 14 天新增提示詞趨勢、最近更新和儲存用量。可切換語言、主題和資料夾，備份附件與設定。覆蓋還原前會建立安全快照。 |

## 安裝

| 平台 | 安裝方式 |
| --- | --- |
| macOS Apple Silicon | [GitHub Releases](https://github.com/Daneliya/prompt-next/releases) · 開啟 DMG，將 Prompt Next 拖入「應用程式」。 |
| Windows x64 | [GitHub Releases](https://github.com/Daneliya/prompt-next/releases) · 執行 EXE 安裝程式。應用程式需要 WebView2。 |

## 專案結構

```text
prompt-next/
├── main.go              # Wails 應用程式入口，建立視窗、註冊桌面服務並嵌入前端資源
├── frontend/            # Vue 3 和 TypeScript 前端
│   ├── src/             # 工作台、提示詞庫、編輯器和設定介面
│   ├── bindings/        # Wails 產生的 TypeScript 呼叫繫結
│   └── public/
│       └── icons/       # 應用程式及提示詞庫圖示
├── internal/            # Go 後端
│   ├── desktop/         # 檔案選擇、剪貼簿及其他桌面介面
│   ├── library/         # SQLite、提示詞、附件、備份及繪畫和影片模型設定
│   └── fileio/          # 共用的原子檔案寫入
├── build/               # 平台封裝設定、建置指令碼和應用程式圖示
└── doc/                 # 開發文件、教學、發行資訊和螢幕擷取畫面
```

## 文件

| 文件 | 內容 |
| --- | --- |
| [簡體中文開發文件](doc/开发文档.md) | 需求、架構、資料和封裝 |
| [簡體中文開發教學](doc/开发教程.md) | 環境安裝和動手練習 |

採用 [MIT 授權](LICENSE)。桌面建置腳本源自 Wails 範本。 [Wails 授權聲明](licenses/Wails-MIT.txt)
