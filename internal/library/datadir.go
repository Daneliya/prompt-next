package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func resolveDataDirectory(base string) (string, error) {
	target := filepath.Join(base, "prompt-next")
	exists, err := dataDirectoryExists(target)
	if err != nil || exists {
		return target, err
	}

	var legacy string
	for _, name := range []string{"PromptManager", "prompt-manager"} {
		candidate := filepath.Join(base, name)
		exists, err := dataDirectoryExists(candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			continue
		}
		if legacy != "" {
			return "", fmt.Errorf("找到两个旧数据目录，无法自动迁移：%s、%s", legacy, candidate)
		}
		legacy = candidate
	}
	if legacy == "" {
		return target, os.MkdirAll(target, 0700)
	}
	if err := os.Rename(legacy, target); err != nil {
		// A concurrent launch may have moved the same source first.
		if errors.Is(err, os.ErrNotExist) {
			if exists, checkErr := dataDirectoryExists(target); exists && checkErr == nil {
				return target, nil
			}
		}
		return "", fmt.Errorf("无法迁移旧数据目录 %s：%w", legacy, err)
	}
	return target, nil
}

func dataDirectoryExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("应用数据路径不是文件夹：%s", path)
	}
	return true, nil
}
