# Prompt Next Development Guide

[简体中文](开发文档.md) · English · [繁體中文](development.zh-TW.md)

## Product and requirements

Prompt Next is a local-first desktop prompt manager for AI artwork and personas. The artwork library is listed first. The target releases are macOS Apple Silicon and Windows x64.

| Area | Requirements and current behavior |
| --- | --- |
| Branding and home | Use the selected red, blue and yellow stacked-photo logo in the app, Dock and installer. The home icon is a house and the persona library icon is a fish. Navigation icons share one background color. The dashboard has a light banner, library counts, recent changes, storage usage, a 14-day image area chart, and category and model pie charts. |
| Browse and search | Cards are the default. The first image is the cover. Users can switch to a three-pane list and retain their view preference. Search titles, Chinese and English text, tags, categories, notes, negative prompts and models. Filter by category, favorite, tag or model, and sort by title, creation or update date. |
| Artwork prompts | Create and edit in a dialog. Fields include title, category, comma-separated tags, searchable multi-model selection in a three-column dropdown, Chinese and English prompts, images and timestamps. At least one language is required. Optional fields include notes, negative prompts and generation parameters. Category and tags each have their own row. |
| Persona prompts | A separate library for persona text, title, category, tags, author, description and timestamps. Legacy BigFish exports are supported. |
| Prompt actions | Edit, favorite, duplicate, delete, export a single Markdown file, and copy the available prompt variants separately. Double-brace variables can be filled and previewed before copying. Confirm deletion in Chinese and warn before discarding edits. Detail views have a bordered Close button. Item dates use local time and the format `YYYY-MM-DD HH:mm:ss`. |
| Images | Select multiple PNG, JPEG, GIF or WebP files. Preview at full size and reorder by dragging or using the arrows. Existing and new images can be mixed. The limits are 24 per prompt, 12 MB and 64 million pixels per image, and 40 MB of new images per save. |
| Preferences | Appearance, AI Models, Categories, Data Management and About are separate sections. Theme and view changes take effect immediately. Each library has built-in and custom categories. Users can add, rename or delete custom model options. Existing prompts retain their historical model names. Data Management opens the data directory and offers import and backup. About links to the author's GitHub. |
| Backups | Preview new, duplicate and invalid records. Standard imports skip duplicates. A complete backup can replace both libraries after confirmation. The app reads older persona JSON and versions 1–3 of its own backup format. |

Image trends count files by local save date. Duplicate references count once and files written during restore count on the restore date. The model chart counts every assigned model. The UI uses compact spacing, small corners and distinct focus, hover, pressed and selected states, inspired by Tiny RDM.

## Architecture and data

| Path | Responsibility |
| --- | --- |
| `main.go` | Wails entry point, service registration and embedded assets |
| `internal/desktop/` | Wails calls, native dialogs, clipboard and file manager |
| `internal/library/` | Prompt model, SQLite, images, backups, settings and metrics |
| `internal/fileio/` | Shared atomic file writes |
| `frontend/src/` | Vue UI, charts and themes |
| `frontend/bindings/` | Generated TypeScript bridge |
| `build/` | Platform configuration and packaging scripts |

The Go module is `github.com/Daneliya/prompt-next`. After changing a service signature, regenerate the bridge with `wails3 generate bindings -clean=true -ts -i` or run a Wails build. The front end uses Vue 3, TypeScript, Vite and Naive UI.

The OS-specific `prompt-next` data folder holds `prompts.db`, per-prompt originals in `images/<number>/`, and generated covers in `thumbnails/<number>/`. SQLite uses WAL and a five-second busy timeout. Startup migrates legacy custom models into SQLite settings. JSON backup v3 includes both libraries, referenced images, theme, layout, custom categories and models, and variable presets. Restoring v1 or v2 preserves current settings. Replace restore first writes a safety snapshot under `backups/` and stops if the snapshot fails. JSON archives are limited to 128 MB. Quit the app before copying the entire directory for larger libraries. The repository contains synthetic fixtures only.

## Build and verification

Run `wails3 dev` for development. Run `go test ./...` after the front-end build has generated its assets. Package macOS with `wails3 task darwin:package:dmg` and Windows with `wails3 task windows:package ARCH=amd64` on a host with NSIS. Packages appear in `bin/`. Before publishing a release, test installation, WebView startup, backups and data preservation on upgrade on each target OS. Complete platform signing and notarization as appropriate.

[Hands-on tutorial in Simplified Chinese](开发教程.md) · [GitHub publishing notes](GitHub发布信息.md)

## Workspace and checks

Data Management previews each orphan image with its filename and size. Cleanup rechecks references and protects attachments in the 24-hour undo window. Use multi-select to favorite, recategorize, delete or export entries. Custom categories support rename and removal. Tags show usage counts and support merging and grouped browsing. Template values can be saved as named presets and rendered fields copied together. Native New and Search commands use Command on macOS and Ctrl on Windows. Normal window geometry is saved locally.

Run `wails3 task lint` and `wails3 task test` in the project directory. CI runs on macOS. The About dialog and macOS/Windows packaging read `info.version` from `build/config.yml`. Mobile templates are excluded.
