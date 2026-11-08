package library

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Daneliya/prompt-next/internal/fileio"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const maxImageSize = 12 << 20
const MaxImageUploadBytes = 80 << 20
const imageRoute = "/prompt-images/"
const thumbnailRoute = "/prompt-thumbnails/"

var imageName = regexp.MustCompile(`^[0-9a-f]{64}\.(png|jpg|gif|webp)$`)

var imageFolder = regexp.MustCompile(`^[1-9][0-9]{0,18}$`)
var imageKey = regexp.MustCompile(`^(?:[1-9][0-9]{0,18}/)?[0-9a-f]{64}\.(png|jpg|gif|webp)$`)

func scopedImage(number int64, name string) string {
	return strconv.FormatInt(number, 10) + "/" + name
}

func (s *promptStore) imagesDir() string {
	return filepath.Join(s.dir, "images")
}

func checkedImage(data []byte) (string, error) {
	if len(data) == 0 || len(data) > maxImageSize {
		return "", errors.New("单张图片大小须在 12 MB 以内")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 64_000_000 {
		return "", errors.New("无法读取图片，请使用 PNG、JPEG、GIF 或 WebP，且不超过 6400 万像素")
	}
	if format == "jpeg" {
		format = "jpg"
	}
	return fmt.Sprintf("%x.%s", sha256.Sum256(data), format), nil
}

func (s *promptStore) imageAssets(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if key, matches := strings.CutPrefix(r.URL.Path, videoPosterRoute); matches {
			s.serveVideo(w, r, key, true)
			return
		}
		if key, matches := strings.CutPrefix(r.URL.Path, videoRoute); matches {
			s.serveVideo(w, r, key, false)
			return
		}
		name, matches := strings.CutPrefix(r.URL.Path, imageRoute)
		thumbnail := false
		if !matches {
			name, thumbnail = strings.CutPrefix(r.URL.Path, thumbnailRoute)
			if !thumbnail {
				next.ServeHTTP(w, r)
				return
			}
		}
		if !imageKey.MatchString(name) || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		path, err := s.imagePath(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		if thumbnail {
			thumb, err := s.thumbnail(name, path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			path = thumb
		}
		http.ServeFile(w, r, path)
	})
}

func (s *promptStore) thumbnailPath(name string) string {
	return filepath.Join(s.dir, "thumbnails", name+".jpg")
}

func (s *promptStore) thumbnail(name, original string) (string, error) {
	path := s.thumbnailPath(name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	s.thumbnailMu.Lock()
	defer s.thumbnailMu.Unlock()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	file, err := os.Open(original)
	if err != nil {
		return "", err
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 64_000_000 {
		return "", errors.New("缩略图来源无效")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	source, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}
	width, height := source.Bounds().Dx(), source.Bounds().Dy()
	const limit = 400
	if width > limit || height > limit {
		if width >= height {
			width, height = limit, max(1, height*limit/width)
		} else {
			width, height = max(1, width*limit/height), limit
		}
	}
	thumb := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(thumb, thumb.Bounds(), image.White, image.Point{}, draw.Src)
	draw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), source, source.Bounds(), draw.Over, nil)
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, thumb, &jpeg.Options{Quality: 78}); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	return path, fileio.WriteAtomic(path, encoded.Bytes())
}

func (s *promptStore) imagePath(key string) (string, error) {
	if !imageKey.MatchString(key) {
		return "", errors.New("图片路径无效")
	}
	return scopedMediaPath(s.imagesDir(), key)
}

func scopedMediaPath(root, key string) (string, error) {
	path := filepath.Join(root, key)
	if strings.Contains(key, "/") {
		info, err := os.Lstat(filepath.Dir(path))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if err == nil && !info.IsDir() {
			return "", errors.New("媒体目录无效")
		}
	}
	return path, nil
}

func (s *promptStore) writeImage(key string, data []byte) error {
	path, err := s.imagePath(key)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("图片路径不是普通文件")
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

func (s *promptStore) copyImage(source, target string, original os.FileInfo) error {
	sourcePath, err := s.imagePath(source)
	if err != nil {
		return err
	}
	targetPath, err := s.imagePath(target)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(targetPath); err == nil && info.Mode().IsRegular() {
		return nil
	}
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > maxImageSize {
		return errors.New("图片来源无效")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if err := s.writeImage(target, data); err != nil {
		return err
	}
	if original != nil {
		return os.Chtimes(targetPath, original.ModTime(), original.ModTime())
	}
	return nil
}

func DecodeImageUpload(encoded string) ([]byte, error) {
	if len(encoded) > base64.StdEncoding.EncodedLen(maxImageSize) {
		return nil, errors.New("单张图片大小须在 12 MB 以内")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("图片读取失败")
	}
	if len(data) > maxImageSize {
		return nil, errors.New("单张图片大小须在 12 MB 以内")
	}
	return data, nil
}

type ImageInput struct {
	Name         string `json:"name"`
	Data         string `json:"data"`
	SourceNumber int64  `json:"sourceNumber"`
	Reference    bool   `json:"reference"`
	UploadID     string `json:"uploadId,omitempty"`
	UploadBytes  []byte `json:"-"`
}

func (s *Service) SaveWithImages(prompt Prompt, uploads []string) (Prompt, error) {
	attachments := make([]ImageInput, 0, len(prompt.Images)+len(prompt.ReferenceImages)+len(uploads))
	for _, name := range prompt.Images {
		attachments = append(attachments, ImageInput{Name: name})
	}
	for _, name := range prompt.ReferenceImages {
		attachments = append(attachments, ImageInput{Name: name, Reference: true})
	}
	for _, data := range uploads {
		attachments = append(attachments, ImageInput{Data: data})
	}
	return s.SaveOrderedImages(prompt, attachments)
}

func (s *Service) SaveOrderedImages(prompt Prompt, attachments []ImageInput) (Prompt, error) {
	return s.saveOrderedMedia(prompt, attachments, 0, nil)
}

func (s *Service) saveOrderedMedia(prompt Prompt, attachments []ImageInput, sourceNumber int64, videoData map[string][]byte) (Prompt, error) {
	prompt.Images, prompt.ReferenceImages = nil, nil
	if err := prompt.validate(); err != nil {
		return Prompt{}, err
	}
	if len(attachments) > 24 || (!supportsMedia(prompt.Kind) && len(attachments) != 0) {
		return Prompt{}, errors.New("绘画和视频条目最多可附加 24 张图片")
	}
	files := make(map[string][]byte)
	sources := make(map[string]int64)
	total := 0
	for _, attachment := range attachments {
		names := &prompt.Images
		if attachment.Reference {
			names = &prompt.ReferenceImages
		}
		if attachment.Name != "" {
			if attachment.Data != "" || attachment.UploadBytes != nil || !imageName.MatchString(attachment.Name) || attachment.SourceNumber < 0 || attachment.SourceNumber > maxPromptNumber {
				return Prompt{}, errors.New("图片引用无效")
			}
			sources[attachment.Name] = attachment.SourceNumber
			*names = append(*names, attachment.Name)
			continue
		}
		data := attachment.UploadBytes
		if data == nil {
			var err error
			data, err = DecodeImageUpload(attachment.Data)
			if err != nil {
				return Prompt{}, err
			}
		}
		total += len(data)
		if total > MaxImageUploadBytes {
			return Prompt{}, errors.New("每次保存的新图片合计须在 80 MB 以内")
		}
		name, err := checkedImage(data)
		if err != nil {
			return Prompt{}, err
		}
		files[name] = data
		*names = append(*names, name)
	}
	tx, err := s.store.db.Begin()
	if err != nil {
		return Prompt{}, err
	}
	defer tx.Rollback()
	saved, err := savePrompt(tx, prompt)
	if err != nil {
		return Prompt{}, err
	}
	for _, name := range cleanTags(saved.imageNames()) {
		key := scopedImage(saved.Number, name)
		if data, ok := files[name]; ok {
			if err := s.store.writeImage(key, data); err != nil {
				return Prompt{}, err
			}
			continue
		}
		if source := sources[name]; source > 0 && source != saved.Number {
			if err := s.store.copyImage(scopedImage(source, name), key, nil); err != nil {
				return Prompt{}, fmt.Errorf("无法复制参考图片：%w", err)
			}
		}
		path, err := s.store.imagePath(key)
		if err != nil {
			return Prompt{}, err
		}
		if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() {
			return Prompt{}, errors.New("附件找不到，请移除后重新添加")
		}
	}
	if err := s.store.commitWithVideos(tx, saved, sourceNumber, videoData); err != nil {
		return Prompt{}, err
	}
	return saved, nil
}

// ImageForExport returns original bytes from a saved image or an unsaved upload.
func ImageForExport(s *Service, number int64, name, upload string) ([]byte, string, error) {
	var data []byte
	if upload != "" {
		if len(upload) > base64.StdEncoding.EncodedLen(maxImageSize) {
			return nil, "", errors.New("单张图片大小须在 12 MB 以内")
		}
		var err error
		data, err = base64.StdEncoding.DecodeString(upload)
		if err != nil {
			return nil, "", err
		}
	} else {
		if number <= 0 || !imageName.MatchString(name) {
			return nil, "", errors.New("图片引用无效")
		}
		path, err := s.store.imagePath(scopedImage(number, name))
		if err != nil {
			return nil, "", err
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxImageSize {
			return nil, "", errors.New("无法读取原图")
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, "", err
		}
		defer file.Close()
		data, err = io.ReadAll(io.LimitReader(file, maxImageSize+1))
		if err != nil {
			return nil, "", err
		}
	}
	original, err := checkedImage(data)
	return data, filepath.Ext(original), err
}

type ImageDetail struct {
	Name   string `json:"name"`
	Bytes  int64  `json:"bytes"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
}

func (s *Service) ImageDetails(number int64, names []string) []ImageDetail {
	details := []ImageDetail{}
	for _, name := range cleanTags(names) {
		if !imageName.MatchString(name) {
			continue
		}
		path, err := s.store.imagePath(scopedImage(number, name))
		if err != nil {
			continue
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		config, format, err := image.DecodeConfig(file)
		file.Close()
		if err == nil {
			details = append(details, ImageDetail{
				Name: name, Bytes: info.Size(), Width: config.Width, Height: config.Height, Format: format,
			})
		}
	}
	return details
}
