package library

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Daneliya/prompt-next/internal/fileio"
)

const MaxVideoBytes = 64 << 20
const MaxPromptVideos = 12
const MaxVideoUploadBytes = 192 << 20
const videoRoute = "/prompt-videos/"
const videoPosterRoute = "/prompt-video-posters/"

var videoName = regexp.MustCompile(`^[0-9a-f]{64}\.(mp4|webm)$`)
var videoKey = regexp.MustCompile(`^[1-9][0-9]{0,18}/[0-9a-f]{64}\.(mp4|webm)$`)

func checkedVideo(data []byte, extension string) (string, error) {
	if len(data) == 0 || len(data) > MaxVideoBytes {
		return "", errors.New("单个视频须在 64 MB 以内")
	}
	valid := false
	switch extension {
	case ".mp4":
		valid = len(data) >= 12 && string(data[4:8]) == "ftyp"
	case ".webm":
		valid = bytes.HasPrefix(data, []byte{0x1a, 0x45, 0xdf, 0xa3}) &&
			bytes.Contains(data[:min(len(data), 4096)], []byte("webm"))
	}
	if !valid {
		return "", errors.New("请选择有效的 MP4 或 WebM 视频")
	}
	return fmt.Sprintf("%x%s", sha256.Sum256(data), extension), nil
}

func readVideoFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > MaxVideoBytes {
		return nil, errors.New("请选择 64 MB 以内的本地视频文件")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxVideoBytes+1))
	if len(data) > MaxVideoBytes {
		return nil, errors.New("单个视频须在 64 MB 以内")
	}
	return data, err
}

func (s *promptStore) videoPath(key string) (string, error) {
	if !videoKey.MatchString(key) {
		return "", errors.New("视频路径无效")
	}
	return scopedMediaPath(filepath.Join(s.dir, "videos"), key)
}

func (s *promptStore) serveVideo(w http.ResponseWriter, r *http.Request, key string, poster bool) {
	path, err := s.videoPath(key)
	if err != nil || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		http.NotFound(w, r)
		return
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if poster {
		path, err = s.videoPoster(key, path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
	} else {
		w.Header().Set("Content-Type", "video/"+strings.TrimPrefix(filepath.Ext(path), "."))
	}
	http.ServeFile(w, r, path)
}

func (s *promptStore) videoPoster(key, source string) (string, error) {
	path := s.thumbnailPath(key)
	s.thumbnailMu.Lock()
	defer s.thumbnailMu.Unlock()
	if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() {
		return path, nil
	}
	data, err := decodeVideoPoster(source)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	return path, fileio.WriteAtomic(path, data)
}

func (s *promptStore) writeVideo(key string, data []byte) error {
	path, err := s.videoPath(key)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("视频路径不是普通文件")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return fileio.WriteAtomic(path, data)
}

func SaveVideoPrompt(s *Service, prompt Prompt, attachments []ImageInput, sourceNumber int64, paths []string) (Prompt, error) {
	if len(prompt.videoNames())+len(paths) > MaxPromptVideos {
		return Prompt{}, errors.New("每条视频提示词最多可附加 12 段视频")
	}
	prompt.Videos = append([]string{}, prompt.videoNames()...)
	files := make(map[string][]byte)
	total := 0
	for _, path := range paths {
		data, err := readVideoFile(path)
		if err != nil {
			return Prompt{}, err
		}
		name, err := checkedVideo(data, strings.ToLower(filepath.Ext(path)))
		if err != nil {
			return Prompt{}, err
		}
		if _, exists := files[name]; !exists {
			total += len(data)
			if total > MaxVideoUploadBytes {
				return Prompt{}, errors.New("每次添加的视频合计须在 192 MB 以内")
			}
			files[name] = data
		}
		prompt.Videos = append(prompt.Videos, name)
	}
	return s.saveOrderedMedia(prompt, attachments, sourceNumber, files)
}

func (s *promptStore) commitWithVideos(tx *sql.Tx, prompt Prompt, sourceNumber int64, files map[string][]byte) (err error) {
	var created []string
	defer func() {
		if err != nil {
			for _, path := range created {
				_ = os.Remove(path)
			}
		}
	}()
	for _, name := range prompt.Videos {
		key := scopedImage(prompt.Number, name)
		path, err := s.videoPath(key)
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err == nil && info.Mode().IsRegular() {
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return errors.New("视频路径无法使用")
		}
		data := files[name]
		if data == nil && sourceNumber > 0 && sourceNumber != prompt.Number {
			source, err := s.videoPath(scopedImage(sourceNumber, name))
			if err != nil {
				return err
			}
			data, err = readVideoFile(source)
			if err != nil {
				return err
			}
		}
		if data == nil {
			return errors.New("视频文件找不到，请重新选择")
		}
		if err := s.writeVideo(key, data); err != nil {
			return err
		}
		created = append(created, path)
	}
	return tx.Commit()
}

type VideoDetail struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

func (s *Service) VideoDetails(number int64, names []string) []VideoDetail {
	details := []VideoDetail{}
	for _, name := range names {
		path, err := s.store.videoPath(scopedImage(number, name))
		if err != nil {
			continue
		}
		if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() {
			details = append(details, VideoDetail{Name: name, Bytes: info.Size()})
		}
	}
	return details
}
