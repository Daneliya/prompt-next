package fileio

import (
	"errors"
	"os"
	"path/filepath"
)

func WriteAtomic(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".prompt-write-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
