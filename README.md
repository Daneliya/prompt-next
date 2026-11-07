<p align="center"><img src="build/appicon.png" alt="Prompt Next" width="108" /></p>

# Prompt Next

**本地优先的 AI 绘画与人设提示词管理器。** 在一个桌面工作台中整理中英文提示词、AI 模型和参考图，随时搜索、预览、复制或备份。

简体中文 · [English](README.en.md) · [繁體中文](README.zh-TW.md)

[![Go 1.26+](https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Wails v3](https://img.shields.io/badge/Wails-v3-DF0000?style=flat-square)](https://wails.io/)
[![Vue 3](https://img.shields.io/badge/Vue-3-42B883?style=flat-square&logo=vuedotjs&logoColor=white)](https://vuejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Naive UI](https://img.shields.io/badge/Naive%20UI-18A058?style=flat-square)](https://www.naiveui.com/)
[![Vite](https://img.shields.io/badge/Vite-646CFF?style=flat-square&logo=vite&logoColor=white)](https://vite.dev/)
[![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://sqlite.org/)
[![MIT License](https://img.shields.io/badge/License-MIT-357A38?style=flat-square)](LICENSE)

## 功能

- **两套提示词库**。AI 绘画库优先，同时保留大肥鱼人设库。默认卡片，可切换三栏列表，支持搜索、分类、标签、收藏、筛选和排序，以及批量整理和删除撤销。
- **绘画资料整理**。保存中英文提示词、多个模型、负向提示词和生成参数。参考图可多选、放大与拖动排序，首图作为封面。
- **复制与迁移**。支持模板变量预设、单条 Markdown 和选中条目 JSON 导出、旧人设导入。完整备份包含引用图片与设置，覆盖恢复前自动保存快照。
- **本地工作台**。展示条目数量、最近更新、存储用量和图表。可切换浅色、深色与跟随系统，或在设置里打开数据文件夹、预览并清理未引用图片。

## 安装与开发

安装包目标为 **macOS Apple Silicon（DMG）** 和 **Windows x64（EXE）**。有发布版本时，可在仓库的 GitHub Releases 中下载。也可在本地构建。

开发环境需要 Go 1.26+、Node.js 24、npm 和 Wails v3。macOS 需要 Xcode Command Line Tools。Windows 运行需要 WebView2，构建安装器需要 NSIS。

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
wails3 doctor
wails3 dev
```

如果找不到 `wails3`，把 Go 的可执行文件目录加入 `PATH`。首次启动时，Wails 会处理前端依赖和调用绑定。安装与故障排查见 [开发教程](doc/开发教程.md)。

```bash
wails3 task lint
wails3 task test
wails3 build
```

| 安装包 | 命令 |
| --- | --- |
| macOS DMG | `wails3 task darwin:package:dmg` |
| Windows x64 NSIS | `wails3 task windows:package ARCH=amd64` |

在具备对应工具链的系统运行命令。产物在 `bin/`。应用数据可通过 **设置 → 数据管理 → 打开文件夹** 查看。

## 文档

- [开发文档](doc/开发文档.md) · 功能要求、工程分层、数据与发布
- [开发教程](doc/开发教程.md) · 环境安装和动手练习

项目采用 [MIT 许可证](LICENSE)。桌面构建脚本源自 Wails 模板。 [Wails 许可声明](licenses/Wails-MIT.txt)
