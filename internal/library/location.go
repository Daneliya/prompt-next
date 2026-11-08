package library

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Daneliya/prompt-next/internal/fileio"
)

const locationFile = "prompt-next-location.json"

type dataLocation struct {
	Directory string `json:"directory"`
}

func openWithLocation(base string) (*Service, error) {
	raw, err := os.ReadFile(filepath.Join(base, locationFile))
	var warning string
	if err == nil {
		var location dataLocation
		if json.Unmarshal(raw, &location) == nil && filepath.IsAbs(location.Directory) {
			store, err := openSelectedStore(location.Directory, false)
			if err == nil {
				return &Service{store: store}, nil
			}
			warning = fmt.Sprintf("无法打开已选数据目录 %s，当前使用默认目录。", location.Directory)
		} else {
			warning = "数据目录设置无效，当前使用默认目录。"
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		warning = "无法读取数据目录设置，当前使用默认目录。"
	}

	dir, err := resolveDataDirectory(base)
	if err != nil {
		return nil, err
	}
	store, err := openStoreAt(dir)
	if err != nil {
		return nil, err
	}
	return &Service{store: store, locationWarning: warning}, nil
}

func openSelectedStore(dir string, allowEmpty bool) (*promptStore, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(filepath.Join(dir, "prompts.db"))
	if errors.Is(err, os.ErrNotExist) {
		if !allowEmpty {
			return nil, errors.New("找不到提示词数据库")
		}
		for _, entry := range files {
			if entry.Name() != ".DS_Store" && entry.Name() != "desktop.ini" {
				return nil, errors.New("请选择空文件夹或已有的 Prompt Next 数据目录")
			}
		}
		return openStoreAt(dir)
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("提示词数据库路径不是普通文件")
	}

	db, err := sql.Open("sqlite", storeURL(dir, "mode=ro&_pragma=busy_timeout%285000%29"))
	if err != nil {
		return nil, err
	}
	var tables int
	err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name IN ('prompts', 'settings')").Scan(&tables)
	db.Close()
	if err != nil {
		return nil, err
	}
	if tables != 2 {
		return nil, errors.New("该文件夹里的数据库不属于 Prompt Next")
	}
	return openStoreAt(dir)
}

func SwitchDataDirectory(s *Service, dir string, relaunch func() error) (bool, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return false, err
	}
	return switchDataDirectory(s, base, dir, relaunch)
}

func ResetDataDirectory(relaunch func() error) (bool, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return false, err
	}
	return resetDataDirectory(base, relaunch)
}

func resetDataDirectory(base string, relaunch func() error) (bool, error) {
	path := filepath.Join(base, locationFile)
	previous, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	dir, err := resolveDataDirectory(base)
	if err != nil {
		return false, err
	}
	store, err := openStoreAt(dir)
	if err != nil {
		return false, fmt.Errorf("无法打开默认数据目录：%w", err)
	}
	if err := store.db.Close(); err != nil {
		return false, err
	}
	if err := os.Remove(path); err != nil {
		return false, err
	}
	if err := relaunch(); err != nil {
		return false, errors.Join(fmt.Errorf("无法重新启动应用：%w", err), fileio.WriteAtomic(path, previous))
	}
	return true, nil
}

func switchDataDirectory(s *Service, base, dir string, relaunch func() error) (bool, error) {
	if !filepath.IsAbs(dir) {
		return false, errors.New("请选择有效的文件夹")
	}
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return false, err
	}
	targetInfo, err := os.Stat(dir)
	if err != nil {
		return false, err
	}
	if currentInfo, err := os.Stat(s.store.dir); err == nil && os.SameFile(currentInfo, targetInfo) && s.locationWarning == "" {
		return false, nil
	}
	store, err := openSelectedStore(dir, true)
	if err != nil {
		return false, fmt.Errorf("无法打开所选数据目录：%w", err)
	}
	if err := store.db.Close(); err != nil {
		return false, err
	}

	path := filepath.Join(base, locationFile)
	previous, err := os.ReadFile(path)
	hadPrevious := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	raw, err := json.Marshal(dataLocation{Directory: dir})
	if err != nil {
		return false, err
	}
	if err := fileio.WriteAtomic(path, raw); err != nil {
		return false, err
	}
	if err := relaunch(); err != nil {
		var rollbackErr error
		if hadPrevious {
			rollbackErr = fileio.WriteAtomic(path, previous)
		} else {
			rollbackErr = os.Remove(path)
		}
		return false, errors.Join(fmt.Errorf("无法重新启动应用：%w", err), rollbackErr)
	}
	return true, nil
}
